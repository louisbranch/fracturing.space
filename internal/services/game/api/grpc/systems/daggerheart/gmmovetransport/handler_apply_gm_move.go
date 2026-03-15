package gmmovetransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	bridgedaggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyGmMove validates one GM move request, applies any fear spend through
// the domain command seam, and returns the resulting campaign fear state.
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
	if _, err := validate.RequiredID(in.GetMove(), "move"); err != nil {
		return Result{}, err
	}
	if in.GetFearSpent() < 0 {
		return Result{}, status.Error(codes.InvalidArgument, "fear_spent must be non-negative")
	}

	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return Result{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpSessionAction); err != nil {
		return Result{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "campaign system does not support daggerheart gm moves"); err != nil {
		return Result{}, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return Result{}, grpcerror.HandleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return Result{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return Result{}, err
	}

	gmFearBefore := 0
	gmFearAfter := 0
	if snap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID); err == nil {
		gmFearBefore = snap.GMFear
		gmFearAfter = snap.GMFear
	}

	fearSpent := int(in.GetFearSpent())
	if fearSpent > 0 {
		before, after, err := bridgedaggerheart.ApplyGMFearSpend(gmFearBefore, fearSpent)
		if err != nil {
			return Result{}, status.Error(codes.InvalidArgument, err.Error())
		}
		gmFearBefore = before
		gmFearAfter = after

		payloadJSON, err := json.Marshal(bridgedaggerheart.GMFearSetPayload{
			After:  &gmFearAfter,
			Reason: "gm_move",
		})
		if err != nil {
			return Result{}, grpcerror.Internal("encode gm fear payload", err)
		}
		if err := h.deps.ExecuteDomainCommand(ctx, DomainCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandids.DaggerheartGMFearSet,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       grpcmeta.RequestIDFromContext(ctx),
			InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
			EntityType:      "campaign",
			EntityID:        campaignID,
			PayloadJSON:     payloadJSON,
			MissingEventMsg: "gm fear update did not emit an event",
			ApplyErrMessage: "apply gm fear event",
		}); err != nil {
			return Result{}, err
		}
	}

	return Result{
		CampaignID:   campaignID,
		GMFearBefore: gmFearBefore,
		GMFearAfter:  gmFearAfter,
	}, nil
}
