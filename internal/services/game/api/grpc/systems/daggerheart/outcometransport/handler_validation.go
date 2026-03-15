package outcometransport

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// sessionOutcomePrelude carries the validated state shared by the session-level
// outcome handlers.
type sessionOutcomePrelude struct {
	campaignID    string
	sessionID     string
	rollPayload   action.RollResolvePayload
	rollMetadata  workflowtransport.RollSystemMetadata
	rollRequestID string
}

// validateCampaignIDFromContext keeps the transport entrypoints aligned on the
// same campaign source before they fan out into workflow-specific behavior.
func validateCampaignIDFromContext(ctx context.Context) (string, error) {
	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return "", status.Error(codes.InvalidArgument, "campaign id is required")
	}
	return campaignID, nil
}

// validateSessionOutcome loads and validates the resolved roll event shared by
// attack, adversary attack, and reaction outcome handlers.
func (h *Handler) validateSessionOutcome(
	ctx context.Context,
	sessionID string,
	rollSeq uint64,
) (sessionOutcomePrelude, error) {
	if err := h.requireSessionOutcomeDependencies(); err != nil {
		return sessionOutcomePrelude{}, err
	}

	campaignID, err := validateCampaignIDFromContext(ctx)
	if err != nil {
		return sessionOutcomePrelude{}, err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "session id is required")
	}
	if rollSeq == 0 {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return sessionOutcomePrelude{}, grpcerror.HandleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return sessionOutcomePrelude{}, grpcerror.HandleDomainError(err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, "campaign system does not support daggerheart outcomes"); err != nil {
		return sessionOutcomePrelude{}, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return sessionOutcomePrelude{}, grpcerror.HandleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return sessionOutcomePrelude{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return sessionOutcomePrelude{}, err
	}

	rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, rollSeq)
	if err != nil {
		return sessionOutcomePrelude{}, grpcerror.HandleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID.String() != sessionID {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return sessionOutcomePrelude{}, grpcerror.Internal("decode roll payload", err)
	}
	rollMetadata, err := workflowtransport.DecodeRollSystemMetadata(rollPayload.SystemData)
	if err != nil {
		return sessionOutcomePrelude{}, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return sessionOutcomePrelude{}, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	return sessionOutcomePrelude{
		campaignID:    campaignID,
		sessionID:     sessionID,
		rollPayload:   rollPayload,
		rollMetadata:  rollMetadata,
		rollRequestID: rollRequestID,
	}, nil
}
