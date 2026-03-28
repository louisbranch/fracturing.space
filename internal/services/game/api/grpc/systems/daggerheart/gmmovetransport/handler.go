package gmmovetransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmconsequence"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler owns the Daggerheart GM-move transport endpoint.
type Handler struct {
	deps Dependencies
}

type gmMoveResolution struct {
	target                  daggerheartpayload.GMMoveTarget
	opensGMConsequence      bool
	spotlightAdversary      *projectionstore.DaggerheartAdversary
	adversaryFeaturePayload *daggerheartpayload.AdversaryFeatureApplyPayload
}

// NewHandler builds a Daggerheart GM-move transport handler from explicit
// read-store and write-callback dependencies.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) ApplyGmMove(ctx context.Context, in *pb.DaggerheartApplyGmMoveRequest) (Result, error) {
	if in == nil {
		return Result{}, status.Error(codes.InvalidArgument, "apply gm move request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return Result{}, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return Result{}, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return Result{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	resolution, err := h.gmMoveTargetFromProto(ctx, campaignID, sessionID, in)
	if err != nil {
		return Result{}, err
	}
	fearSpent := int(in.GetFearSpent())
	if fearSpent <= 0 {
		return Result{}, status.Error(codes.InvalidArgument, "fear_spent must be greater than zero")
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return Result{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpSessionAction); err != nil {
		return Result{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart gm moves"); err != nil {
		return Result{}, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return Result{}, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if sess.Status != session.StatusActive {
		return Result{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if resolution.spotlightAdversary != nil {
		if err := h.validateAdversarySpotlight(ctx, campaignID, sessionID, *resolution.spotlightAdversary); err != nil {
			return Result{}, err
		}
	} else {
		if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
			return Result{}, err
		}
	}

	gmFearBefore := 0
	gmFearAfter := 0
	if snap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID); err == nil {
		gmFearBefore = snap.GMFear
		gmFearAfter = snap.GMFear
	}

	before, after, err := rules.ApplyGMFearSpend(gmFearBefore, fearSpent)
	if err != nil {
		return Result{}, status.Error(codes.InvalidArgument, err.Error())
	}
	gmFearBefore = before
	gmFearAfter = after

	payloadJSON, err := json.Marshal(daggerheartpayload.GMMoveApplyPayload{
		Target:    resolution.target,
		FearSpent: fearSpent,
	})
	if err != nil {
		return Result{}, grpcerror.Internal("encode gm move payload", err)
	}
	if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartGMMoveApply,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "session",
		EntityID:        sessionID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "gm move did not emit an event",
		ApplyErrMessage: "apply gm move event",
	}); err != nil {
		return Result{}, err
	}
	if resolution.adversaryFeaturePayload != nil {
		adversaryPayloadJSON, err := json.Marshal(resolution.adversaryFeaturePayload)
		if err != nil {
			return Result{}, grpcerror.Internal("encode adversary feature gm move payload", err)
		}
		if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.DaggerheartAdversaryFeatureApply,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       grpcmeta.RequestIDFromContext(ctx),
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "adversary",
			EntityID:        resolution.adversaryFeaturePayload.AdversaryID.String(),
			PayloadJSON:     adversaryPayloadJSON,
			MissingEventMsg: "gm move adversary feature stage did not emit an event",
			ApplyErrMessage: "apply adversary feature stage",
		}); err != nil {
			return Result{}, err
		}
	}
	if resolution.opensGMConsequence {
		if err := gmconsequence.Open(
			ctx,
			h.gmConsequenceDependencies(),
			campaignID,
			sessionID,
			sceneID,
			grpcmeta.RequestIDFromContext(ctx),
			grpcmeta.InvocationIDFromContext(ctx),
			nil,
		); err != nil {
			return Result{}, err
		}
	}
	if resolution.spotlightAdversary != nil {
		gateID, err := h.ensureGMConsequenceGate(ctx, campaignID, sessionID, sceneID)
		if err != nil {
			return Result{}, err
		}
		if err := h.recordAdversarySpotlight(ctx, campaignID, sessionID, sceneID, *resolution.spotlightAdversary, gateID); err != nil {
			return Result{}, err
		}
	}

	return Result{
		CampaignID:   campaignID,
		GMFearBefore: gmFearBefore,
		GMFearAfter:  gmFearAfter,
	}, nil
}

func (h *Handler) requireDependencies() error {
	switch {
	case h.deps.Campaign == nil:
		return status.Error(codes.Internal, "campaign store is not configured")
	case h.deps.Session == nil:
		return status.Error(codes.Internal, "session store is not configured")
	case h.deps.SessionGate == nil:
		return status.Error(codes.Internal, "session gate store is not configured")
	case h.deps.SessionSpotlight == nil:
		return status.Error(codes.Internal, "session spotlight store is not configured")
	case h.deps.Daggerheart == nil:
		return status.Error(codes.Internal, "daggerheart store is not configured")
	case h.deps.Content == nil:
		return status.Error(codes.Internal, "daggerheart content store is not configured")
	case h.deps.ExecuteDomainCommand == nil:
		return status.Error(codes.Internal, "domain command executor is not configured")
	case h.deps.ExecuteCoreCommand == nil:
		return status.Error(codes.Internal, "core command executor is not configured")
	default:
		return nil
	}
}

func (h *Handler) gmConsequenceDependencies() gmconsequence.Dependencies {
	return gmconsequence.Dependencies{
		SessionGate:        h.deps.SessionGate,
		SessionSpotlight:   h.deps.SessionSpotlight,
		ExecuteCoreCommand: h.deps.ExecuteCoreCommand,
	}
}
