package ai

import (
	"context"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
)

type fakeProviderOAuthAdapter struct {
	buildAuthorizationURLErr error
	exchangeErr              error
	exchangeResult           provideroauth.TokenExchangeResult
	refreshErr               error
	refreshResult            provideroauth.TokenExchangeResult
	revokeErr                error

	lastAuthorizationInput provideroauth.AuthorizationURLInput
	lastRefreshToken       string
	lastRevokedToken       string
}

func (f *fakeProviderOAuthAdapter) BuildAuthorizationURL(input provideroauth.AuthorizationURLInput) (string, error) {
	f.lastAuthorizationInput = input
	if f.buildAuthorizationURLErr != nil {
		return "", f.buildAuthorizationURLErr
	}
	return "https://provider.example.com/auth", nil
}

func (f *fakeProviderOAuthAdapter) ExchangeAuthorizationCode(_ context.Context, _ provideroauth.AuthorizationCodeInput) (provideroauth.TokenExchangeResult, error) {
	if f.exchangeErr != nil {
		return provideroauth.TokenExchangeResult{}, f.exchangeErr
	}
	if strings.TrimSpace(f.exchangeResult.TokenPayload.AccessToken) == "" {
		return provideroauth.TokenExchangeResult{
			TokenPayload: provideroauth.TokenPayload{AccessToken: "at-1", RefreshToken: "rt-1"},
		}, nil
	}
	return f.exchangeResult, nil
}

func (f *fakeProviderOAuthAdapter) RefreshToken(_ context.Context, input provideroauth.RefreshTokenInput) (provideroauth.TokenExchangeResult, error) {
	f.lastRefreshToken = input.RefreshToken
	if f.refreshErr != nil {
		return provideroauth.TokenExchangeResult{}, f.refreshErr
	}
	return f.refreshResult, nil
}

func (f *fakeProviderOAuthAdapter) RevokeToken(_ context.Context, input provideroauth.RevokeTokenInput) error {
	f.lastRevokedToken = input.Token
	return f.revokeErr
}

type defaultProviderOAuthAdapterForTests struct{}

func (d *defaultProviderOAuthAdapterForTests) BuildAuthorizationURL(input provideroauth.AuthorizationURLInput) (string, error) {
	return "https://oauth.fracturing.space/openai?state=" + strings.TrimSpace(input.State), nil
}

func (d *defaultProviderOAuthAdapterForTests) ExchangeAuthorizationCode(_ context.Context, input provideroauth.AuthorizationCodeInput) (provideroauth.TokenExchangeResult, error) {
	code := strings.TrimSpace(input.AuthorizationCode)
	if code == "" {
		return provideroauth.TokenExchangeResult{}, errors.New("authorization code is required")
	}
	return provideroauth.TokenExchangeResult{
		TokenPayload: provideroauth.TokenPayload{
			AccessToken:  "token:" + code,
			RefreshToken: "refresh:" + code,
		},
	}, nil
}

func (d *defaultProviderOAuthAdapterForTests) RefreshToken(_ context.Context, input provideroauth.RefreshTokenInput) (provideroauth.TokenExchangeResult, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return provideroauth.TokenExchangeResult{}, errors.New("refresh token is required")
	}
	return provideroauth.TokenExchangeResult{
		TokenPayload: provideroauth.TokenPayload{
			AccessToken:  "token:refresh:" + refreshToken,
			RefreshToken: "refresh:" + refreshToken,
		},
	}, nil
}

type fakeProviderInvocationAdapter struct {
	invokeErr           error
	invokeResult        provider.InvokeResult
	lastInput           provider.InvokeInput
	listModelsErr       error
	listModelsResult    []provider.Model
	lastListModelsInput provider.ListModelsInput
}

func (f *fakeProviderInvocationAdapter) Invoke(_ context.Context, input provider.InvokeInput) (provider.InvokeResult, error) {
	f.lastInput = input
	if f.invokeErr != nil {
		return provider.InvokeResult{}, f.invokeErr
	}
	return f.invokeResult, nil
}

func (f *fakeProviderInvocationAdapter) ListModels(_ context.Context, input provider.ListModelsInput) ([]provider.Model, error) {
	f.lastListModelsInput = input
	if f.listModelsErr != nil {
		return nil, f.listModelsErr
	}
	if f.listModelsResult == nil {
		return []provider.Model{
			{ID: "gpt-4o-mini"},
			{ID: "gpt-4o"},
		}, nil
	}
	return f.listModelsResult, nil
}
