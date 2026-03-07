package app

import (
	"context"
	"encoding/json"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
)

// service defines an internal contract used at this web package boundary.
type service struct {
	auth Gateway
}

// NewService constructs a publicauth service with fail-closed gateway defaults.
func NewService(gateway Gateway) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{auth: gateway}
}

// HealthBody centralizes this web behavior in one helper seam.
func (service) HealthBody() string {
	return "ok"
}

// PasskeyLoginStart centralizes this web behavior in one helper seam.
func (s service) PasskeyLoginStart(ctx context.Context) (PasskeyChallenge, error) {
	return s.auth.BeginPasskeyLogin(ctx)
}

// PasskeyLoginFinish centralizes this web behavior in one helper seam.
func (s service) PasskeyLoginFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyFinish, error) {
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	if err := requireCredential(credential); err != nil {
		return PasskeyFinish{}, err
	}
	userID, err := s.auth.FinishPasskeyLogin(ctx, resolvedSessionID, credential)
	if err != nil {
		return PasskeyFinish{}, err
	}
	resolvedUserID, err := requireGatewayUserID(userID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	webSessionID, err := s.auth.CreateWebSession(ctx, resolvedUserID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	resolvedWebSessionID, err := requireGatewaySessionID(webSessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	return PasskeyFinish{
		SessionID: resolvedWebSessionID,
		UserID:    resolvedUserID,
	}, nil
}

// PasskeyRegisterStart centralizes this web behavior in one helper seam.
func (s service) PasskeyRegisterStart(ctx context.Context, email string) (PasskeyRegisterResult, error) {
	resolvedEmail, err := requireEmail(email)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	userID, err := s.auth.CreateUser(ctx, resolvedEmail)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	resolvedUserID, err := requireGatewayUserID(userID)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	challenge, err := s.auth.BeginPasskeyRegistration(ctx, resolvedUserID)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	resolvedChallengeSessionID, err := requireGatewaySessionID(challenge.SessionID)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	return PasskeyRegisterResult{
		SessionID: resolvedChallengeSessionID,
		UserID:    resolvedUserID,
		PublicKey: challenge.PublicKey,
	}, nil
}

// PasskeyRegisterFinish centralizes this web behavior in one helper seam.
func (s service) PasskeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyFinish, error) {
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	if err := requireCredential(credential); err != nil {
		return PasskeyFinish{}, err
	}
	userID, err := s.auth.FinishPasskeyRegistration(ctx, resolvedSessionID, credential)
	if err != nil {
		return PasskeyFinish{}, err
	}
	resolvedUserID, err := requireGatewayUserID(userID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	return PasskeyFinish{UserID: resolvedUserID}, nil
}

// HasValidWebSession reports whether this package condition is satisfied.
func (s service) HasValidWebSession(ctx context.Context, sessionID string) bool {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return false
	}
	return s.auth.HasValidWebSession(ctx, resolvedSessionID)
}

// RevokeWebSession applies this package workflow transition.
func (s service) RevokeWebSession(ctx context.Context, sessionID string) error {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return nil
	}
	return s.auth.RevokeWebSession(ctx, resolvedSessionID)
}

// requireSessionID validates inbound session ids for passkey completion flows.
func requireSessionID(sessionID string) (string, error) {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return "", apperrors.E(apperrors.KindInvalidInput, "session_id is required")
	}
	return resolvedSessionID, nil
}

// requireCredential validates passkey credential payloads for finish operations.
func requireCredential(credential json.RawMessage) error {
	if len(credential) == 0 {
		return apperrors.E(apperrors.KindInvalidInput, "credential is required")
	}
	return nil
}

// requireEmail validates inbound registration email payloads.
func requireEmail(email string) (string, error) {
	resolvedEmail := strings.TrimSpace(email)
	if resolvedEmail == "" {
		return "", apperrors.E(apperrors.KindInvalidInput, "email is required")
	}
	return resolvedEmail, nil
}

// requireGatewayUserID validates gateway-returned user IDs for auth workflows.
func requireGatewayUserID(userID string) (string, error) {
	resolvedUserID := userid.Normalize(userID)
	if resolvedUserID == "" {
		return "", apperrors.E(apperrors.KindUnavailable, "auth gateway user id unavailable")
	}
	return resolvedUserID, nil
}

// requireGatewaySessionID validates gateway-returned session IDs for auth workflows.
func requireGatewaySessionID(sessionID string) (string, error) {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return "", apperrors.E(apperrors.KindUnavailable, "auth gateway session id unavailable")
	}
	return resolvedSessionID, nil
}
