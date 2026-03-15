package recoverytransport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
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

func handleDomainError(err error) error {
	return grpcerror.HandleDomainError(err)
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
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	return grpcerror.Internal("load session gate", err)
}

func resolveSeed(rng *commonv1.RngRequest, seedFunc func() (int64, error), resolve func(*commonv1.RngRequest, func() (int64, error), func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error)) (int64, error) {
	seed, _, _, err := resolve(rng, seedFunc, func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY })
	return seed, err
}

func countdownFromStorage(countdown projectionstore.DaggerheartCountdown) daggerheart.Countdown {
	return daggerheart.Countdown{
		CampaignID: countdown.CampaignID,
		ID:         countdown.CountdownID,
		Name:       countdown.Name,
		Kind:       countdown.Kind,
		Current:    countdown.Current,
		Max:        countdown.Max,
		Direction:  countdown.Direction,
		Looping:    countdown.Looping,
	}
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

func downtimeMoveFromProto(m pb.DaggerheartDowntimeMove) (daggerheart.DowntimeMove, error) {
	switch m {
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED:
		return daggerheart.DowntimePrepare, fmt.Errorf("downtime move is required")
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS:
		return daggerheart.DowntimeClearAllStress, nil
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_REPAIR_ALL_ARMOR:
		return daggerheart.DowntimeRepairAllArmor, nil
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_PREPARE:
		return daggerheart.DowntimePrepare, nil
	case pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_WORK_ON_PROJECT:
		return daggerheart.DowntimeWorkOnProject, nil
	default:
		return daggerheart.DowntimePrepare, fmt.Errorf("downtime move %v is invalid", m)
	}
}

func downtimeMoveToString(m daggerheart.DowntimeMove) string {
	switch m {
	case daggerheart.DowntimeClearAllStress:
		return "clear_all_stress"
	case daggerheart.DowntimeRepairAllArmor:
		return "repair_all_armor"
	case daggerheart.DowntimePrepare:
		return "prepare"
	case daggerheart.DowntimeWorkOnProject:
		return "work_on_project"
	default:
		return "unknown"
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
