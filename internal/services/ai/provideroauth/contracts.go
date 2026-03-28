package provideroauth

import (
	"context"
	"time"
)

// Adapter handles provider-specific OAuth URL/token exchange logic.
type Adapter interface {
	BuildAuthorizationURL(input AuthorizationURLInput) (string, error)
	ExchangeAuthorizationCode(ctx context.Context, input AuthorizationCodeInput) (TokenExchangeResult, error)
	RefreshToken(ctx context.Context, input RefreshTokenInput) (TokenExchangeResult, error)
}

// TokenRevoker optionally handles provider-side token revocation.
type TokenRevoker interface {
	RevokeToken(ctx context.Context, input RevokeTokenInput) error
}

// AuthorizationURLInput contains parameters for building a provider auth URL.
type AuthorizationURLInput struct {
	State           string
	CodeChallenge   string
	RequestedScopes []string
}

// AuthorizationCodeInput contains token-exchange input fields.
type AuthorizationCodeInput struct {
	AuthorizationCode string
	CodeVerifier      string
}

// RefreshTokenInput contains refresh-token input fields.
type RefreshTokenInput struct {
	RefreshToken string
}

// RevokeTokenInput contains token-revocation input fields.
type RevokeTokenInput struct {
	Token string
}

// TokenExchangeResult contains provider token exchange output.
type TokenExchangeResult struct {
	TokenPayload TokenPayload
	ExpiresAt    *time.Time
}
