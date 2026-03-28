package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
)

type defaultOpenAIOAuthAdapter struct{}

func newDefaultOpenAIOAuthAdapter() provideroauth.Adapter {
	return defaultOpenAIOAuthAdapter{}
}

func (defaultOpenAIOAuthAdapter) BuildAuthorizationURL(input provideroauth.AuthorizationURLInput) (string, error) {
	return fmt.Sprintf("https://oauth.fracturing.space/openai?state=%s", strings.TrimSpace(input.State)), nil
}

func (defaultOpenAIOAuthAdapter) ExchangeAuthorizationCode(_ context.Context, input provideroauth.AuthorizationCodeInput) (provideroauth.TokenExchangeResult, error) {
	code := strings.TrimSpace(input.AuthorizationCode)
	if code == "" {
		return provideroauth.TokenExchangeResult{}, fmt.Errorf("authorization code is required")
	}
	return provideroauth.TokenExchangeResult{
		TokenPayload: provideroauth.TokenPayload{
			AccessToken:  "token:" + code,
			RefreshToken: "refresh:" + code,
		},
	}, nil
}

func (defaultOpenAIOAuthAdapter) RefreshToken(_ context.Context, input provideroauth.RefreshTokenInput) (provideroauth.TokenExchangeResult, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return provideroauth.TokenExchangeResult{}, fmt.Errorf("refresh token is required")
	}
	return provideroauth.TokenExchangeResult{
		TokenPayload: provideroauth.TokenPayload{
			AccessToken:  "token:refresh:" + refreshToken,
			RefreshToken: "refresh:" + refreshToken,
		},
	}, nil
}
