package recoverytransport

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func campaignSupportsDaggerheart(record storage.CampaignRecord) bool {
	systemID, ok := systembridge.NormalizeSystemID(record.System.String())
	return ok && systemID == systembridge.SystemIDDaggerheart
}

func requireDaggerheartSystem(record storage.CampaignRecord, unsupportedMessage string) error {
	if campaignSupportsDaggerheart(record) {
		return nil
	}
	return status.Error(codes.FailedPrecondition, unsupportedMessage)
}

func handleDomainError(ctx context.Context, err error) error {
	return grpcerror.HandleDomainErrorContext(ctx, err)
}

func ensureNoOpenSessionGate(ctx context.Context, store SessionGateStore, campaignID, sessionID string) error {
	if store == nil {
		return status.Error(codes.Internal, "session gate store is not configured")
	}
	if strings.TrimSpace(campaignID) == "" || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	gate, err := store.GetOpenSessionGate(ctx, campaignID, sessionID)
	if err == nil {
		return status.Errorf(codes.FailedPrecondition, "session gate is open: %s", gate.GateID)
	}
	return grpcerror.OptionalLookupErrorContext(ctx, err, "load session gate")
}

func resolveSeed(rng *commonv1.RngRequest, seedFunc func() (int64, error), resolve func(*commonv1.RngRequest, func() (int64, error), func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error)) (int64, error) {
	seed, _, _, err := resolve(rng, seedFunc, func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY })
	return seed, err
}

func countdownFromStorage(countdown projectionstore.DaggerheartCountdown) rules.Countdown {
	value := rules.Countdown{
		CampaignID:        countdown.CampaignID,
		ID:                countdown.CountdownID,
		Name:              countdown.Name,
		Tone:              countdown.Tone,
		AdvancementPolicy: countdown.AdvancementPolicy,
		StartingValue:     countdown.StartingValue,
		RemainingValue:    countdown.RemainingValue,
		LoopBehavior:      countdown.LoopBehavior,
		Status:            countdown.Status,
		LinkedCountdownID: countdown.LinkedCountdownID,
	}
	if value.AdvancementPolicy == "" {
		value.AdvancementPolicy = rules.CountdownAdvancementPolicyManual
	}
	if value.LoopBehavior == "" {
		value.LoopBehavior = rules.CountdownLoopBehaviorNone
	}
	if value.Status == "" {
		value.Status = rules.CountdownStatusActive
	}
	if countdown.StartingRollMin > 0 && countdown.StartingRollMax > 0 {
		value.StartingRoll = &rules.CountdownStartingRoll{
			Min:   countdown.StartingRollMin,
			Max:   countdown.StartingRollMax,
			Value: countdown.StartingRollValue,
		}
	}
	return value
}

func restTypeFromProto(t pb.DaggerheartRestType) (daggerheart.RestType, error) {
	switch t {
	case pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED:
		return daggerheart.RestTypeShort, fmt.Errorf("rest_type is required")
	case pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT:
		return daggerheart.RestTypeShort, nil
	case pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG:
		return daggerheart.RestTypeLong, nil
	default:
		return daggerheart.RestTypeShort, fmt.Errorf("rest_type %v is invalid", t)
	}
}

func downtimeSelectionFromProto(
	selection *pb.DaggerheartDowntimeSelection,
	resolve func(*commonv1.RngRequest, func() (int64, error), func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error),
	seedFunc func() (int64, error),
) (daggerheart.DowntimeSelection, error) {
	if selection == nil {
		return daggerheart.DowntimeSelection{}, fmt.Errorf("downtime move is required")
	}
	switch move := selection.GetMove().(type) {
	case *pb.DaggerheartDowntimeSelection_TendToWounds:
		seed, err := resolveSeed(move.TendToWounds.GetRng(), seedFunc, resolve)
		if err != nil {
			return daggerheart.DowntimeSelection{}, grpcerror.Internal("failed to resolve tend_to_wounds seed", err)
		}
		return daggerheart.DowntimeSelection{
			Move:              daggerheart.DowntimeMoveTendToWounds,
			TargetCharacterID: ids.CharacterID(strings.TrimSpace(move.TendToWounds.GetTargetCharacterId())),
			RollSeed:          &seed,
		}, nil
	case *pb.DaggerheartDowntimeSelection_ClearStress:
		seed, err := resolveSeed(move.ClearStress.GetRng(), seedFunc, resolve)
		if err != nil {
			return daggerheart.DowntimeSelection{}, grpcerror.Internal("failed to resolve clear_stress seed", err)
		}
		return daggerheart.DowntimeSelection{Move: daggerheart.DowntimeMoveClearStress, RollSeed: &seed}, nil
	case *pb.DaggerheartDowntimeSelection_RepairArmor:
		seed, err := resolveSeed(move.RepairArmor.GetRng(), seedFunc, resolve)
		if err != nil {
			return daggerheart.DowntimeSelection{}, grpcerror.Internal("failed to resolve repair_armor seed", err)
		}
		return daggerheart.DowntimeSelection{
			Move:              daggerheart.DowntimeMoveRepairArmor,
			TargetCharacterID: ids.CharacterID(strings.TrimSpace(move.RepairArmor.GetTargetCharacterId())),
			RollSeed:          &seed,
		}, nil
	case *pb.DaggerheartDowntimeSelection_Prepare:
		return daggerheart.DowntimeSelection{
			Move:    daggerheart.DowntimeMovePrepare,
			GroupID: strings.TrimSpace(move.Prepare.GetGroupId()),
		}, nil
	case *pb.DaggerheartDowntimeSelection_TendToAllWounds:
		return daggerheart.DowntimeSelection{
			Move:              daggerheart.DowntimeMoveTendToAllWounds,
			TargetCharacterID: ids.CharacterID(strings.TrimSpace(move.TendToAllWounds.GetTargetCharacterId())),
		}, nil
	case *pb.DaggerheartDowntimeSelection_ClearAllStress:
		return daggerheart.DowntimeSelection{Move: daggerheart.DowntimeMoveClearAllStress}, nil
	case *pb.DaggerheartDowntimeSelection_RepairAllArmor:
		return daggerheart.DowntimeSelection{
			Move:              daggerheart.DowntimeMoveRepairAllArmor,
			TargetCharacterID: ids.CharacterID(strings.TrimSpace(move.RepairAllArmor.GetTargetCharacterId())),
		}, nil
	case *pb.DaggerheartDowntimeSelection_WorkOnProject:
		return daggerheart.DowntimeSelection{
			Move:                daggerheart.DowntimeMoveWorkOnProject,
			CountdownID:         dhids.CountdownID(strings.TrimSpace(move.WorkOnProject.GetProjectCampaignCountdownId())),
			ProjectAdvanceMode:  projectAdvanceModeFromProto(move.WorkOnProject.GetAdvanceMode()),
			ProjectAdvanceDelta: int(move.WorkOnProject.GetAdvanceDelta()),
			ProjectReason:       strings.TrimSpace(move.WorkOnProject.GetReason()),
		}, nil
	default:
		return daggerheart.DowntimeSelection{}, fmt.Errorf("downtime move is required")
	}
}

func projectAdvanceModeFromProto(mode pb.DaggerheartProjectAdvanceMode) string {
	switch mode {
	case pb.DaggerheartProjectAdvanceMode_DAGGERHEART_PROJECT_ADVANCE_MODE_GM_SET_DELTA:
		return daggerheart.ProjectAdvanceModeGMSetDelta
	default:
		return daggerheart.ProjectAdvanceModeAuto
	}
}

func deathMoveFromProto(move pb.DaggerheartDeathMove) (string, error) {
	switch move {
	case pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED:
		return "", fmt.Errorf("death move is required")
	case pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY:
		return daggerheart.DeathMoveBlazeOfGlory, nil
	case pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH:
		return daggerheart.DeathMoveAvoidDeath, nil
	case pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL:
		return daggerheart.DeathMoveRiskItAll, nil
	default:
		return "", fmt.Errorf("death move %v is invalid", move)
	}
}

// DeathMoveToProto maps stored death-move strings into the public gRPC enum so
// root response shaping does not retain duplicate death-move helpers.
func DeathMoveToProto(move string) pb.DaggerheartDeathMove {
	switch move {
	case daggerheart.DeathMoveBlazeOfGlory:
		return pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY
	case daggerheart.DeathMoveAvoidDeath:
		return pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH
	case daggerheart.DeathMoveRiskItAll:
		return pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL
	default:
		return pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED
	}
}
