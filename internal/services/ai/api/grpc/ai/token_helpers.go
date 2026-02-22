package ai

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type providerTokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func decodeProviderTokenPayload(tokenPlaintext string) (providerTokenPayload, error) {
	tokenPlaintext = strings.TrimSpace(tokenPlaintext)
	if tokenPlaintext == "" {
		return providerTokenPayload{}, fmt.Errorf("token payload is empty")
	}
	var payload providerTokenPayload
	if err := json.Unmarshal([]byte(tokenPlaintext), &payload); err != nil {
		return providerTokenPayload{}, fmt.Errorf("decode provider token payload: %w", err)
	}
	payload.AccessToken = strings.TrimSpace(payload.AccessToken)
	payload.RefreshToken = strings.TrimSpace(payload.RefreshToken)
	return payload, nil
}

func refreshTokenFromTokenPayload(tokenPlaintext string) (string, error) {
	payload, err := decodeProviderTokenPayload(tokenPlaintext)
	if err != nil {
		return "", err
	}
	refreshToken := strings.TrimSpace(payload.RefreshToken)
	if refreshToken == "" {
		return "", fmt.Errorf("refresh token is unavailable")
	}
	return refreshToken, nil
}

// accessTokenFromTokenPayload extracts only the provider access token used for
// invocation and avoids leaking unrelated token payload fields downstream.
func accessTokenFromTokenPayload(tokenPlaintext string) (string, error) {
	payload, err := decodeProviderTokenPayload(tokenPlaintext)
	if err != nil {
		return "", err
	}
	accessToken := strings.TrimSpace(payload.AccessToken)
	if accessToken == "" {
		return "", fmt.Errorf("access token is unavailable")
	}
	return accessToken, nil
}

func revokeTokenFromTokenPayload(tokenPlaintext string) (string, error) {
	payload, err := decodeProviderTokenPayload(tokenPlaintext)
	if err == nil {
		token := firstNonEmpty(payload.RefreshToken, payload.AccessToken)
		if token == "" {
			return "", fmt.Errorf("token payload is missing revoke token")
		}
		return token, nil
	}
	token := strings.TrimSpace(tokenPlaintext)
	if token == "" {
		return "", fmt.Errorf("token payload is empty")
	}
	return token, nil
}

func normalizeScopes(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	scopes := make([]string, 0, len(values))
	for _, value := range values {
		scope := strings.TrimSpace(value)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}
	if len(scopes) == 0 {
		return nil
	}
	return scopes
}

// generatePKCECodeVerifier returns an RFC 7636-compliant verifier string with
// cryptographic entropy suitable for S256 code challenge derivation.
func generatePKCECodeVerifier() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("read pkce entropy: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func pkceCodeChallengeS256(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func isValidPKCECodeVerifier(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-', r == '.', r == '_', r == '~':
		default:
			return false
		}
	}
	return true
}

func hashState(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}
