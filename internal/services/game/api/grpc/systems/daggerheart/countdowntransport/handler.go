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
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the Daggerheart countdown mutation transport endpoints.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a countdown mutation transport handler from explicit read
// stores, ID generation, and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

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

	payloadJSON, err := json.Marshal(daggerheart.CountdownDeletePayload{
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

func (h *Handler) requireDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.NewID == nil:
		return status.Error(codes.Internal, "countdown id generator is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	default:
		return nil
	}
}

func (h *Handler) validateCampaignSession(ctx context.Context, campaignID, sessionID string, operation campaign.CampaignOperation, unsupportedMessage string) error {
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, operation); err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, unsupportedMessage); err != nil {
		return err
	}
	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return grpcerror.HandleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return status.Error(codes.FailedPrecondition, "session is not active")
	}
	return daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID)
}
