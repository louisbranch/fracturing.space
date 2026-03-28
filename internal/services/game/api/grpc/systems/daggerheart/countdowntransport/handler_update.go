package countdowntransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/countdowns"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) AdvanceSceneCountdown(ctx context.Context, in *pb.DaggerheartAdvanceSceneCountdownRequest) (AdvanceResult, error) {
	if in == nil {
		return AdvanceResult{}, status.Error(codes.InvalidArgument, "advance scene countdown request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return AdvanceResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return AdvanceResult{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return AdvanceResult{}, err
	}
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return AdvanceResult{}, err
	}
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return AdvanceResult{}, err
	}
	if in.GetAmount() <= 0 {
		return AdvanceResult{}, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if err := h.validateCampaignSession(ctx, campaignID, sessionID, campaign.CampaignOpSessionAction, "campaign system does not support daggerheart scene countdowns"); err != nil {
		return AdvanceResult{}, err
	}
	storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return AdvanceResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if storedCountdown.SessionID != sessionID || storedCountdown.SceneID != sceneID {
		return AdvanceResult{}, status.Error(codes.NotFound, "scene countdown was not found")
	}
	mutation, err := resolveCountdownAdvance(storedCountdown, int(in.GetAmount()), strings.TrimSpace(in.GetReason()))
	if err != nil {
		return AdvanceResult{}, err
	}
	if err := h.executeCountdownMutation(ctx, campaignID, sessionID, sceneID, countdownID, mutation.Payload, commandids.DaggerheartSceneCountdownAdvance, "scene_countdown", "scene countdown advance did not emit an event", "apply scene countdown advance event"); err != nil {
		return AdvanceResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return AdvanceResult{}, grpcerror.Internal("load scene countdown", err)
	}
	return AdvanceResult{
		Countdown: updated,
		Summary: CountdownAdvanceSummary{
			BeforeRemaining: mutation.Advance.BeforeRemaining,
			AfterRemaining:  mutation.Advance.AfterRemaining,
			AdvancedBy:      mutation.Advance.AdvancedBy,
			StatusBefore:    mutation.Advance.StatusBefore,
			StatusAfter:     mutation.Advance.StatusAfter,
			Triggered:       mutation.Advance.Triggered,
		},
	}, nil
}

func (h *Handler) AdvanceCampaignCountdown(ctx context.Context, in *pb.DaggerheartAdvanceCampaignCountdownRequest) (AdvanceResult, error) {
	if in == nil {
		return AdvanceResult{}, status.Error(codes.InvalidArgument, "advance campaign countdown request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return AdvanceResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return AdvanceResult{}, err
	}
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return AdvanceResult{}, err
	}
	if in.GetAmount() <= 0 {
		return AdvanceResult{}, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if err := h.validateCampaignMutate(ctx, campaignID, "campaign system does not support daggerheart campaign countdowns"); err != nil {
		return AdvanceResult{}, err
	}
	storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return AdvanceResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if storedCountdown.SessionID != "" || storedCountdown.SceneID != "" {
		return AdvanceResult{}, status.Error(codes.NotFound, "campaign countdown was not found")
	}
	mutation, err := resolveCountdownAdvance(storedCountdown, int(in.GetAmount()), strings.TrimSpace(in.GetReason()))
	if err != nil {
		return AdvanceResult{}, err
	}
	if err := h.executeCountdownMutation(ctx, campaignID, "", "", countdownID, mutation.Payload, commandids.DaggerheartCampaignCountdownAdvance, "campaign_countdown", "campaign countdown advance did not emit an event", "apply campaign countdown advance event"); err != nil {
		return AdvanceResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return AdvanceResult{}, grpcerror.Internal("load campaign countdown", err)
	}
	return AdvanceResult{
		Countdown: updated,
		Summary: CountdownAdvanceSummary{
			BeforeRemaining: mutation.Advance.BeforeRemaining,
			AfterRemaining:  mutation.Advance.AfterRemaining,
			AdvancedBy:      mutation.Advance.AdvancedBy,
			StatusBefore:    mutation.Advance.StatusBefore,
			StatusAfter:     mutation.Advance.StatusAfter,
			Triggered:       mutation.Advance.Triggered,
		},
	}, nil
}

func (h *Handler) ResolveSceneCountdownTrigger(ctx context.Context, in *pb.DaggerheartResolveSceneCountdownTriggerRequest) (TriggerResolveResult, error) {
	if in == nil {
		return TriggerResolveResult{}, status.Error(codes.InvalidArgument, "resolve scene countdown trigger request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return TriggerResolveResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return TriggerResolveResult{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return TriggerResolveResult{}, err
	}
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return TriggerResolveResult{}, err
	}
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return TriggerResolveResult{}, err
	}
	if err := h.validateCampaignSession(ctx, campaignID, sessionID, campaign.CampaignOpSessionAction, "campaign system does not support daggerheart scene countdowns"); err != nil {
		return TriggerResolveResult{}, err
	}
	storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return TriggerResolveResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if storedCountdown.SessionID != sessionID || storedCountdown.SceneID != sceneID {
		return TriggerResolveResult{}, status.Error(codes.NotFound, "scene countdown was not found")
	}
	mutation, err := countdowns.ResolveCountdownTrigger(countdownFromStorage(storedCountdown), strings.TrimSpace(in.GetReason()))
	if err != nil {
		return TriggerResolveResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := h.executeCountdownMutation(ctx, campaignID, sessionID, sceneID, countdownID, mutation.Payload, commandids.DaggerheartSceneCountdownTriggerResolve, "scene_countdown", "scene countdown trigger resolution did not emit an event", "apply scene countdown trigger resolution event"); err != nil {
		return TriggerResolveResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return TriggerResolveResult{}, grpcerror.Internal("load scene countdown", err)
	}
	return TriggerResolveResult{Countdown: updated}, nil
}

func (h *Handler) ResolveCampaignCountdownTrigger(ctx context.Context, in *pb.DaggerheartResolveCampaignCountdownTriggerRequest) (TriggerResolveResult, error) {
	if in == nil {
		return TriggerResolveResult{}, status.Error(codes.InvalidArgument, "resolve campaign countdown trigger request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return TriggerResolveResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return TriggerResolveResult{}, err
	}
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return TriggerResolveResult{}, err
	}
	if err := h.validateCampaignMutate(ctx, campaignID, "campaign system does not support daggerheart campaign countdowns"); err != nil {
		return TriggerResolveResult{}, err
	}
	storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return TriggerResolveResult{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if storedCountdown.SessionID != "" || storedCountdown.SceneID != "" {
		return TriggerResolveResult{}, status.Error(codes.NotFound, "campaign countdown was not found")
	}
	mutation, err := countdowns.ResolveCountdownTrigger(countdownFromStorage(storedCountdown), strings.TrimSpace(in.GetReason()))
	if err != nil {
		return TriggerResolveResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := h.executeCountdownMutation(ctx, campaignID, "", "", countdownID, mutation.Payload, commandids.DaggerheartCampaignCountdownTriggerResolve, "campaign_countdown", "campaign countdown trigger resolution did not emit an event", "apply campaign countdown trigger resolution event"); err != nil {
		return TriggerResolveResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return TriggerResolveResult{}, grpcerror.Internal("load campaign countdown", err)
	}
	return TriggerResolveResult{Countdown: updated}, nil
}

func resolveCountdownAdvance(storedCountdown projectionstore.DaggerheartCountdown, amount int, reason string) (countdowns.CountdownAdvanceMutation, error) {
	mutation, err := countdowns.ResolveCountdownAdvance(countdowns.CountdownAdvanceInput{
		Countdown: countdownFromStorage(storedCountdown),
		Amount:    amount,
		Reason:    reason,
	})
	if err != nil {
		return countdowns.CountdownAdvanceMutation{}, status.Error(codes.InvalidArgument, err.Error())
	}
	return mutation, nil
}

func (h *Handler) executeCountdownMutation(ctx context.Context, campaignID, sessionID, sceneID, countdownID string, payload any, commandType command.Type, entityType, missingEventMsg, applyErrMessage string) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode countdown payload", err)
	}
	return h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandType,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      entityType,
		EntityID:        countdownID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: missingEventMsg,
		ApplyErrMessage: applyErrMessage,
	})
}
