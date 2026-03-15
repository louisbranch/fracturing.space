package countdowntransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UpdateCountdown resolves one countdown mutation, emits the update command,
// and returns both the projection and canonical delta summary.
func (h *Handler) UpdateCountdown(ctx context.Context, in *pb.DaggerheartUpdateCountdownRequest) (UpdateResult, error) {
	if in == nil {
		return UpdateResult{}, status.Error(codes.InvalidArgument, "update countdown request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return UpdateResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return UpdateResult{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return UpdateResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return UpdateResult{}, err
	}
	if in.Current == nil && in.GetDelta() == 0 {
		return UpdateResult{}, status.Error(codes.InvalidArgument, "delta or current is required")
	}

	if err := h.validateCampaignSession(ctx, campaignID, sessionID, campaign.CampaignOpSessionAction, "campaign system does not support daggerheart countdowns"); err != nil {
		return UpdateResult{}, err
	}

	storedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return UpdateResult{}, grpcerror.HandleDomainError(err)
	}
	var override *int
	if in.Current != nil {
		value := int(in.GetCurrent())
		override = &value
	}
	mutation, err := daggerheart.ResolveCountdownMutation(daggerheart.CountdownMutationInput{
		Countdown: countdownFromStorage(storedCountdown),
		Delta:     int(in.GetDelta()),
		Override:  override,
		Reason:    strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return UpdateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	payloadJSON, err := json.Marshal(mutation.Payload)
	if err != nil {
		return UpdateResult{}, grpcerror.Internal("encode countdown update payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCountdownUpdate,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "countdown",
		EntityID:        countdownID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "countdown update did not emit an event",
		ApplyErrMessage: "apply countdown update event",
	}); err != nil {
		return UpdateResult{}, err
	}

	updatedCountdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return UpdateResult{}, grpcerror.Internal("load countdown", err)
	}
	return UpdateResult{
		Countdown: updatedCountdown,
		Before:    mutation.Update.Before,
		After:     mutation.Update.After,
		Delta:     mutation.Update.Delta,
	}, nil
}
