package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

type defaultOpenAIOAuthAdapter struct{}

func newDefaultOpenAIOAuthAdapter() provider.OAuthAdapter {
	return defaultOpenAIOAuthAdapter{}
}

func (defaultOpenAIOAuthAdapter) BuildAuthorizationURL(input provider.AuthorizationURLInput) (string, error) {
	return fmt.Sprintf("https://oauth.fracturing.space/openai?state=%s", strings.TrimSpace(input.State)), nil
}

func (defaultOpenAIOAuthAdapter) ExchangeAuthorizationCode(_ context.Context, input provider.AuthorizationCodeInput) (provider.TokenExchangeResult, error) {
	code := strings.TrimSpace(input.AuthorizationCode)
	if code == "" {
		return provider.TokenExchangeResult{}, fmt.Errorf("authorization code is required")
	}
	return provider.TokenExchangeResult{
		TokenPlaintext:   "token:" + code,
		RefreshSupported: true,
	}, nil
}

func (defaultOpenAIOAuthAdapter) RefreshToken(_ context.Context, input provider.RefreshTokenInput) (provider.TokenExchangeResult, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return provider.TokenExchangeResult{}, fmt.Errorf("refresh token is required")
	}
	return provider.TokenExchangeResult{
		TokenPlaintext:   "token:refresh:" + refreshToken,
		RefreshSupported: true,
	}, nil
}

func (defaultOpenAIOAuthAdapter) RevokeToken(_ context.Context, input provider.RevokeTokenInput) error {
	if strings.TrimSpace(input.Token) == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}
