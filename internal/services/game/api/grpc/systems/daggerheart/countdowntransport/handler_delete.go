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

// DeleteCountdown validates the active session context for a countdown, emits
// the delete command, and returns the deleted identity.
func (h *Handler) DeleteCountdown(ctx context.Context, in *pb.DaggerheartDeleteCountdownRequest) (DeleteResult, error) {
	if in == nil {
		return DeleteResult{}, status.Error(codes.InvalidArgument, "delete countdown request is required")
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
	sceneID := strings.TrimSpace(in.GetSceneId())
	countdownID, err := validate.RequiredID(in.GetCountdownId(), "countdown id")
	if err != nil {
		return DeleteResult{}, err
	}

	if err := h.validateCampaignSession(ctx, campaignID, sessionID, campaign.CampaignOpSessionAction, "campaign system does not support daggerheart countdowns"); err != nil {
		return DeleteResult{}, err
	}
	if _, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err != nil {
		return DeleteResult{}, grpcerror.HandleDomainError(err)
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.CountdownDeletePayload{
		CountdownID: ids.CountdownID(countdownID),
		Reason:      strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return DeleteResult{}, grpcerror.Internal("encode countdown delete payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCountdownDelete,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "countdown",
		EntityID:        countdownID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "countdown delete did not emit an event",
		ApplyErrMessage: "apply countdown delete event",
	}); err != nil {
		return DeleteResult{}, err
	}

	return DeleteResult{CountdownID: countdownID}, nil
}
