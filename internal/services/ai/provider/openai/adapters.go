package openai

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

	aiservice "github.com/louisbranch/fracturing.space/internal/services/ai/api/grpc/ai"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	anyllm "github.com/mozilla-ai/any-llm-go"
	anyllmopenai "github.com/mozilla-ai/any-llm-go/providers/openai"
)

type defaultOAuthAdapter struct{}

// OAuthConfig configures OpenAI OAuth endpoints and credentials.
type OAuthConfig struct {
	AuthorizationURL string
	TokenURL         string
	ClientID         string
	ClientSecret     string
	RedirectURI      string
	HTTPClient       *http.Client
}

type oauthAdapter struct {
	cfg OAuthConfig
}

// InvokeConfig configures OpenAI provider behavior.
type InvokeConfig struct {
	// ResponsesURL is kept for compatibility with existing configuration and is
	// used to derive the OpenAI base URL for inference and model-listing calls.
	ResponsesURL string
	HTTPClient   *http.Client
}

type invokeAdapter struct {
	cfg InvokeConfig
}

// NewDefaultOAuthAdapter builds the development/default OpenAI OAuth adapter.
func NewDefaultOAuthAdapter() aiservice.ProviderOAuthAdapter {
	return &defaultOAuthAdapter{}
}

// NewOAuthAdapter builds an OpenAI OAuth adapter using HTTP token exchange.
func NewOAuthAdapter(cfg OAuthConfig) aiservice.ProviderOAuthAdapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return &oauthAdapter{cfg: cfg}
}

// NewInvokeAdapter builds an OpenAI invocation adapter.
func NewInvokeAdapter(cfg InvokeConfig) aiservice.ProviderInvocationAdapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if strings.TrimSpace(cfg.ResponsesURL) == "" {
		cfg.ResponsesURL = "https://api.openai.com/v1/responses"
	}
	return &invokeAdapter{cfg: cfg}
}

func (a *defaultOAuthAdapter) BuildAuthorizationURL(input aiservice.ProviderAuthorizationURLInput) (string, error) {
	return fmt.Sprintf("https://oauth.fracturing.space/openai?state=%s", strings.TrimSpace(input.State)), nil
}

func (a *defaultOAuthAdapter) ExchangeAuthorizationCode(_ context.Context, input aiservice.ProviderAuthorizationCodeInput) (aiservice.ProviderTokenExchangeResult, error) {
	code := strings.TrimSpace(input.AuthorizationCode)
	if code == "" {
		return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("authorization code is required")
	}
	token := "token:" + code
	return aiservice.ProviderTokenExchangeResult{
		TokenPlaintext:   token,
		RefreshSupported: true,
	}, nil
}

func (a *defaultOAuthAdapter) RefreshToken(_ context.Context, input aiservice.ProviderRefreshTokenInput) (aiservice.ProviderTokenExchangeResult, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("refresh token is required")
	}
	token := "token:refresh:" + refreshToken
	return aiservice.ProviderTokenExchangeResult{
		TokenPlaintext:   token,
		RefreshSupported: true,
	}, nil
}

func (a *defaultOAuthAdapter) RevokeToken(_ context.Context, input aiservice.ProviderRevokeTokenInput) error {
	if strings.TrimSpace(input.Token) == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

func (a *oauthAdapter) BuildAuthorizationURL(input aiservice.ProviderAuthorizationURLInput) (string, error) {
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

func (a *oauthAdapter) ExchangeAuthorizationCode(ctx context.Context, input aiservice.ProviderAuthorizationCodeInput) (aiservice.ProviderTokenExchangeResult, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(input.AuthorizationCode))
	form.Set("code_verifier", strings.TrimSpace(input.CodeVerifier))
	form.Set("client_id", strings.TrimSpace(a.cfg.ClientID))
	form.Set("client_secret", strings.TrimSpace(a.cfg.ClientSecret))
	form.Set("redirect_uri", strings.TrimSpace(a.cfg.RedirectURI))
	return a.tokenRequest(ctx, form)
}

func (a *oauthAdapter) RefreshToken(ctx context.Context, input aiservice.ProviderRefreshTokenInput) (aiservice.ProviderTokenExchangeResult, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", strings.TrimSpace(input.RefreshToken))
	form.Set("client_id", strings.TrimSpace(a.cfg.ClientID))
	form.Set("client_secret", strings.TrimSpace(a.cfg.ClientSecret))
	return a.tokenRequest(ctx, form)
}

func (a *oauthAdapter) RevokeToken(_ context.Context, input aiservice.ProviderRevokeTokenInput) error {
	if strings.TrimSpace(input.Token) == "" {
		return fmt.Errorf("token is required")
	}
	// OpenAI revocation endpoint support is optional at this phase boundary.
	// Returning nil here avoids leaking token material into error/log paths.
	return nil
}

func (a *invokeAdapter) Invoke(ctx context.Context, input aiservice.ProviderInvokeInput) (aiservice.ProviderInvokeResult, error) {
	responsesURL := strings.TrimSpace(a.cfg.ResponsesURL)
	credentialSecret := strings.TrimSpace(input.CredentialSecret)
	model := strings.TrimSpace(input.Model)
	prompt := strings.TrimSpace(input.Input)
	if credentialSecret == "" {
		return aiservice.ProviderInvokeResult{}, fmt.Errorf("credential secret is required")
	}
	if model == "" {
		return aiservice.ProviderInvokeResult{}, fmt.Errorf("model is required")
	}
	if prompt == "" {
		return aiservice.ProviderInvokeResult{}, fmt.Errorf("input is required")
	}
	if responsesURL != "" {
		return a.invokeResponsesAPI(ctx, responsesURL, input)
	}
	provider, err := a.provider(credentialSecret)
	if err != nil {
		return aiservice.ProviderInvokeResult{}, err
	}
	messages := make([]anyllm.Message, 0, 2)
	if instructions := strings.TrimSpace(input.Instructions); instructions != "" {
		messages = append(messages, anyllm.Message{Role: anyllm.RoleSystem, Content: instructions})
	}
	messages = append(messages, anyllm.Message{Role: anyllm.RoleUser, Content: prompt})
	resp, err := provider.Completion(ctx, anyllm.CompletionParams{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return aiservice.ProviderInvokeResult{}, fmt.Errorf("invoke provider: %w", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return aiservice.ProviderInvokeResult{}, fmt.Errorf("invoke response missing choices")
	}
	outputText := strings.TrimSpace(resp.Choices[0].Message.ContentString())
	if outputText == "" {
		return aiservice.ProviderInvokeResult{}, fmt.Errorf("invoke response missing output text")
	}
	return aiservice.ProviderInvokeResult{OutputText: outputText}, nil
}

// Run executes one OpenAI Responses API step with native tool calling.
func (a *invokeAdapter) Run(ctx context.Context, input orchestration.ProviderInput) (orchestration.ProviderOutput, error) {
	credentialSecret := strings.TrimSpace(input.CredentialSecret)
	model := strings.TrimSpace(input.Model)
	if credentialSecret == "" {
		return orchestration.ProviderOutput{}, fmt.Errorf("credential secret is required")
	}
	if model == "" {
		return orchestration.ProviderOutput{}, fmt.Errorf("model is required")
	}

	tools := make([]map[string]any, 0, len(input.Tools))
	for _, tool := range input.Tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		tools = append(tools, map[string]any{
			"type":        "function",
			"name":        name,
			"description": strings.TrimSpace(tool.Description),
			"parameters":  openAIToolSchema(tool.InputSchema),
		})
	}

	body := map[string]any{
		"model":               model,
		"tools":               tools,
		"parallel_tool_calls": true,
	}
	if effort := strings.TrimSpace(input.ReasoningEffort); effort != "" {
		body["reasoning"] = map[string]any{
			"effort": effort,
		}
	}
	if instructions := strings.TrimSpace(input.Instructions); instructions != "" {
		body["instructions"] = instructions
	}
	if convo := strings.TrimSpace(input.ConversationID); convo != "" {
		body["previous_response_id"] = convo
		items := make([]map[string]any, 0, len(input.Results))
		for _, item := range input.Results {
			items = append(items, map[string]any{
				"type":    "function_call_output",
				"call_id": strings.TrimSpace(item.CallID),
				"output":  item.Output,
			})
		}
		if followUp := strings.TrimSpace(input.FollowUpPrompt); followUp != "" {
			items = append(items, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type": "input_text",
					"text": followUp,
				}},
			})
		}
		body["input"] = items
	} else {
		prompt := strings.TrimSpace(input.Prompt)
		if prompt == "" {
			return orchestration.ProviderOutput{}, fmt.Errorf("prompt is required")
		}
		body["input"] = []map[string]any{{
			"role": "user",
			"content": []map[string]any{{
				"type": "input_text",
				"text": prompt,
			}},
		}}
	}

	payload, err := a.responsesRequest(ctx, body, credentialSecret)
	if err != nil {
		return orchestration.ProviderOutput{}, err
	}
	out := orchestration.ProviderOutput{
		ConversationID: strings.TrimSpace(payload.ID),
		OutputText:     strings.TrimSpace(payload.OutputText),
		ToolCalls:      make([]orchestration.ProviderToolCall, 0, len(payload.Output)),
		Usage:          openAIUsageFromPayload(payload),
	}
	for _, item := range payload.Output {
		if strings.TrimSpace(item.Type) == "function_call" {
			out.ToolCalls = append(out.ToolCalls, orchestration.ProviderToolCall{
				CallID:    strings.TrimSpace(item.CallID),
				Name:      strings.TrimSpace(item.Name),
				Arguments: strings.TrimSpace(item.Arguments),
			})
			continue
		}
		if out.OutputText != "" {
			continue
		}
		for _, content := range item.Content {
			if strings.TrimSpace(content.Text) == "" {
				continue
			}
			out.OutputText = strings.TrimSpace(content.Text)
			break
		}
	}
	if out.OutputText == "" && len(out.ToolCalls) == 0 {
		return orchestration.ProviderOutput{}, fmt.Errorf("responses output missing text and tool calls")
	}
	return out, nil
}

func (a *invokeAdapter) invokeResponsesAPI(ctx context.Context, responsesURL string, input aiservice.ProviderInvokeInput) (aiservice.ProviderInvokeResult, error) {
	requestPayload := map[string]any{
		"model": input.Model,
		"input": input.Input,
	}
	if effort := strings.TrimSpace(input.ReasoningEffort); effort != "" {
		requestPayload["reasoning"] = map[string]any{
			"effort": effort,
		}
	}
	if instructions := strings.TrimSpace(input.Instructions); instructions != "" {
		requestPayload["instructions"] = instructions
	}
	payload, err := a.responsesRequest(ctx, requestPayload, input.CredentialSecret)
	if err != nil {
		return aiservice.ProviderInvokeResult{}, err
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
		return aiservice.ProviderInvokeResult{}, fmt.Errorf("invoke response missing output text")
	}
	return aiservice.ProviderInvokeResult{
		OutputText: outputText,
		Usage:      openAIUsageFromPayload(payload),
	}, nil
}

func openAIToolSchema(schema any) map[string]any {
	value := cloneSchemaMap(schema)
	if value == nil {
		return map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		}
	}
	if strings.TrimSpace(stringValue(value["type"])) == "" {
		value["type"] = "object"
	}
	if strings.EqualFold(stringValue(value["type"]), "object") {
		props, ok := value["properties"].(map[string]any)
		if !ok || props == nil {
			value["properties"] = map[string]any{}
		}
		if _, ok := value["additionalProperties"]; !ok {
			value["additionalProperties"] = false
		}
	}
	return value
}

func cloneSchemaMap(schema any) map[string]any {
	if schema == nil {
		return nil
	}
	if value, ok := schema.(map[string]any); ok {
		clone := make(map[string]any, len(value))
		for key, item := range value {
			clone[key] = item
		}
		return clone
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil
	}
	return value
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func (a *invokeAdapter) ListModels(ctx context.Context, input aiservice.ProviderListModelsInput) ([]aiservice.ProviderModel, error) {
	credentialSecret := strings.TrimSpace(input.CredentialSecret)
	if credentialSecret == "" {
		return nil, fmt.Errorf("credential secret is required")
	}
	provider, err := a.provider(credentialSecret)
	if err != nil {
		return nil, err
	}
	resp, err := provider.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	models := make([]aiservice.ProviderModel, 0, len(resp.Data))
	for _, model := range resp.Data {
		modelID := strings.TrimSpace(model.ID)
		if modelID == "" {
			continue
		}
		models = append(models, aiservice.ProviderModel{
			ID:      modelID,
			OwnedBy: strings.TrimSpace(model.OwnedBy),
			Created: model.Created,
		})
	}
	return models, nil
}

func (a *invokeAdapter) provider(credentialSecret string) (*anyllmopenai.Provider, error) {
	opts := []anyllm.Option{
		anyllm.WithAPIKey(credentialSecret),
		anyllm.WithHTTPClient(a.cfg.HTTPClient),
	}
	baseURL := strings.TrimSpace(openAIBaseURLFromResponsesURL(a.cfg.ResponsesURL))
	if baseURL != "" {
		opts = append(opts, anyllm.WithBaseURL(baseURL))
	}
	provider, err := anyllmopenai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("build openai provider: %w", err)
	}
	return provider, nil
}

func openAIBaseURLFromResponsesURL(responsesURL string) string {
	responsesURL = strings.TrimSpace(responsesURL)
	if responsesURL == "" {
		return "https://api.openai.com/v1"
	}
	trimmed := strings.TrimSuffix(responsesURL, "/")
	trimmed = strings.TrimSuffix(trimmed, "/responses")
	return strings.TrimSpace(trimmed)
}

func (a *oauthAdapter) tokenRequest(ctx context.Context, form url.Values) (aiservice.ProviderTokenExchangeResult, error) {
	tokenURL := strings.TrimSpace(a.cfg.TokenURL)
	if tokenURL == "" {
		return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("token url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("token request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("read token error body: %w", err)
		}
		return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("token request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("decode token response: %w", err)
	}

	accessToken := asString(payload["access_token"])
	refreshToken := asString(payload["refresh_token"])
	tokenType := asString(payload["token_type"])
	scope := asString(payload["scope"])
	if accessToken == "" {
		return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("token response missing access_token")
	}
	tokenPlaintextBytes, err := json.Marshal(map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    tokenType,
		"scope":         scope,
	})
	if err != nil {
		return aiservice.ProviderTokenExchangeResult{}, fmt.Errorf("marshal token payload: %w", err)
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
	return aiservice.ProviderTokenExchangeResult{
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

type openAIResponsesPayload struct {
	ID         string `json:"id"`
	OutputText string `json:"output_text"`
	Usage      struct {
		InputTokens        int32 `json:"input_tokens"`
		OutputTokens       int32 `json:"output_tokens"`
		TotalTokens        int32 `json:"total_tokens"`
		OutputTokenDetails struct {
			ReasoningTokens int32 `json:"reasoning_tokens"`
		} `json:"output_tokens_details"`
	} `json:"usage"`
	Output []struct {
		Type      string `json:"type"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
		CallID    string `json:"call_id"`
		Content   []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func (a *invokeAdapter) responsesRequest(ctx context.Context, body map[string]any, secret string) (openAIResponsesPayload, error) {
	requestBody, err := json.Marshal(body)
	if err != nil {
		return openAIResponsesPayload{}, fmt.Errorf("marshal invoke request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(a.cfg.ResponsesURL), bytes.NewReader(requestBody))
	if err != nil {
		return openAIResponsesPayload{}, fmt.Errorf("build invoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(secret))

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return openAIResponsesPayload{}, fmt.Errorf("invoke request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return openAIResponsesPayload{}, fmt.Errorf("read invoke error body: %w", err)
		}
		return openAIResponsesPayload{}, fmt.Errorf("invoke request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload openAIResponsesPayload
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return openAIResponsesPayload{}, fmt.Errorf("decode invoke response: %w", err)
	}
	return payload, nil
}

func openAIUsageFromPayload(payload openAIResponsesPayload) provider.Usage {
	return provider.Usage{
		InputTokens:     payload.Usage.InputTokens,
		OutputTokens:    payload.Usage.OutputTokens,
		ReasoningTokens: payload.Usage.OutputTokenDetails.ReasoningTokens,
		TotalTokens:     payload.Usage.TotalTokens,
	}
}

func normalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	items := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, raw := range scopes {
		scope := strings.TrimSpace(raw)
		if scope == "" {
			continue
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		items = append(items, scope)
	}
	return items
}
