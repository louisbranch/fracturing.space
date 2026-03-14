package app

import (
	"context"
	"encoding/json"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// ListPasskeys returns read-only passkey summaries for the security page.
func (s service) ListPasskeys(ctx context.Context, userID string) ([]SettingsPasskey, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return nil, err
	}
	passkeys, err := s.gateway.ListPasskeys(ctx, resolvedUserID)
	if err != nil {
		return nil, err
	}
	if passkeys == nil {
		return []SettingsPasskey{}, nil
	}
	normalized := make([]SettingsPasskey, 0, len(passkeys))
	for _, passkey := range passkeys {
		normalized = append(normalized, normalizeSettingsPasskey(passkey))
	}
	return normalized, nil
}

// BeginPasskeyRegistration starts authenticated passkey enrollment for the current user.
func (s service) BeginPasskeyRegistration(ctx context.Context, userID string) (PasskeyChallenge, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return PasskeyChallenge{}, err
	}
	challenge, err := s.gateway.BeginPasskeyRegistration(ctx, resolvedUserID)
	if err != nil {
		return PasskeyChallenge{}, err
	}
	challenge.SessionID = strings.TrimSpace(challenge.SessionID)
	if challenge.SessionID == "" {
		return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, "passkey session is unavailable")
	}
	return challenge, nil
}

// FinishPasskeyRegistration completes authenticated passkey enrollment.
func (s service) FinishPasskeyRegistration(ctx context.Context, sessionID string, credential json.RawMessage) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "session id is required")
	}
	if len(credential) == 0 {
		return apperrors.E(apperrors.KindInvalidInput, "credential is required")
	}
	return s.gateway.FinishPasskeyRegistration(ctx, sessionID, credential)
}

// normalizeSettingsPasskey normalizes one passkey row for stable rendering.
func normalizeSettingsPasskey(passkey SettingsPasskey) SettingsPasskey {
	if passkey.Number <= 0 {
		passkey.Number = 1
	}
	passkey.CreatedAt = strings.TrimSpace(passkey.CreatedAt)
	passkey.LastUsedAt = strings.TrimSpace(passkey.LastUsedAt)
	if passkey.CreatedAt == "" {
		passkey.CreatedAt = "-"
	}
	if passkey.LastUsedAt == "" {
		passkey.LastUsedAt = "-"
	}
	return passkey
}
