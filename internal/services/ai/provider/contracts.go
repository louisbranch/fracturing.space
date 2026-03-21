package provider

import (
	"context"
	"time"
)

// OAuthAdapter handles provider-specific OAuth URL/token exchange logic.
type OAuthAdapter interface {
	BuildAuthorizationURL(input AuthorizationURLInput) (string, error)
	ExchangeAuthorizationCode(ctx context.Context, input AuthorizationCodeInput) (TokenExchangeResult, error)
	RefreshToken(ctx context.Context, input RefreshTokenInput) (TokenExchangeResult, error)
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
	TokenPlaintext   string
	RefreshSupported bool
	ExpiresAt        *time.Time
	LastRefreshError string
}

// InvocationAdapter handles provider-specific inference invocation.
type InvocationAdapter interface {
	Invoke(ctx context.Context, input InvokeInput) (InvokeResult, error)
}

// ModelAdapter handles provider-backed model discovery.
type ModelAdapter interface {
	ListModels(ctx context.Context, input ListModelsInput) ([]Model, error)
}

// InvokeInput contains provider invocation input fields.
type InvokeInput struct {
	Model           string
	Input           string
	Instructions    string
	ReasoningEffort string
	// CredentialSecret is decrypted only at call-time and must never be logged.
	CredentialSecret string
}

// InvokeResult contains invocation output.
type InvokeResult struct {
	OutputText string
	Usage      Usage
}

// ListModelsInput contains provider model-listing input fields.
type ListModelsInput struct {
	// CredentialSecret is decrypted only at call-time and must never be logged.
	CredentialSecret string
}

// Model contains one provider model option.
type Model struct {
	ID      string
	OwnedBy string
	Created int64
}
