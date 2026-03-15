package countdowntransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateCountdown validates a new countdown request, allocates identity when
// needed, emits the create command, and reloads the resulting projection.
func (h *Handler) CreateCountdown(ctx context.Context, in *pb.DaggerheartCreateCountdownRequest) (CreateResult, error) {
	if in == nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, "create countdown request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return CreateResult{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CreateResult{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return CreateResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	name, err := validate.RequiredID(in.GetName(), "name")
	if err != nil {
		return CreateResult{}, err
	}
	kind, err := countdownKindFromProto(in.GetKind())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	direction, err := countdownDirectionFromProto(in.GetDirection())
	if err != nil {
		return CreateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	max := int(in.GetMax())
	if max <= 0 {
		return CreateResult{}, status.Error(codes.InvalidArgument, "max must be positive")
	}
	current := int(in.GetCurrent())
	if current < 0 || current > max {
		return CreateResult{}, status.Errorf(codes.InvalidArgument, "current must be in range 0..%d", max)
	}

	if err := h.validateCampaignSession(ctx, campaignID, sessionID, campaign.CampaignOpSessionAction, "campaign system does not support daggerheart countdowns"); err != nil {
		return CreateResult{}, err
	}

	countdownID := strings.TrimSpace(in.GetCountdownId())
	if countdownID == "" {
		countdownID, err = h.deps.NewID()
		if err != nil {
			return CreateResult{}, grpcerror.Internal("generate countdown id", err)
		}
	}
	if _, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err == nil {
		return CreateResult{}, status.Error(codes.FailedPrecondition, "countdown already exists")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return CreateResult{}, grpcerror.HandleDomainError(err)
	}

	payloadJSON, err := json.Marshal(daggerheart.CountdownCreatePayload{
		CountdownID: ids.CountdownID(countdownID),
		Name:        name,
		Kind:        kind,
		Current:     current,
		Max:         max,
		Direction:   direction,
		Looping:     in.GetLooping(),
	})
	if err != nil {
		return CreateResult{}, grpcerror.Internal("encode countdown payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCountdownCreate,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "countdown",
		EntityID:        countdownID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "countdown create did not emit an event",
		ApplyErrMessage: "apply countdown created event",
	}); err != nil {
		return CreateResult{}, err
	}

	countdown, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return CreateResult{}, grpcerror.Internal("load countdown", err)
	}
	return CreateResult{Countdown: countdown}, nil
}
