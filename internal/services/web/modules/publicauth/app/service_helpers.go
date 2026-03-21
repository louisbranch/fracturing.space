package app

import (
	"encoding/json"
	"net/url"
	"path"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/redirectpath"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// normalizeAuthBaseURL trims composition input once so redirect building does
// not need to repeat auth-base normalization at each call site.
func normalizeAuthBaseURL(authBaseURL string) string {
	return strings.TrimSpace(authBaseURL)
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
