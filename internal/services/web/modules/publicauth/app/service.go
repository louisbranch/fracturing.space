package app

import (
	"context"
	"encoding/json"
	"net/url"
	"path"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/redirectpath"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// service centralizes public auth orchestration so handlers stay transport-focused.
type service struct {
	session     SessionGateway
	passkeys    PasskeyGateway
	recovery    RecoveryGateway
	authBaseURL string
}

// serviceConfig keeps ceremony-owned auth dependencies explicit.
type serviceConfig struct {
	SessionGateway  SessionGateway
	PasskeyGateway  PasskeyGateway
	RecoveryGateway RecoveryGateway
	AuthBaseURL     string
}

// newServiceState wires auth-backed public auth flows behind input validation.
func newServiceState(config serviceConfig) service {
	sessionGateway := config.SessionGateway
	if sessionGateway == nil {
		sessionGateway = unavailableGateway{}
	}
	passkeyGateway := config.PasskeyGateway
	if passkeyGateway == nil {
		passkeyGateway = unavailableGateway{}
	}
	recoveryGateway := config.RecoveryGateway
	if recoveryGateway == nil {
		recoveryGateway = unavailableGateway{}
	}
	return service{
		session:     sessionGateway,
		passkeys:    passkeyGateway,
		recovery:    recoveryGateway,
		authBaseURL: strings.TrimSpace(config.AuthBaseURL),
	}
}

// NewPageService wires page-only public auth flows behind input validation.
func NewPageService(authBaseURL string) PageService {
	return newServiceState(serviceConfig{AuthBaseURL: authBaseURL})
}

// NewSessionService wires session-only public auth flows behind input validation.
func NewSessionService(gateway SessionGateway, authBaseURL string) SessionService {
	return newServiceState(serviceConfig{
		SessionGateway: gateway,
		AuthBaseURL:    authBaseURL,
	})
}

// NewPasskeyService wires passkey-only public auth flows behind input validation.
func NewPasskeyService(gateway PasskeyGateway, authBaseURL string) PasskeyService {
	return newServiceState(serviceConfig{
		PasskeyGateway: gateway,
		AuthBaseURL:    authBaseURL,
	})
}

// NewRecoveryService wires recovery-only public auth flows behind input validation.
func NewRecoveryService(gateway RecoveryGateway, authBaseURL string) RecoveryService {
	return newServiceState(serviceConfig{
		RecoveryGateway: gateway,
		AuthBaseURL:     authBaseURL,
	})
}

// HealthBody returns the plain-text health response expected by the endpoint.
func (service) HealthBody() string {
	return "ok"
}

// CheckUsernameAvailability returns advisory live validation state for signup.
func (s service) CheckUsernameAvailability(ctx context.Context, username string) (UsernameAvailability, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return UsernameAvailability{State: UsernameAvailabilityStateInvalid}, nil
	}
	return s.passkeys.CheckUsernameAvailability(ctx, username)
}

// PasskeyLoginStart normalizes the username before asking auth to begin login.
func (s service) PasskeyLoginStart(ctx context.Context, username string) (PasskeyChallenge, error) {
	resolvedUsername, err := requireUsername(username)
	if err != nil {
		return PasskeyChallenge{}, err
	}
	return s.passkeys.BeginPasskeyLogin(ctx, resolvedUsername)
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
	userID, err := s.passkeys.FinishPasskeyLogin(ctx, resolvedSessionID, credential, strings.TrimSpace(pendingID))
	if err != nil {
		return PasskeyFinish{}, err
	}
	resolvedUserID, err := requireGatewayUserID(userID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	webSessionID, err := s.passkeys.CreateWebSession(ctx, resolvedUserID)
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
	challenge, err := s.passkeys.BeginAccountRegistration(ctx, resolvedUsername)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	resolvedChallengeSessionID, err := requireGatewaySessionID(challenge.SessionID)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	return PasskeyRegisterResult{SessionID: resolvedChallengeSessionID, PublicKey: challenge.PublicKey}, nil
}

// PasskeyRegisterFinish stages signup and normalizes the recovery-code reveal.
func (s service) PasskeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyRegistrationReveal, error) {
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyRegistrationReveal{}, err
	}
	if err := requireCredential(credential); err != nil {
		return PasskeyRegistrationReveal{}, err
	}
	finished, err := s.passkeys.FinishAccountRegistration(ctx, resolvedSessionID, credential)
	if err != nil {
		return PasskeyRegistrationReveal{}, err
	}
	finished.RecoveryCode, err = requireRecoveryCode(finished.RecoveryCode)
	if err != nil {
		return PasskeyRegistrationReveal{}, err
	}
	return finished, nil
}

// PasskeyRegisterAcknowledge activates one staged signup and returns the signed-in session.
func (s service) PasskeyRegisterAcknowledge(ctx context.Context, sessionID string, pendingID string) (PasskeyFinish, error) {
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished, err := s.passkeys.AcknowledgeAccountRegistration(ctx, resolvedSessionID, strings.TrimSpace(pendingID))
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
	recoverySessionID, err := s.recovery.BeginAccountRecovery(ctx, resolvedUsername, resolvedRecoveryCode)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	resolvedRecoverySessionID, err := requireGatewaySessionID(recoverySessionID)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	challenge, err := s.recovery.BeginRecoveryPasskeyRegistration(ctx, resolvedRecoverySessionID)
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
	finished, err := s.recovery.FinishRecoveryPasskeyRegistration(ctx, resolvedRecoverySessionID, resolvedSessionID, credential, strings.TrimSpace(pendingID))
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

// ResolvePostAuthRedirect returns the auth consent URL, validated continuation, or dashboard.
func (s service) ResolvePostAuthRedirect(pendingID string, nextPath string) string {
	pendingID = strings.TrimSpace(pendingID)
	if pendingID == "" {
		if resolved := redirectpath.ResolveSafe(nextPath); resolved != "" {
			return resolved
		}
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

// RevokeWebSession treats blank cookie values as already-cleared sessions.
func (s service) RevokeWebSession(ctx context.Context, sessionID string) error {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return nil
	}
	return s.session.RevokeWebSession(ctx, resolvedSessionID)
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
