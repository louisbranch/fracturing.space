package recoverytransport

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
		return RestResult{}, handleDomainError(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return RestResult{}, handleDomainError(ctx, err)
	}
	if err := requireDaggerheartSystem(record, "campaign system does not support daggerheart rest"); err != nil {
		return RestResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return RestResult{}, err
	}
	if err := ensureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
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
	if err != nil {
		if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "get daggerheart snapshot"); lookupErr != nil {
			return RestResult{}, lookupErr
		}
	}
	restType, err := restTypeFromProto(in.Rest.RestType)
	if err != nil {
		return RestResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	participants := in.Rest.GetParticipants()
	if len(participants) == 0 {
		return RestResult{}, status.Error(codes.InvalidArgument, "rest participants are required")
	}

	countdownsByID := make(map[dhids.CountdownID]rules.Countdown)
	longTermCountdownID := dhids.CountdownID(strings.TrimSpace(in.Rest.GetLongRestCampaignCountdownId()))
	if longTermCountdownID != "" {
		storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, longTermCountdownID.String())
		if err != nil {
			return RestResult{}, handleDomainError(ctx, err)
		}
		if storedCountdown.SessionID != "" || storedCountdown.SceneID != "" {
			return RestResult{}, status.Error(codes.InvalidArgument, "campaign_countdown_id must reference a campaign countdown")
		}
		countdown := countdownFromStorage(storedCountdown)
		countdownsByID[longTermCountdownID] = countdown
	}

	profilesByCharacterID := make(map[ids.CharacterID]projectionstore.DaggerheartCharacterProfile, len(participants))
	statesByCharacterID := make(map[ids.CharacterID]projectionstore.DaggerheartCharacterState, len(participants))
	participantInputs := make([]daggerheart.RestParticipantInput, 0, len(participants))
	for _, participant := range participants {
		characterID, err := validate.RequiredID(participant.GetCharacterId(), "participant character id")
		if err != nil {
			return RestResult{}, err
		}
		profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
		if err != nil {
			return RestResult{}, handleDomainError(ctx, err)
		}
		current, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			return RestResult{}, handleDomainError(ctx, err)
		}
		normalizedCharacterID := ids.CharacterID(characterID)
		profilesByCharacterID[normalizedCharacterID] = profile
		statesByCharacterID[normalizedCharacterID] = current

		moves := make([]daggerheart.DowntimeSelection, 0, len(participant.GetDowntimeMoves()))
		for _, selection := range participant.GetDowntimeMoves() {
			move, err := downtimeSelectionFromProto(selection, h.deps.ResolveSeed, h.deps.SeedGenerator)
			if err != nil {
				return RestResult{}, err
			}
			if move.CountdownID != "" {
				if _, exists := countdownsByID[move.CountdownID]; !exists {
					storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, move.CountdownID.String())
					if err != nil {
						return RestResult{}, handleDomainError(ctx, err)
					}
					if storedCountdown.SessionID != "" || storedCountdown.SceneID != "" {
						return RestResult{}, status.Error(codes.InvalidArgument, "work_on_project requires a campaign countdown")
					}
					countdown := countdownFromStorage(storedCountdown)
					countdownsByID[move.CountdownID] = countdown
				}
			}
			moves = append(moves, move)
		}

		participantState := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
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
		participantInputs = append(participantInputs, daggerheart.RestParticipantInput{
			CharacterID: ids.CharacterID(characterID),
			Level:       profile.Level,
			State:       *participantState,
			Moves:       moves,
		})
	}

	var longTermCountdown *rules.Countdown
	if countdown, ok := countdownsByID[longTermCountdownID]; ok {
		longTermCountdown = &countdown
	}

	result, err := daggerheart.ResolveRestPackage(daggerheart.RestPackageInput{
		RestType:              restType,
		Interrupted:           in.Rest.Interrupted,
		RestSeed:              seed,
		CurrentGMFear:         currentSnap.GMFear,
		ConsecutiveShortRests: currentSnap.ConsecutiveShortRests,
		Participants:          participantInputs,
		AvailableCountdowns:   countdownsByID,
		LongTermCountdown:     longTermCountdown,
	})
	if err != nil {
		return RestResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	payloadJSON, err := json.Marshal(result.Payload)
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
	if err := h.applyRestStressConditionChanges(ctx, campaignID, sessionID, profilesByCharacterID, statesByCharacterID, result.Payload); err != nil {
		return RestResult{}, err
	}
	updatedSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return RestResult{}, grpcerror.Internal("load daggerheart snapshot", err)
	}
	affectedCharacterIDs := affectedRestCharacterIDs(result.Payload)
	entries := make([]CharacterStateEntry, 0, len(affectedCharacterIDs))
	for _, characterID := range affectedCharacterIDs {
		currentState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
		if err != nil {
			if grpcerror.OptionalLookupErrorContext(ctx, err, "get daggerheart character state") == nil {
				continue
			}
			return RestResult{}, grpcerror.Internal("get daggerheart character state", err)
		}
		entries = append(entries, CharacterStateEntry{CharacterID: characterID, State: currentState})
	}
	countdowns := make([]projectionstore.DaggerheartCountdown, 0, len(result.UpdatedCountdownIDs))
	for _, countdownID := range result.UpdatedCountdownIDs {
		countdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID.String())
		if err != nil {
			if grpcerror.OptionalLookupErrorContext(ctx, err, "load daggerheart countdown") == nil {
				continue
			}
			return RestResult{}, grpcerror.Internal("load daggerheart countdown", err)
		}
		countdowns = append(countdowns, countdown)
	}
	return RestResult{
		Snapshot:                  updatedSnap,
		CharacterStates:           entries,
		Countdowns:                countdowns,
		CampaignCountdownAdvances: result.Payload.CampaignCountdownAdvances,
	}, nil
}

func affectedRestCharacterIDs(payload daggerheartpayload.RestTakePayload) []string {
	idsSeen := make([]string, 0, len(payload.Participants))
	for _, participantID := range payload.Participants {
		raw := strings.TrimSpace(participantID.String())
		if raw != "" {
			idsSeen = append(idsSeen, raw)
		}
	}
	for _, move := range payload.DowntimeMoves {
		targetID := strings.TrimSpace(move.TargetCharacterID.String())
		if targetID != "" {
			idsSeen = append(idsSeen, targetID)
		}
	}
	idsSeen = slices.Compact(idsSeen)
	return idsSeen
}

func (h *Handler) applyRestStressConditionChanges(
	ctx context.Context,
	campaignID string,
	sessionID string,
	profiles map[ids.CharacterID]projectionstore.DaggerheartCharacterProfile,
	states map[ids.CharacterID]projectionstore.DaggerheartCharacterState,
	payload daggerheartpayload.RestTakePayload,
) error {
	type stressChange struct {
		before int
		after  int
	}
	changes := make(map[ids.CharacterID]stressChange)
	for _, move := range payload.DowntimeMoves {
		if move.Stress == nil {
			continue
		}
		targetID := move.TargetCharacterID
		if strings.TrimSpace(targetID.String()) == "" {
			targetID = move.ActorCharacterID
		}
		if strings.TrimSpace(targetID.String()) == "" {
			continue
		}
		change, exists := changes[targetID]
		if !exists {
			state, ok := states[targetID]
			if !ok {
				return grpcerror.Internal("rest stress target state missing", fmt.Errorf("missing stress baseline for %s", targetID))
			}
			change.before = state.Stress
		}
		change.after = *move.Stress
		changes[targetID] = change
	}

	for characterID, change := range changes {
		profile, ok := profiles[characterID]
		if !ok {
			return grpcerror.Internal("rest stress target profile missing", fmt.Errorf("missing stress profile for %s", characterID))
		}
		current, ok := states[characterID]
		if !ok {
			return grpcerror.Internal("rest stress target state missing", fmt.Errorf("missing stress state for %s", characterID))
		}
		if err := h.deps.ApplyStressConditionChange(ctx, StressConditionInput{
			CampaignID:   campaignID,
			SessionID:    sessionID,
			CharacterID:  characterID.String(),
			Conditions:   current.Conditions,
			StressBefore: change.before,
			StressAfter:  change.after,
			StressMax:    profile.StressMax,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
		}); err != nil {
			return err
		}
	}
	return nil
}
