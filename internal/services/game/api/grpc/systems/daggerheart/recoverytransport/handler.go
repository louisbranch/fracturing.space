package recoverytransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns Daggerheart recovery and life-state mutation transport.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a recovery transport handler from explicit reads and
// callback seams.
func NewHandler(deps Dependencies) *Handler {
	if deps.ResolveSeed == nil {
		deps.ResolveSeed = random.ResolveSeed
	}
	return &Handler{deps: deps}
}

func (h *Handler) ApplyRest(ctx context.Context, in *pb.DaggerheartApplyRestRequest) (RestResult, error) {
	if in == nil {
		return RestResult{}, status.Error(codes.InvalidArgument, "apply rest request is required")
	}
	if err := h.requireDependencies(true); err != nil {
		return RestResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return RestResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return RestResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return RestResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart rest"); err != nil {
		return RestResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return RestResult{}, err
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return RestResult{}, err
	}
	if in.Rest == nil {
		return RestResult{}, status.Error(codes.InvalidArgument, "rest is required")
	}
	if in.Rest.RestType == pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED {
		return RestResult{}, status.Error(codes.InvalidArgument, "rest_type is required")
	}

	seed, err := resolveSeed(in.Rest.GetRng(), h.deps.SeedGenerator, h.deps.ResolveSeed)
	if err != nil {
		return RestResult{}, grpcerror.Internal("failed to resolve rest seed", err)
	}
	currentSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return RestResult{}, grpcerror.Internal("get daggerheart snapshot", err)
	}
	state := daggerheart.RestState{ConsecutiveShortRests: currentSnap.ConsecutiveShortRests}
	restType, err := restTypeFromProto(in.Rest.RestType)
	if err != nil {
		return RestResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	outcome, err := daggerheart.ResolveRestOutcome(state, restType, in.Rest.Interrupted, seed, int(in.Rest.PartySize))
	if err != nil {
		return RestResult{}, grpcerror.HandleDomainError(err)
	}

	longTermCountdownID := strings.TrimSpace(in.Rest.GetLongTermCountdownId())
	var longTermCountdown *daggerheart.Countdown
	if outcome.AdvanceCountdown && longTermCountdownID != "" {
		storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, longTermCountdownID)
		if err != nil {
			return RestResult{}, grpcerror.HandleDomainError(err)
		}
		countdown := countdownFromStorage(storedCountdown)
		longTermCountdown = &countdown
	}

	characterIDs := append([]string(nil), in.GetCharacterIds()...)
	payload, err := daggerheart.ResolveRestApplication(daggerheart.RestApplicationInput{
		RestType:               restType,
		Interrupted:            in.Rest.Interrupted,
		Outcome:                outcome,
		CurrentGMFear:          currentSnap.GMFear,
		ConsecutiveShortRests:  currentSnap.ConsecutiveShortRests,
		CharacterIDs:           characterIDs,
		LongTermCountdownState: longTermCountdown,
	})
	if err != nil {
		return RestResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return RestResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartRestTake,
		SessionID:       sessionID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "session",
		EntityID:        campaignID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "rest did not emit an event",
		ApplyErrMessage: "apply rest event",
	}); err != nil {
		return RestResult{}, err
	}

	updatedSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return RestResult{}, grpcerror.Internal("load daggerheart snapshot", err)
	}
	entries := make([]CharacterStateEntry, 0, len(characterIDs))
	for _, characterID := range characterIDs {
		if strings.TrimSpace(characterID) == "" {
			continue
		}
		currentState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return RestResult{}, grpcerror.Internal("get daggerheart character state", err)
		}
		entries = append(entries, CharacterStateEntry{CharacterID: characterID, State: currentState})
	}
	return RestResult{Snapshot: updatedSnap, CharacterStates: entries}, nil
}

func (h *Handler) ApplyDowntimeMove(ctx context.Context, in *pb.DaggerheartApplyDowntimeMoveRequest) (CharacterStateResult, error) {
	if in == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "apply downtime request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return CharacterStateResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart downtime"); err != nil {
		return CharacterStateResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return CharacterStateResult{}, err
	}
	if in.Move == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "move is required")
	}
	if in.Move.Move == pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "downtime move is required")
	}

	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	current, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	state := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
		CampaignID:  campaignID,
		CharacterID: characterID,
		HP:          current.Hp,
		HPMax:       profile.HpMax,
		Hope:        current.Hope,
		HopeMax:     current.HopeMax,
		Stress:      current.Stress,
		StressMax:   profile.StressMax,
		Armor:       current.Armor,
		ArmorMax:    profile.ArmorMax,
		LifeState:   current.LifeState,
	})
	move, err := downtimeMoveFromProto(in.Move.Move)
	if err != nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	result := daggerheart.ApplyDowntimeMove(state, move, daggerheart.DowntimeOptions{PrepareWithGroup: in.Move.PrepareWithGroup})
	hopeBefore, hopeAfter := result.HopeBefore, result.HopeAfter
	stressBefore, stressAfter := result.StressBefore, result.StressAfter
	armorBefore, armorAfter := result.ArmorBefore, result.ArmorAfter
	payloadJSON, err := json.Marshal(daggerheart.DowntimeMoveApplyPayload{
		CharacterID:  ids.CharacterID(characterID),
		Move:         downtimeMoveToString(move),
		HopeBefore:   &hopeBefore,
		HopeAfter:    &hopeAfter,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
		ArmorBefore:  &armorBefore,
		ArmorAfter:   &armorAfter,
	})
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartDowntimeMoveApply,
		SessionID:       sessionID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "downtime move did not emit an event",
		ApplyErrMessage: "apply downtime move event",
	}); err != nil {
		return CharacterStateResult{}, err
	}
	if err := h.deps.ApplyStressConditionChange(ctx, StressConditionInput{
		CampaignID:   campaignID,
		SessionID:    sessionID,
		CharacterID:  characterID,
		Conditions:   current.Conditions,
		StressBefore: stressBefore,
		StressAfter:  stressAfter,
		StressMax:    profile.StressMax,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
	}); err != nil {
		return CharacterStateResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	return CharacterStateResult{CharacterID: characterID, State: updated}, nil
}

func (h *Handler) ApplyTemporaryArmor(ctx context.Context, in *pb.DaggerheartApplyTemporaryArmorRequest) (CharacterStateResult, error) {
	if in == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "apply temporary armor request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return CharacterStateResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart temporary armor"); err != nil {
		return CharacterStateResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return CharacterStateResult{}, err
	}
	if in.Armor == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "armor is required")
	}
	if _, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID); err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	if _, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID); err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	payloadJSON, err := json.Marshal(daggerheart.CharacterTemporaryArmorApplyPayload{
		CharacterID: ids.CharacterID(characterID),
		Source:      strings.TrimSpace(in.Armor.GetSource()),
		Duration:    strings.TrimSpace(in.Armor.GetDuration()),
		Amount:      int(in.Armor.GetAmount()),
		SourceID:    strings.TrimSpace(in.Armor.GetSourceId()),
	})
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCharacterTemporaryArmorApply,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "temporary armor apply did not emit an event",
		ApplyErrMessage: "apply temporary armor event",
	}); err != nil {
		return CharacterStateResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	return CharacterStateResult{CharacterID: characterID, State: updated}, nil
}

func (h *Handler) SwapLoadout(ctx context.Context, in *pb.DaggerheartSwapLoadoutRequest) (CharacterStateResult, error) {
	if in == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "swap loadout request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return CharacterStateResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart loadout"); err != nil {
		return CharacterStateResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return CharacterStateResult{}, err
	}
	if in.Swap == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "swap is required")
	}
	if _, err := validate.RequiredID(in.Swap.CardId, "card_id"); err != nil {
		return CharacterStateResult{}, err
	}
	if in.Swap.RecallCost < 0 {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "recall_cost must be non-negative")
	}
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	current, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	state := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
		CampaignID:  campaignID,
		CharacterID: characterID,
		HP:          current.Hp,
		HPMax:       profile.HpMax,
		Hope:        current.Hope,
		HopeMax:     current.HopeMax,
		Stress:      current.Stress,
		StressMax:   profile.StressMax,
		Armor:       current.Armor,
		ArmorMax:    profile.ArmorMax,
		LifeState:   current.LifeState,
	})
	stressBefore := state.Stress
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		if _, _, err := state.SpendResource(daggerheart.ResourceStress, int(in.Swap.RecallCost)); err != nil {
			return CharacterStateResult{}, grpcerror.HandleDomainError(err)
		}
	}
	stressAfter := state.Stress
	loadoutJSON, err := json.Marshal(daggerheart.LoadoutSwapPayload{
		CharacterID:  ids.CharacterID(characterID),
		CardID:       in.Swap.CardId,
		From:         "vault",
		To:           "active",
		RecallCost:   int(in.Swap.RecallCost),
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	})
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartLoadoutSwap,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     loadoutJSON,
		MissingEventMsg: "loadout swap did not emit an event",
		ApplyErrMessage: "apply loadout swap event",
	}); err != nil {
		return CharacterStateResult{}, err
	}
	if err := h.deps.ApplyStressConditionChange(ctx, StressConditionInput{
		CampaignID:   campaignID,
		SessionID:    sessionID,
		CharacterID:  characterID,
		Conditions:   current.Conditions,
		StressBefore: stressBefore,
		StressAfter:  stressAfter,
		StressMax:    profile.StressMax,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
	}); err != nil {
		return CharacterStateResult{}, err
	}
	if !in.Swap.InRest && in.Swap.RecallCost > 0 {
		stressSpendJSON, err := json.Marshal(daggerheart.StressSpendPayload{
			CharacterID: ids.CharacterID(characterID),
			Amount:      int(in.Swap.RecallCost),
			Before:      stressBefore,
			After:       stressAfter,
			Source:      "loadout_swap",
		})
		if err != nil {
			return CharacterStateResult{}, grpcerror.Internal("encode stress spend payload", err)
		}
		if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.DaggerheartStressSpend,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       grpcmeta.RequestIDFromContext(ctx),
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "character",
			EntityID:        characterID,
			PayloadJSON:     stressSpendJSON,
			MissingEventMsg: "stress spend did not emit an event",
			ApplyErrMessage: "apply stress spend event",
		}); err != nil {
			return CharacterStateResult{}, err
		}
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	return CharacterStateResult{CharacterID: characterID, State: updated}, nil
}

func (h *Handler) ApplyDeathMove(ctx context.Context, in *pb.DaggerheartApplyDeathMoveRequest) (DeathMoveResult, error) {
	if in == nil {
		return DeathMoveResult{}, status.Error(codes.InvalidArgument, "apply death move request is required")
	}
	if err := h.requireDependencies(true); err != nil {
		return DeathMoveResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return DeathMoveResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return DeathMoveResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return DeathMoveResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return DeathMoveResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart death moves"); err != nil {
		return DeathMoveResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return DeathMoveResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return DeathMoveResult{}, err
	}
	move, err := deathMoveFromProto(in.Move)
	if err != nil {
		return DeathMoveResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if move != daggerheart.DeathMoveRiskItAll && (in.HpClear != nil || in.StressClear != nil) {
		return DeathMoveResult{}, status.Error(codes.InvalidArgument, "hp_clear and stress_clear are only valid for risk it all")
	}
	seed, err := resolveSeed(in.GetRng(), h.deps.SeedGenerator, h.deps.ResolveSeed)
	if err != nil {
		return DeathMoveResult{}, grpcerror.Internal("failed to resolve death move seed", err)
	}
	profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return DeathMoveResult{}, grpcerror.HandleDomainError(err)
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return DeathMoveResult{}, grpcerror.HandleDomainError(err)
	}
	if state.Hp > 0 {
		return DeathMoveResult{}, status.Error(codes.FailedPrecondition, "death move requires hp to be zero")
	}
	if state.LifeState == daggerheart.LifeStateDead {
		return DeathMoveResult{}, status.Error(codes.FailedPrecondition, "character is already dead")
	}

	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = daggerheartprofile.PCHpMax
	}
	stressMax := profile.StressMax
	if stressMax < 0 {
		stressMax = 0
	}
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	level := profile.Level
	if level == 0 {
		level = daggerheartprofile.PCLevelDefault
	}

	var hpClear *int
	var stressClear *int
	if in.HpClear != nil {
		value := int(in.GetHpClear())
		hpClear = &value
	}
	if in.StressClear != nil {
		value := int(in.GetStressClear())
		stressClear = &value
	}

	outcome, err := daggerheart.ResolveDeathMove(daggerheart.DeathMoveInput{
		Move:             move,
		Level:            level,
		HP:               state.Hp,
		HPMax:            hpMax,
		Hope:             state.Hope,
		HopeMax:          hopeMax,
		Stress:           state.Stress,
		StressMax:        stressMax,
		RiskItAllHPClear: hpClear,
		RiskItAllStClear: stressClear,
		Seed:             seed,
	})
	if err != nil {
		return DeathMoveResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	hpBefore, hpAfter := outcome.HPBefore, outcome.HPAfter
	hopeBefore, hopeAfter := outcome.HopeBefore, outcome.HopeAfter
	hopeMaxBefore, hopeMaxAfter := outcome.HopeMaxBefore, outcome.HopeMaxAfter
	stressBefore, stressAfter := outcome.StressBefore, outcome.StressAfter
	lifeStateBefore := state.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheart.LifeStateAlive
	}
	patchPayload := daggerheart.CharacterStatePatchPayload{CharacterID: ids.CharacterID(characterID)}
	if hpBefore != hpAfter {
		patchPayload.HPBefore = &hpBefore
		patchPayload.HPAfter = &hpAfter
	}
	if hopeBefore != hopeAfter {
		patchPayload.HopeBefore = &hopeBefore
		patchPayload.HopeAfter = &hopeAfter
	}
	if hopeMaxBefore != hopeMaxAfter {
		patchPayload.HopeMaxBefore = &hopeMaxBefore
		patchPayload.HopeMaxAfter = &hopeMaxAfter
	}
	if stressBefore != stressAfter {
		patchPayload.StressBefore = &stressBefore
		patchPayload.StressAfter = &stressAfter
	}
	if lifeStateBefore != outcome.LifeState {
		lifeStateAfter := outcome.LifeState
		patchPayload.LifeStateBefore = &lifeStateBefore
		patchPayload.LifeStateAfter = &lifeStateAfter
	}
	if patchPayload.HPBefore == nil && patchPayload.HopeBefore == nil && patchPayload.HopeMaxBefore == nil && patchPayload.StressBefore == nil && patchPayload.LifeStateBefore == nil {
		return DeathMoveResult{}, status.Error(codes.Internal, "death move did not change character state")
	}
	payloadJSON, err := json.Marshal(patchPayload)
	if err != nil {
		return DeathMoveResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCharacterStatePatch,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "death move did not emit an event",
		ApplyErrMessage: "apply character state event",
	}); err != nil {
		return DeathMoveResult{}, err
	}
	if outcome.LifeState == daggerheart.LifeStateDead {
		if err := h.deps.AppendCharacterDeletedEvent(ctx, CharacterDeleteInput{
			CampaignID:  campaignID,
			CharacterID: characterID,
			Reason:      outcome.Move,
		}); err != nil {
			return DeathMoveResult{}, err
		}
	}
	if err := h.deps.ApplyStressConditionChange(ctx, StressConditionInput{
		CampaignID:   campaignID,
		SessionID:    sessionID,
		CharacterID:  characterID,
		Conditions:   state.Conditions,
		StressBefore: stressBefore,
		StressAfter:  stressAfter,
		StressMax:    stressMax,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
	}); err != nil {
		return DeathMoveResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return DeathMoveResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	return DeathMoveResult{
		CharacterID: characterID,
		State:       updated,
		Outcome: DeathOutcome{
			Move:          outcome.Move,
			LifeState:     outcome.LifeState,
			HopeDie:       outcome.HopeDie,
			FearDie:       outcome.FearDie,
			HPCleared:     outcome.HPCleared,
			StressCleared: outcome.StressCleared,
			ScarGained:    outcome.ScarGained,
		},
	}, nil
}

func (h *Handler) ResolveBlazeOfGlory(ctx context.Context, in *pb.DaggerheartResolveBlazeOfGloryRequest) (BlazeResult, error) {
	if in == nil {
		return BlazeResult{}, status.Error(codes.InvalidArgument, "resolve blaze of glory request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return BlazeResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return BlazeResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return BlazeResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return BlazeResult{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return BlazeResult{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart blaze of glory"); err != nil {
		return BlazeResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return BlazeResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return BlazeResult{}, err
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return BlazeResult{}, grpcerror.HandleDomainError(err)
	}
	if state.LifeState == "" {
		state.LifeState = daggerheart.LifeStateAlive
	}
	if state.LifeState == daggerheart.LifeStateDead {
		return BlazeResult{}, status.Error(codes.FailedPrecondition, "character is already dead")
	}
	if state.LifeState != daggerheart.LifeStateBlazeOfGlory {
		return BlazeResult{}, status.Error(codes.FailedPrecondition, "character is not in blaze of glory")
	}
	lifeStateBefore := state.LifeState
	lifeStateAfter := daggerheart.LifeStateDead
	payloadJSON, err := json.Marshal(daggerheart.CharacterStatePatchPayload{
		CharacterID:     ids.CharacterID(characterID),
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &lifeStateAfter,
	})
	if err != nil {
		return BlazeResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCharacterStatePatch,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "blaze of glory did not emit an event",
		ApplyErrMessage: "apply character state event",
	}); err != nil {
		return BlazeResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return BlazeResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	if err := h.deps.AppendCharacterDeletedEvent(ctx, CharacterDeleteInput{
		CampaignID:  campaignID,
		CharacterID: characterID,
		Reason:      daggerheart.LifeStateBlazeOfGlory,
	}); err != nil {
		return BlazeResult{}, err
	}
	return BlazeResult{CharacterID: characterID, State: updated, LifeState: lifeStateAfter}, nil
}

func (h *Handler) requireDependencies(requireSeed bool) error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.ExecuteSystemCommand == nil:
		return status.Error(codes.Internal, "system command executor is not configured")
	case h.deps.ApplyStressConditionChange == nil:
		return status.Error(codes.Internal, "stress condition callback is not configured")
	case h.deps.AppendCharacterDeletedEvent == nil:
		return status.Error(codes.Internal, "character deleted callback is not configured")
	case requireSeed && h.deps.SeedGenerator == nil:
		return status.Error(codes.Internal, "seed generator is not configured")
	default:
		return nil
	}
}
