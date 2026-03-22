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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DeleteSceneCountdown validates the active session context for one scene
// countdown, emits the delete command, and returns the deleted identity.
func (h *Handler) DeleteSceneCountdown(ctx context.Context, in *pb.DaggerheartDeleteSceneCountdownRequest) (DeleteResult, error) {
	if in == nil {
		return DeleteResult{}, status.Error(codes.InvalidArgument, "delete scene countdown request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return DeleteResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return DeleteResult{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return DeleteResult{}, err
	}
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return DeleteResult{}, err
	}
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return DeleteResult{}, err
	}
	if err := h.validateCampaignSession(ctx, campaignID, sessionID, campaign.CampaignOpSessionAction, "campaign system does not support daggerheart scene countdowns"); err != nil {
		return DeleteResult{}, err
	}

	storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return DeleteResult{}, grpcerror.HandleDomainError(err)
	}
	if storedCountdown.SessionID != sessionID || storedCountdown.SceneID != sceneID {
		return DeleteResult{}, status.Error(codes.NotFound, "scene countdown was not found")
	}
	payloadJSON, err := json.Marshal(daggerheartpayload.SceneCountdownDeletePayload{
		CountdownID: ids.CountdownID(countdownID),
		Reason:      strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return DeleteResult{}, grpcerror.Internal("encode scene countdown delete payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartSceneCountdownDelete,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "scene_countdown",
		EntityID:        countdownID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "scene countdown delete did not emit an event",
		ApplyErrMessage: "apply scene countdown delete event",
	}); err != nil {
		return DeleteResult{}, err
	}
	return DeleteResult{CountdownID: countdownID}, nil
}

// DeleteCampaignCountdown validates one campaign countdown, emits the delete
// command, and returns the deleted identity.
func (h *Handler) DeleteCampaignCountdown(ctx context.Context, in *pb.DaggerheartDeleteCampaignCountdownRequest) (DeleteResult, error) {
	if in == nil {
		return DeleteResult{}, status.Error(codes.InvalidArgument, "delete campaign countdown request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return DeleteResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return DeleteResult{}, err
	}
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return DeleteResult{}, err
	}
	if err := h.validateCampaignMutate(ctx, campaignID, "campaign system does not support daggerheart campaign countdowns"); err != nil {
		return DeleteResult{}, err
	}

	storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return DeleteResult{}, grpcerror.HandleDomainError(err)
	}
	if storedCountdown.SessionID != "" || storedCountdown.SceneID != "" {
		return DeleteResult{}, status.Error(codes.NotFound, "campaign countdown was not found")
	}
	payloadJSON, err := json.Marshal(daggerheartpayload.CampaignCountdownDeletePayload{
		CountdownID: ids.CountdownID(countdownID),
		Reason:      strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return DeleteResult{}, grpcerror.Internal("encode campaign countdown delete payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCampaignCountdownDelete,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "campaign_countdown",
		EntityID:        countdownID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "campaign countdown delete did not emit an event",
		ApplyErrMessage: "apply campaign countdown delete event",
	}); err != nil {
		return DeleteResult{}, err
	}
	return DeleteResult{CountdownID: countdownID}, nil
}
