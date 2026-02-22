package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type defaultOpenAIOAuthAdapter struct{}

// OpenAIOAuthConfig configures OpenAI OAuth endpoints and credentials.
type OpenAIOAuthConfig struct {
	AuthorizationURL string
	TokenURL         string
	ClientID         string
	ClientSecret     string
	RedirectURI      string
	HTTPClient       *http.Client
}

type openAIOAuthAdapter struct {
	cfg OpenAIOAuthConfig
}

// OpenAIInvokeConfig configures OpenAI responses endpoint and HTTP behavior.
type OpenAIInvokeConfig struct {
	ResponsesURL string
	HTTPClient   *http.Client
}

type openAIInvokeAdapter struct {
	cfg OpenAIInvokeConfig
}

// NewOpenAIOAuthAdapter builds an OpenAI OAuth adapter using HTTP token exchange.
func NewOpenAIOAuthAdapter(cfg OpenAIOAuthConfig) ProviderOAuthAdapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return &openAIOAuthAdapter{cfg: cfg}
}

// NewOpenAIInvokeAdapter builds an OpenAI invocation adapter.
func NewOpenAIInvokeAdapter(cfg OpenAIInvokeConfig) ProviderInvocationAdapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if strings.TrimSpace(cfg.ResponsesURL) == "" {
		cfg.ResponsesURL = "https://api.openai.com/v1/responses"
	}
	return &openAIInvokeAdapter{cfg: cfg}
}

func (a *defaultOpenAIOAuthAdapter) BuildAuthorizationURL(input ProviderAuthorizationURLInput) (string, error) {
	return fmt.Sprintf("https://oauth.fracturing.space/openai?state=%s", strings.TrimSpace(input.State)), nil
}

func (a *defaultOpenAIOAuthAdapter) ExchangeAuthorizationCode(_ context.Context, input ProviderAuthorizationCodeInput) (ProviderTokenExchangeResult, error) {
	code := strings.TrimSpace(input.AuthorizationCode)
	if code == "" {
		return ProviderTokenExchangeResult{}, fmt.Errorf("authorization code is required")
	}
	token := "token:" + code
	return ProviderTokenExchangeResult{
		TokenPlaintext:   token,
		RefreshSupported: true,
	}, nil
}

func (a *defaultOpenAIOAuthAdapter) RefreshToken(_ context.Context, input ProviderRefreshTokenInput) (ProviderTokenExchangeResult, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return ProviderTokenExchangeResult{}, fmt.Errorf("refresh token is required")
	}
	token := "token:refresh:" + refreshToken
	return ProviderTokenExchangeResult{
		TokenPlaintext:   token,
		RefreshSupported: true,
	}, nil
}

func (a *defaultOpenAIOAuthAdapter) RevokeToken(_ context.Context, input ProviderRevokeTokenInput) error {
	if strings.TrimSpace(input.Token) == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

func (a *openAIOAuthAdapter) BuildAuthorizationURL(input ProviderAuthorizationURLInput) (string, error) {
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
	scopes := strings.TrimSpace(strings.Join(normalizeScopes(input.RequestedScopes), " "))
	if scopes != "" {
		q.Set("scope", scopes)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (a *openAIOAuthAdapter) ExchangeAuthorizationCode(ctx context.Context, input ProviderAuthorizationCodeInput) (ProviderTokenExchangeResult, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(input.AuthorizationCode))
	form.Set("code_verifier", strings.TrimSpace(input.CodeVerifier))
	form.Set("client_id", strings.TrimSpace(a.cfg.ClientID))
	form.Set("client_secret", strings.TrimSpace(a.cfg.ClientSecret))
	form.Set("redirect_uri", strings.TrimSpace(a.cfg.RedirectURI))
	return a.tokenRequest(ctx, form)
}

func (a *openAIOAuthAdapter) RefreshToken(ctx context.Context, input ProviderRefreshTokenInput) (ProviderTokenExchangeResult, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", strings.TrimSpace(input.RefreshToken))
	form.Set("client_id", strings.TrimSpace(a.cfg.ClientID))
	form.Set("client_secret", strings.TrimSpace(a.cfg.ClientSecret))
	return a.tokenRequest(ctx, form)
}

func (a *openAIOAuthAdapter) RevokeToken(_ context.Context, input ProviderRevokeTokenInput) error {
	if strings.TrimSpace(input.Token) == "" {
		return fmt.Errorf("token is required")
	}
	// OpenAI revocation endpoint support is optional at this phase boundary.
	// Returning nil here avoids leaking token material into error/log paths.
	return nil
}

func (a *openAIInvokeAdapter) Invoke(ctx context.Context, input ProviderInvokeInput) (ProviderInvokeResult, error) {
	responsesURL := strings.TrimSpace(a.cfg.ResponsesURL)
	credentialSecret := strings.TrimSpace(input.CredentialSecret)
	model := strings.TrimSpace(input.Model)
	prompt := strings.TrimSpace(input.Input)
	if responsesURL == "" {
		return ProviderInvokeResult{}, fmt.Errorf("responses url is required")
	}
	if credentialSecret == "" {
		return ProviderInvokeResult{}, fmt.Errorf("credential secret is required")
	}
	if model == "" {
		return ProviderInvokeResult{}, fmt.Errorf("model is required")
	}
	if prompt == "" {
		return ProviderInvokeResult{}, fmt.Errorf("input is required")
	}

	requestBody, err := json.Marshal(map[string]any{
		"model": model,
		"input": prompt,
	})
	if err != nil {
		return ProviderInvokeResult{}, fmt.Errorf("marshal invoke request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responsesURL, bytes.NewReader(requestBody))
	if err != nil {
		return ProviderInvokeResult{}, fmt.Errorf("build invoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Credential material is sent only as an Authorization header and is never
	// echoed in errors or response payloads.
	req.Header.Set("Authorization", "Bearer "+credentialSecret)

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return ProviderInvokeResult{}, fmt.Errorf("invoke request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return ProviderInvokeResult{}, fmt.Errorf("read invoke error body: %w", err)
		}
		return ProviderInvokeResult{}, fmt.Errorf("invoke request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return ProviderInvokeResult{}, fmt.Errorf("decode invoke response: %w", err)
	}
	outputText := strings.TrimSpace(payload.OutputText)
	if outputText == "" {
		for _, item := range payload.Output {
			for _, content := range item.Content {
				if strings.TrimSpace(content.Text) != "" {
					outputText = strings.TrimSpace(content.Text)
					break
				}
			}
			if outputText != "" {
				break
			}
		}
	}
	if outputText == "" {
		return ProviderInvokeResult{}, fmt.Errorf("invoke response missing output text")
	}
	return ProviderInvokeResult{OutputText: outputText}, nil
}

func (a *openAIOAuthAdapter) tokenRequest(ctx context.Context, form url.Values) (ProviderTokenExchangeResult, error) {
	tokenURL := strings.TrimSpace(a.cfg.TokenURL)
	if tokenURL == "" {
		return ProviderTokenExchangeResult{}, fmt.Errorf("token url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return ProviderTokenExchangeResult{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return ProviderTokenExchangeResult{}, fmt.Errorf("token request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return ProviderTokenExchangeResult{}, fmt.Errorf("read token error body: %w", err)
		}
		return ProviderTokenExchangeResult{}, fmt.Errorf("token request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return ProviderTokenExchangeResult{}, fmt.Errorf("decode token response: %w", err)
	}

	accessToken := asString(payload["access_token"])
	refreshToken := asString(payload["refresh_token"])
	tokenType := asString(payload["token_type"])
	scope := asString(payload["scope"])
	if accessToken == "" {
		return ProviderTokenExchangeResult{}, fmt.Errorf("token response missing access_token")
	}
	tokenPlaintextBytes, err := json.Marshal(map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    tokenType,
		"scope":         scope,
	})
	if err != nil {
		return ProviderTokenExchangeResult{}, fmt.Errorf("marshal token payload: %w", err)
	}

	var expiresAt *time.Time
	switch value := payload["expires_in"].(type) {
	case float64:
		if value > 0 {
			exp := time.Now().UTC().Add(time.Duration(value) * time.Second)
			expiresAt = &exp
		}
	case int:
		if value > 0 {
			exp := time.Now().UTC().Add(time.Duration(value) * time.Second)
			expiresAt = &exp
		}
	}
	return ProviderTokenExchangeResult{
		TokenPlaintext:   string(tokenPlaintextBytes),
		RefreshSupported: refreshToken != "",
		ExpiresAt:        expiresAt,
	}, nil
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}
