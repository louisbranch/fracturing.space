package workfloweffects

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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
	if h == nil || h.deps.Daggerheart == nil || h.deps.CreateCountdown == nil || h.deps.UpdateCountdown == nil {
		return status.Error(codes.Internal, "workflow effects dependencies are not configured")
	}

	if _, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return grpcerror.HandleDomainError(err)
		}
		createErr := h.deps.CreateCountdown(ctx, &pb.DaggerheartCreateCountdownRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			CountdownId: countdownID,
			Name:        daggerheart.BreathCountdownName,
			Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE,
			Current:     daggerheart.BreathCountdownInitial,
			Max:         daggerheart.BreathCountdownMax,
			Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
			Looping:     false,
		})
		if createErr != nil && status.Code(createErr) != codes.FailedPrecondition {
			return createErr
		}
	}

	advance := daggerheart.ResolveBreathCountdownAdvance(failed)
	return h.deps.UpdateCountdown(ctx, &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CountdownId: countdownID,
		Delta:       int32(advance.Delta),
		Reason:      advance.Reason,
	})
}

func (h *Handler) buildStressVulnerableConditionEffect(
	ctx context.Context,
	campaignID string,
	sessionID string,
	characterID string,
	conditions []string,
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

	normalized, err := daggerheart.NormalizeConditions(conditions)
	if err != nil {
		return nil, grpcerror.Internal("invalid stored conditions", err)
	}
	hasVulnerable := containsString(normalized, daggerheart.ConditionVulnerable)
	if shouldAdd && hasVulnerable {
		return nil, nil
	}
	if shouldRemove && !hasVulnerable {
		return nil, nil
	}

	afterSet := make(map[string]struct{}, len(normalized)+1)
	for _, value := range normalized {
		afterSet[value] = struct{}{}
	}
	if shouldAdd {
		afterSet[daggerheart.ConditionVulnerable] = struct{}{}
	}
	if shouldRemove {
		delete(afterSet, daggerheart.ConditionVulnerable)
	}
	afterList := make([]string, 0, len(afterSet))
	for value := range afterSet {
		afterList = append(afterList, value)
	}
	after, err := daggerheart.NormalizeConditions(afterList)
	if err != nil {
		return nil, grpcerror.Internal("invalid condition set", err)
	}
	added, removed := daggerheart.DiffConditions(normalized, after)
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

	payloadJSON, err := json.Marshal(daggerheart.ConditionChangePayload{
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

func containsString(values []string, target string) bool {
	if target == "" {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
