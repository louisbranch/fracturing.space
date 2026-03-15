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

// pageService owns page-only public auth behavior.
type pageService struct {
	authBaseURL string
}

// sessionService owns session-backed redirect and logout behavior.
type sessionService struct {
	session     SessionGateway
	authBaseURL string
}

// passkeyService owns username, login, and signup ceremonies.
type passkeyService struct {
	passkeys PasskeyGateway
}

// recoveryService owns account recovery ceremonies.
type recoveryService struct {
	recovery RecoveryGateway
}

// NewPageService wires page-only public auth flows behind input validation.
func NewPageService(authBaseURL string) PageService {
	return pageService{authBaseURL: normalizeAuthBaseURL(authBaseURL)}
}

// NewSessionService wires session-only public auth flows behind input validation.
func NewSessionService(gateway SessionGateway, authBaseURL string) SessionService {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return sessionService{
		session:     gateway,
		authBaseURL: normalizeAuthBaseURL(authBaseURL),
	}
}

// NewPasskeyService wires passkey-only public auth flows behind input validation.
func NewPasskeyService(gateway PasskeyGateway, authBaseURL string) PasskeyService {
	_ = authBaseURL
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return passkeyService{passkeys: gateway}
}

// NewRecoveryService wires recovery-only public auth flows behind input validation.
func NewRecoveryService(gateway RecoveryGateway, authBaseURL string) RecoveryService {
	_ = authBaseURL
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return recoveryService{recovery: gateway}
}

// HealthBody returns the plain-text health response expected by the endpoint.
func (pageService) HealthBody() string {
	return "ok"
}

// CheckUsernameAvailability returns advisory live validation state for signup.
func (s passkeyService) CheckUsernameAvailability(ctx context.Context, username string) (UsernameAvailability, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return UsernameAvailability{State: UsernameAvailabilityStateInvalid}, nil
	}
	return s.passkeys.CheckUsernameAvailability(ctx, username)
}

// PasskeyLoginStart normalizes the username before asking auth to begin login.
func (s passkeyService) PasskeyLoginStart(ctx context.Context, username string) (PasskeyChallenge, error) {
	resolvedUsername, err := requireUsername(username)
	if err != nil {
		return PasskeyChallenge{}, err
	}
	return s.passkeys.BeginPasskeyLogin(ctx, resolvedUsername)
}

// PasskeyLoginFinish validates the ceremony response, then creates a web session.
func (s passkeyService) PasskeyLoginFinish(ctx context.Context, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error) {
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
func (s passkeyService) PasskeyRegisterStart(ctx context.Context, username string) (PasskeyRegisterResult, error) {
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
func (s passkeyService) PasskeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyRegistrationReveal, error) {
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
func (s passkeyService) PasskeyRegisterAcknowledge(ctx context.Context, sessionID string, pendingID string) (PasskeyFinish, error) {
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
func (s recoveryService) RecoveryStart(ctx context.Context, username string, recoveryCode string) (RecoveryChallenge, error) {
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
func (s recoveryService) RecoveryFinish(ctx context.Context, recoverySessionID string, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error) {
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

// ResolvePostAuthRedirect returns the auth consent URL, validated continuation,
// or dashboard.
func (s pageService) ResolvePostAuthRedirect(pendingID string, nextPath string) string {
	return resolvePostAuthRedirect(s.authBaseURL, pendingID, nextPath)
}

// ResolvePostAuthRedirect returns the auth consent URL, validated continuation,
// or dashboard.
func (s sessionService) ResolvePostAuthRedirect(pendingID string, nextPath string) string {
	return resolvePostAuthRedirect(s.authBaseURL, pendingID, nextPath)
}

// resolvePostAuthRedirect centralizes the validated post-auth landing decision
// shared by page-only and session-backed public auth flows.
func resolvePostAuthRedirect(authBaseURL string, pendingID string, nextPath string) string {
	pendingID = strings.TrimSpace(pendingID)
	if pendingID == "" {
		if resolved := redirectpath.ResolveSafe(nextPath); resolved != "" {
			return resolved
		}
		return routepath.AppDashboard
	}
	base := strings.TrimRight(authBaseURL, "/")
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
func (s sessionService) RevokeWebSession(ctx context.Context, sessionID string) error {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return nil
	}
	return s.session.RevokeWebSession(ctx, resolvedSessionID)
}

// normalizeAuthBaseURL trims composition input once so redirect building does
// not need to repeat auth-base normalization at each call site.
func normalizeAuthBaseURL(authBaseURL string) string {
	return strings.TrimSpace(authBaseURL)
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
