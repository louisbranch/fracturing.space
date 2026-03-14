package app

import (
	"context"
	"encoding/json"
	"net/url"
	"path"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// service centralizes public auth orchestration so handlers stay transport-focused.
type service struct {
	auth        Gateway
	authBaseURL string
}

// NewService wires auth-backed public auth flows behind input validation.
func NewService(gateway Gateway, authBaseURL string) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{auth: gateway, authBaseURL: strings.TrimSpace(authBaseURL)}
}

// HealthBody returns the plain-text health response expected by the endpoint.
func (service) HealthBody() string {
	return "ok"
}

// PasskeyLoginStart normalizes the username before asking auth to begin login.
func (s service) PasskeyLoginStart(ctx context.Context, username string) (PasskeyChallenge, error) {
	resolvedUsername, err := requireUsername(username)
	if err != nil {
		return PasskeyChallenge{}, err
	}
	return s.auth.BeginPasskeyLogin(ctx, resolvedUsername)
}

// PasskeyLoginFinish validates the ceremony response, then creates a web session.
func (s service) PasskeyLoginFinish(ctx context.Context, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error) {
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	if err := requireCredential(credential); err != nil {
		return PasskeyFinish{}, err
	}
	userID, err := s.auth.FinishPasskeyLogin(ctx, resolvedSessionID, credential, strings.TrimSpace(pendingID))
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
	return PasskeyFinish{SessionID: resolvedWebSessionID, UserID: resolvedUserID}, nil
}

// PasskeyRegisterStart validates the requested username before starting signup.
func (s service) PasskeyRegisterStart(ctx context.Context, username string) (PasskeyRegisterResult, error) {
	resolvedUsername, err := requireUsername(username)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	challenge, err := s.auth.BeginAccountRegistration(ctx, resolvedUsername)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	resolvedChallengeSessionID, err := requireGatewaySessionID(challenge.SessionID)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	return PasskeyRegisterResult{SessionID: resolvedChallengeSessionID, PublicKey: challenge.PublicKey}, nil
}

// PasskeyRegisterFinish completes signup and normalizes returned identity fields.
func (s service) PasskeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyFinish, error) {
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	if err := requireCredential(credential); err != nil {
		return PasskeyFinish{}, err
	}
	finished, err := s.auth.FinishAccountRegistration(ctx, resolvedSessionID, credential)
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished.UserID, err = requireGatewayUserID(finished.UserID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished.SessionID, err = requireGatewaySessionID(finished.SessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished.RecoveryCode, err = requireRecoveryCode(finished.RecoveryCode)
	if err != nil {
		return PasskeyFinish{}, err
	}
	return finished, nil
}

// RecoveryStart verifies the recovery code, then starts replacement passkey enrollment.
func (s service) RecoveryStart(ctx context.Context, username string, recoveryCode string) (RecoveryChallenge, error) {
	resolvedUsername, err := requireUsername(username)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	resolvedRecoveryCode, err := requireRecoveryCode(recoveryCode)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	recoverySessionID, err := s.auth.BeginAccountRecovery(ctx, resolvedUsername, resolvedRecoveryCode)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	resolvedRecoverySessionID, err := requireGatewaySessionID(recoverySessionID)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	challenge, err := s.auth.BeginRecoveryPasskeyRegistration(ctx, resolvedRecoverySessionID)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	resolvedPasskeySessionID, err := requireGatewaySessionID(challenge.SessionID)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	return RecoveryChallenge{
		RecoverySessionID: resolvedRecoverySessionID,
		SessionID:         resolvedPasskeySessionID,
		PublicKey:         challenge.PublicKey,
	}, nil
}

// RecoveryFinish completes replacement passkey enrollment and returns the signed-in session.
func (s service) RecoveryFinish(ctx context.Context, recoverySessionID string, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error) {
	resolvedRecoverySessionID, err := requireSessionID(recoverySessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	if err := requireCredential(credential); err != nil {
		return PasskeyFinish{}, err
	}
	finished, err := s.auth.FinishRecoveryPasskeyRegistration(ctx, resolvedRecoverySessionID, resolvedSessionID, credential, strings.TrimSpace(pendingID))
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished.UserID, err = requireGatewayUserID(finished.UserID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished.SessionID, err = requireGatewaySessionID(finished.SessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished.RecoveryCode, err = requireRecoveryCode(finished.RecoveryCode)
	if err != nil {
		return PasskeyFinish{}, err
	}
	return finished, nil
}

// ResolvePostAuthRedirect returns the app dashboard or auth consent URL.
func (s service) ResolvePostAuthRedirect(pendingID string) string {
	pendingID = strings.TrimSpace(pendingID)
	if pendingID == "" {
		return routepath.AppDashboard
	}
	base := strings.TrimRight(strings.TrimSpace(s.authBaseURL), "/")
	if base == "" {
		return routepath.AppDashboard
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return routepath.AppDashboard
	}
	parsed.Path = path.Clean(strings.TrimRight(parsed.Path, "/") + "/authorize/consent")
	if parsed.Path == "." {
		parsed.Path = "/authorize/consent"
	}
	query := parsed.Query()
	query.Set("pending_id", pendingID)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// HasValidWebSession trims cookie input before delegating to auth session checks.
func (s service) HasValidWebSession(ctx context.Context, sessionID string) bool {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return false
	}
	return s.auth.HasValidWebSession(ctx, resolvedSessionID)
}

// RevokeWebSession treats blank cookie values as already-cleared sessions.
func (s service) RevokeWebSession(ctx context.Context, sessionID string) error {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return nil
	}
	return s.auth.RevokeWebSession(ctx, resolvedSessionID)
}

// requireSessionID rejects empty ceremony and cookie session identifiers early.
func requireSessionID(sessionID string) (string, error) {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return "", apperrors.E(apperrors.KindInvalidInput, "Session ID is required.")
	}
	return resolvedSessionID, nil
}

// requireCredential ensures the browser supplied a WebAuthn response payload.
func requireCredential(credential json.RawMessage) error {
	if len(credential) == 0 {
		return apperrors.E(apperrors.KindInvalidInput, "Credential is required.")
	}
	return nil
}

// requireUsername rejects blank account locators before calling auth.
func requireUsername(username string) (string, error) {
	resolvedUsername := strings.TrimSpace(username)
	if resolvedUsername == "" {
		return "", apperrors.E(apperrors.KindInvalidInput, "Username is required.")
	}
	return resolvedUsername, nil
}

// requireRecoveryCode rejects blank recovery credentials before calling auth.
func requireRecoveryCode(recoveryCode string) (string, error) {
	resolvedRecoveryCode := strings.TrimSpace(recoveryCode)
	if resolvedRecoveryCode == "" {
		return "", apperrors.E(apperrors.KindInvalidInput, "Recovery code is required.")
	}
	return resolvedRecoveryCode, nil
}

// requireGatewayUserID protects the web tier from empty auth responses.
func requireGatewayUserID(userID string) (string, error) {
	resolvedUserID := userid.Normalize(userID)
	if resolvedUserID == "" {
		return "", apperrors.E(apperrors.KindUnavailable, "Auth gateway user ID is unavailable.")
	}
	return resolvedUserID, nil
}

// requireGatewaySessionID protects the web tier from empty auth session output.
func requireGatewaySessionID(sessionID string) (string, error) {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return "", apperrors.E(apperrors.KindUnavailable, "Auth gateway session ID is unavailable.")
	}
	return resolvedSessionID, nil
}
