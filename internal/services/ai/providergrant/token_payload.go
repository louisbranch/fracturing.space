package providergrant

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TokenPayload represents the structured token pair stored as ciphertext in a
// provider grant. The access token is used for API calls; the refresh token is
// used for automatic renewal.
type TokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// DecodeTokenPayload deserializes a plaintext token payload.
func DecodeTokenPayload(tokenPlaintext string) (TokenPayload, error) {
	tokenPlaintext = strings.TrimSpace(tokenPlaintext)
	if tokenPlaintext == "" {
		return TokenPayload{}, fmt.Errorf("token payload is empty")
	}
	var payload TokenPayload
	if err := json.Unmarshal([]byte(tokenPlaintext), &payload); err != nil {
		return TokenPayload{}, fmt.Errorf("decode provider token payload: %w", err)
	}
	payload.AccessToken = strings.TrimSpace(payload.AccessToken)
	payload.RefreshToken = strings.TrimSpace(payload.RefreshToken)
	return payload, nil
}

// RefreshTokenFromPayload extracts the refresh token from a plaintext payload.
func RefreshTokenFromPayload(tokenPlaintext string) (string, error) {
	payload, err := DecodeTokenPayload(tokenPlaintext)
	if err != nil {
		return "", err
	}
	if payload.RefreshToken == "" {
		return "", fmt.Errorf("refresh token is unavailable")
	}
	return payload.RefreshToken, nil
}

// AccessTokenFromPayload extracts only the provider access token used for
// invocation and avoids leaking unrelated token payload fields downstream.
func AccessTokenFromPayload(tokenPlaintext string) (string, error) {
	payload, err := DecodeTokenPayload(tokenPlaintext)
	if err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("access token is unavailable")
	}
	return payload.AccessToken, nil
}

// RevokeTokenFromPayload extracts the best available token for revocation,
// preferring refresh over access. Falls back to the raw plaintext if the
// payload is not JSON-structured.
func RevokeTokenFromPayload(tokenPlaintext string) (string, error) {
	payload, err := DecodeTokenPayload(tokenPlaintext)
	if err == nil {
		switch {
		case payload.RefreshToken != "":
			return payload.RefreshToken, nil
		case payload.AccessToken != "":
			return payload.AccessToken, nil
		default:
			return "", fmt.Errorf("token payload is missing revoke token")
		}
	}
	token := strings.TrimSpace(tokenPlaintext)
	if token == "" {
		return "", fmt.Errorf("token payload is empty")
	}
	return token, nil
}
