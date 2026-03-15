package sessionrolltransport

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

type adversaryRollContext struct {
	CampaignID  string
	SessionID   string
	SceneID     string
	AdversaryID string
}

func (h *Handler) loadAdversaryRollContext(
	ctx context.Context,
	campaignID string,
	sessionID string,
	sceneID string,
	adversaryID string,
	notSupportedMessage string,
) (adversaryRollContext, error) {
	var err error

	campaignID, err = validate.RequiredID(campaignID, "campaign id")
	if err != nil {
		return adversaryRollContext{}, err
	}
	sessionID, err = validate.RequiredID(sessionID, "session id")
	if err != nil {
		return adversaryRollContext{}, err
	}
	adversaryID, err = validate.RequiredID(adversaryID, "adversary id")
	if err != nil {
		return adversaryRollContext{}, err
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return adversaryRollContext{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return adversaryRollContext{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, notSupportedMessage); err != nil {
		return adversaryRollContext{}, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return adversaryRollContext{}, grpcerror.HandleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return adversaryRollContext{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return adversaryRollContext{}, err
	}

	if _, err := h.deps.LoadAdversaryForSession(ctx, campaignID, sessionID, adversaryID); err != nil {
		return adversaryRollContext{}, err
	}

	return adversaryRollContext{
		CampaignID:  campaignID,
		SessionID:   sessionID,
		SceneID:     strings.TrimSpace(sceneID),
		AdversaryID: adversaryID,
	}, nil
}
