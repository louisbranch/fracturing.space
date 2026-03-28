package provideroauth

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TokenPayload is the structured provider token material exchanged and
// refreshed through OAuth. It is encoded once before ciphertext persistence.
type TokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// NormalizeTokenPayload trims token-payload fields and validates required data.
func NormalizeTokenPayload(payload TokenPayload) (TokenPayload, error) {
	payload.AccessToken = strings.TrimSpace(payload.AccessToken)
	payload.RefreshToken = strings.TrimSpace(payload.RefreshToken)
	payload.TokenType = strings.TrimSpace(payload.TokenType)
	payload.Scope = strings.TrimSpace(payload.Scope)
	if payload.AccessToken == "" {
		return TokenPayload{}, fmt.Errorf("access token is unavailable")
	}
	return payload, nil
}

// EncodeTokenPayload serializes a normalized provider token payload.
func EncodeTokenPayload(payload TokenPayload) (string, error) {
	normalized, err := NormalizeTokenPayload(payload)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("encode provider token payload: %w", err)
	}
	return string(raw), nil
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
	return NormalizeTokenPayload(payload)
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

// AccessTokenFromPayload extracts the access token used for provider calls.
func AccessTokenFromPayload(tokenPlaintext string) (string, error) {
	payload, err := DecodeTokenPayload(tokenPlaintext)
	if err != nil {
		return "", err
	}
	return payload.AccessToken, nil
}

// RevokeTokenFromPayload extracts the best available token for revocation,
// preferring refresh over access.
func RevokeTokenFromPayload(tokenPlaintext string) (string, error) {
	payload, err := DecodeTokenPayload(tokenPlaintext)
	if err != nil {
		return "", err
	}
	switch {
	case payload.RefreshToken != "":
		return payload.RefreshToken, nil
	case payload.AccessToken != "":
		return payload.AccessToken, nil
	default:
		return "", fmt.Errorf("token payload is missing revoke token")
	}
}
