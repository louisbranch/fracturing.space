package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
)

// OAuthConfig configures OpenAI OAuth endpoints and credentials.
type OAuthConfig struct {
	AuthorizationURL string
	TokenURL         string
	ClientID         string
	ClientSecret     string
	RedirectURI      string
	HTTPClient       *http.Client
	Now              func() time.Time
}

type oauthAdapter struct {
	cfg OAuthConfig
}

type tokenResponsePayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
}

// NewOAuthAdapter builds an OpenAI OAuth adapter using HTTP token exchange.
func NewOAuthAdapter(cfg OAuthConfig) provideroauth.Adapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &oauthAdapter{cfg: cfg}
}

func (a *oauthAdapter) BuildAuthorizationURL(input provideroauth.AuthorizationURLInput) (string, error) {
	authURL := strings.TrimSpace(a.cfg.AuthorizationURL)
	clientID := strings.TrimSpace(a.cfg.ClientID)
	redirectURI := strings.TrimSpace(a.cfg.RedirectURI)
	state := strings.TrimSpace(input.State)
	challenge := strings.TrimSpace(input.CodeChallenge)
	if authURL == "" || clientID == "" || redirectURI == "" || state == "" || challenge == "" {
		return "", fmt.Errorf("authorization url configuration is incomplete")
	}

	u, err := url.Parse(authURL)
	if err != nil {
		return "", fmt.Errorf("parse authorization url: %w", err)
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	scopes := strings.TrimSpace(strings.Join(provideroauth.NormalizeScopes(input.RequestedScopes), " "))
	if scopes != "" {
		q.Set("scope", scopes)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (a *oauthAdapter) ExchangeAuthorizationCode(ctx context.Context, input provideroauth.AuthorizationCodeInput) (provideroauth.TokenExchangeResult, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(input.AuthorizationCode))
	form.Set("code_verifier", strings.TrimSpace(input.CodeVerifier))
	form.Set("client_id", strings.TrimSpace(a.cfg.ClientID))
	form.Set("client_secret", strings.TrimSpace(a.cfg.ClientSecret))
	form.Set("redirect_uri", strings.TrimSpace(a.cfg.RedirectURI))
	return a.tokenRequest(ctx, form)
}

func (a *oauthAdapter) RefreshToken(ctx context.Context, input provideroauth.RefreshTokenInput) (provideroauth.TokenExchangeResult, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", strings.TrimSpace(input.RefreshToken))
	form.Set("client_id", strings.TrimSpace(a.cfg.ClientID))
	form.Set("client_secret", strings.TrimSpace(a.cfg.ClientSecret))
	return a.tokenRequest(ctx, form)
}

func (a *oauthAdapter) tokenRequest(ctx context.Context, form url.Values) (provideroauth.TokenExchangeResult, error) {
	tokenURL := strings.TrimSpace(a.cfg.TokenURL)
	if tokenURL == "" {
		return provideroauth.TokenExchangeResult{}, fmt.Errorf("token url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return provideroauth.TokenExchangeResult{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return provideroauth.TokenExchangeResult{}, fmt.Errorf("token request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return provideroauth.TokenExchangeResult{}, fmt.Errorf("read token error body: %w", err)
		}
		return provideroauth.TokenExchangeResult{}, fmt.Errorf("token request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload tokenResponsePayload
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return provideroauth.TokenExchangeResult{}, fmt.Errorf("decode token response: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return provideroauth.TokenExchangeResult{}, fmt.Errorf("token response missing access_token")
	}

	var expiresAt *time.Time
	if payload.ExpiresIn > 0 {
		exp := a.cfg.Now().UTC().Add(time.Duration(payload.ExpiresIn) * time.Second)
		expiresAt = &exp
	}

	return provideroauth.TokenExchangeResult{
		TokenPayload: provideroauth.TokenPayload{
			AccessToken:  payload.AccessToken,
			RefreshToken: payload.RefreshToken,
			TokenType:    payload.TokenType,
			Scope:        payload.Scope,
		},
		ExpiresAt: expiresAt,
	}, nil
}
