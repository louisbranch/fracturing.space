package workfloweffects

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/countdowns"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	deps Dependencies
}

func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) ApplyStressVulnerableCondition(ctx context.Context, in ApplyStressVulnerableConditionInput) error {
	effect, err := h.buildStressVulnerableConditionEffect(
		ctx,
		in.CampaignID,
		in.SessionID,
		in.CharacterID,
		in.Conditions,
		in.StressBefore,
		in.StressAfter,
		in.StressMax,
		in.RollSeq,
		in.RequestID,
	)
	if err != nil {
		return err
	}
	if effect == nil {
		return nil
	}
	if h == nil || h.deps.ExecuteConditionChange == nil {
		return status.Error(codes.Internal, "condition change executor is not configured")
	}

	return h.deps.ExecuteConditionChange(ctx, ConditionChangeCommandInput{
		CampaignID:    in.CampaignID,
		SessionID:     in.SessionID,
		RequestID:     in.RequestID,
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		CorrelationID: in.CorrelationID,
		CharacterID:   in.CharacterID,
		PayloadJSON:   effect.PayloadJSON,
	})
}

func (h *Handler) AdvanceBreathCountdown(ctx context.Context, campaignID, sessionID, countdownID string, failed bool) error {
	countdownID = strings.TrimSpace(countdownID)
	if countdownID == "" {
		return nil
	}
	if h == nil || h.deps.Daggerheart == nil || h.deps.CreateSceneCountdown == nil || h.deps.AdvanceSceneCountdown == nil {
		return status.Error(codes.Internal, "workflow effects dependencies are not configured")
	}

	if _, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err != nil {
		if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load breath countdown"); lookupErr != nil {
			return lookupErr
		}
		createErr := h.deps.CreateSceneCountdown(ctx, &pb.DaggerheartCreateSceneCountdownRequest{
			CampaignId:        campaignID,
			SessionId:         sessionID,
			SceneId:           "",
			CountdownId:       countdownID,
			Name:              countdowns.BreathCountdownName,
			Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_NEUTRAL,
			AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
			StartingValue: &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{
				FixedStartingValue: 3,
			},
			LoopBehavior: pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
		})
		if createErr != nil && status.Code(createErr) != codes.FailedPrecondition {
			return createErr
		}
	}

	advance := countdowns.ResolveBreathCountdownAdvance(failed)
	return h.deps.AdvanceSceneCountdown(ctx, &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     "",
		CountdownId: countdownID,
		Amount:      int32(advance.Amount),
		Reason:      advance.Reason,
	})
}

func (h *Handler) buildStressVulnerableConditionEffect(
	ctx context.Context,
	campaignID string,
	sessionID string,
	characterID string,
	conditions []projectionstore.DaggerheartConditionState,
	stressBefore int,
	stressAfter int,
	stressMax int,
	rollSeq *uint64,
	requestID string,
) (*action.OutcomeAppliedEffect, error) {
	if stressMax <= 0 || stressBefore == stressAfter {
		return nil, nil
	}
	shouldAdd := stressBefore < stressMax && stressAfter == stressMax
	shouldRemove := stressBefore == stressMax && stressAfter < stressMax
	if !shouldAdd && !shouldRemove {
		return nil, nil
	}

	normalized, err := rules.NormalizeConditionStates(projectionConditionsToDomain(conditions))
	if err != nil {
		return nil, grpcerror.Internal("invalid stored conditions", err)
	}
	hasVulnerable := rules.HasConditionCode(normalized, rules.ConditionVulnerable)
	if shouldAdd && hasVulnerable {
		return nil, nil
	}
	if shouldRemove && !hasVulnerable {
		return nil, nil
	}

	afterSet := make(map[string]rules.ConditionState, len(normalized)+1)
	for _, value := range normalized {
		afterSet[value.ID] = value
	}
	if shouldAdd {
		vulnerable, err := rules.StandardConditionState(rules.ConditionVulnerable)
		if err != nil {
			return nil, grpcerror.Internal("build vulnerable condition", err)
		}
		afterSet[vulnerable.ID] = vulnerable
	}
	if shouldRemove {
		delete(afterSet, rules.ConditionVulnerable)
	}
	afterList := make([]rules.ConditionState, 0, len(afterSet))
	for _, value := range afterSet {
		afterList = append(afterList, value)
	}
	after, err := rules.NormalizeConditionStates(afterList)
	if err != nil {
		return nil, grpcerror.Internal("invalid condition set", err)
	}
	added, removed := rules.DiffConditionStates(normalized, after)
	if len(added) == 0 && len(removed) == 0 {
		return nil, nil
	}

	if rollSeq != nil && *rollSeq > 0 {
		if h == nil || h.deps.ConditionChangeAlreadyApplied == nil {
			return nil, status.Error(codes.Internal, "condition replay check is not configured")
		}
		exists, err := h.deps.ConditionChangeAlreadyApplied(ctx, ConditionChangeReplayCheckInput{
			CampaignID:  campaignID,
			SessionID:   sessionID,
			RollSeq:     *rollSeq,
			RequestID:   requestID,
			CharacterID: characterID,
		})
		if err != nil {
			return nil, grpcerror.Internal("check condition change applied", err)
		}
		if exists {
			return nil, nil
		}
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.ConditionChangePayload{
		CharacterID:      ids.CharacterID(characterID),
		ConditionsBefore: normalized,
		ConditionsAfter:  after,
		Added:            added,
		Removed:          removed,
		RollSeq:          rollSeq,
	})
	if err != nil {
		return nil, grpcerror.Internal("encode condition payload", err)
	}
	return &action.OutcomeAppliedEffect{
		Type:          "sys.daggerheart.condition_changed",
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, nil
}

func projectionConditionsToDomain(values []projectionstore.DaggerheartConditionState) []rules.ConditionState {
	if len(values) == 0 {
		return nil
	}
	items := make([]rules.ConditionState, 0, len(values))
	for _, value := range values {
		triggers := make([]rules.ConditionClearTrigger, 0, len(value.ClearTriggers))
		for _, trigger := range value.ClearTriggers {
			triggers = append(triggers, rules.ConditionClearTrigger(trigger))
		}
		items = append(items, rules.ConditionState{
			ID:            value.ID,
			Class:         rules.ConditionClass(value.Class),
			Standard:      value.Standard,
			Code:          value.Code,
			Label:         value.Label,
			Source:        value.Source,
			SourceID:      value.SourceID,
			ClearTriggers: triggers,
		})
	}
	return items
}
