package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// startSession applies this package workflow transition.
func (s service) startSession(ctx context.Context, campaignID string, input StartSessionInput) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageSession); err != nil {
		return err
	}
	return s.mutationGateway.StartSession(ctx, campaignID, StartSessionInput{Name: strings.TrimSpace(input.Name)})
}

// endSession applies this package workflow transition.
func (s service) endSession(ctx context.Context, campaignID string, input EndSessionInput) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageSession); err != nil {
		return err
	}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.session_id_is_required", "session id is required")
	}
	return s.mutationGateway.EndSession(ctx, campaignID, EndSessionInput{SessionID: sessionID})
}
