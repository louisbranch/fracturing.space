package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// StartSession applies this package workflow transition.
func (s sessionMutationService) StartSession(ctx context.Context, campaignID string, input StartSessionInput) error {
	return s.startSession(ctx, campaignID, input)
}

// EndSession applies this package workflow transition.
func (s sessionMutationService) EndSession(ctx context.Context, campaignID string, input EndSessionInput) error {
	return s.endSession(ctx, campaignID, input)
}

// startSession applies this package workflow transition.
func (s sessionMutationService) startSession(ctx context.Context, campaignID string, input StartSessionInput) error {
	if err := s.auth.requirePolicy(ctx, campaignID, policyManageSession); err != nil {
		return err
	}
	assignments := make([]SessionCharacterControllerAssignment, 0, len(input.CharacterControllers))
	for _, assignment := range input.CharacterControllers {
		assignments = append(assignments, SessionCharacterControllerAssignment{
			CharacterID:   strings.TrimSpace(assignment.CharacterID),
			ParticipantID: strings.TrimSpace(assignment.ParticipantID),
		})
	}
	return s.mutation.StartSession(ctx, campaignID, StartSessionInput{
		Name:                 strings.TrimSpace(input.Name),
		CharacterControllers: assignments,
	})
}

// endSession applies this package workflow transition.
func (s sessionMutationService) endSession(ctx context.Context, campaignID string, input EndSessionInput) error {
	if err := s.auth.requirePolicy(ctx, campaignID, policyManageSession); err != nil {
		return err
	}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.session_id_is_required", "session id is required")
	}
	return s.mutation.EndSession(ctx, campaignID, EndSessionInput{SessionID: sessionID})
}
