package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

const (
	defaultBaseURL    = "https://api.anthropic.com"
	defaultAPIVersion = "2023-06-01"
	defaultMaxTokens  = 1024
)

// Config configures Anthropic invocation and model-listing behavior.
type Config struct {
	BaseURL    string
	APIVersion string
	MaxTokens  int32
	HTTPClient *http.Client
}

// Adapter implements provider.InvocationAdapter and provider.ModelAdapter for
// the Anthropic Messages and Models APIs.
type Adapter struct {
	cfg Config
}

// NewAdapter builds an Anthropic provider adapter with stable defaults.
func NewAdapter(cfg Config) *Adapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if strings.TrimSpace(cfg.APIVersion) == "" {
		cfg.APIVersion = defaultAPIVersion
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = defaultMaxTokens
	}
	return &Adapter{cfg: cfg}
}

// Invoke executes one Anthropic Messages API request.
func (a *Adapter) Invoke(ctx context.Context, input provider.InvokeInput) (provider.InvokeResult, error) {
	authToken := strings.TrimSpace(input.AuthToken)
	model := strings.TrimSpace(input.Model)
	prompt := strings.TrimSpace(input.Input)
	if authToken == "" {
		return provider.InvokeResult{}, fmt.Errorf("auth token is required")
	}
	if model == "" {
		return provider.InvokeResult{}, fmt.Errorf("model is required")
	}
	if prompt == "" {
		return provider.InvokeResult{}, fmt.Errorf("input is required")
	}

	requestBody := anthropicMessagesRequest{
		Model:     model,
		MaxTokens: a.cfg.MaxTokens,
		Messages: []anthropicMessage{{
			Role:    "user",
			Content: prompt,
		}},
	}
	if instructions := strings.TrimSpace(input.Instructions); instructions != "" {
		requestBody.System = instructions
	}

	payload, err := a.messagesRequest(ctx, authToken, requestBody)
	if err != nil {
		return provider.InvokeResult{}, err
	}

	outputText := strings.TrimSpace(payload.outputText())
	if outputText == "" {
		return provider.InvokeResult{}, fmt.Errorf("invoke response missing output text")
	}
	return provider.InvokeResult{
		OutputText: outputText,
		Usage: provider.Usage{
			InputTokens:  payload.Usage.InputTokens,
			OutputTokens: payload.Usage.OutputTokens,
			TotalTokens:  payload.Usage.InputTokens + payload.Usage.OutputTokens,
		},
	}, nil
}

// ListModels returns the model IDs visible to one Anthropic credential.
func (a *Adapter) ListModels(ctx context.Context, input provider.ListModelsInput) ([]provider.Model, error) {
	authToken := strings.TrimSpace(input.AuthToken)
	if authToken == "" {
		return nil, fmt.Errorf("auth token is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url("/v1/models"), nil)
	if err != nil {
		return nil, fmt.Errorf("build list models request: %w", err)
	}
	a.applyHeaders(req, authToken)

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list models request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return nil, fmt.Errorf("read list models error body: %w", err)
		}
		return nil, fmt.Errorf("list models request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload anthropicModelsResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode list models response: %w", err)
	}

	models := make([]provider.Model, 0, len(payload.Data))
	for _, model := range payload.Data {
		modelID := strings.TrimSpace(model.ID)
		if modelID == "" {
			continue
		}
		models = append(models, provider.Model{ID: modelID})
	}
	return models, nil
}

func (a *Adapter) messagesRequest(ctx context.Context, authToken string, body anthropicMessagesRequest) (anthropicMessagesResponse, error) {
	requestBody, err := json.Marshal(body)
	if err != nil {
		return anthropicMessagesResponse{}, fmt.Errorf("marshal invoke request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url("/v1/messages"), bytes.NewReader(requestBody))
	if err != nil {
		return anthropicMessagesResponse{}, fmt.Errorf("build invoke request: %w", err)
	}
	a.applyHeaders(req, authToken)

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return anthropicMessagesResponse{}, fmt.Errorf("invoke request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return anthropicMessagesResponse{}, fmt.Errorf("read invoke error body: %w", err)
		}
		return anthropicMessagesResponse{}, fmt.Errorf("invoke request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload anthropicMessagesResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return anthropicMessagesResponse{}, fmt.Errorf("decode invoke response: %w", err)
	}
	return payload, nil
}

func (a *Adapter) applyHeaders(req *http.Request, authToken string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", strings.TrimSpace(authToken))
	req.Header.Set("anthropic-version", strings.TrimSpace(a.cfg.APIVersion))
}

func (a *Adapter) url(path string) string {
	return strings.TrimRight(strings.TrimSpace(a.cfg.BaseURL), "/") + "/" + strings.TrimLeft(path, "/")
}

type anthropicMessagesRequest struct {
	Model     string             `json:"model"`
	MaxTokens int32              `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicMessagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int32 `json:"input_tokens"`
		OutputTokens int32 `json:"output_tokens"`
	} `json:"usage"`
}

func (r anthropicMessagesResponse) outputText() string {
	parts := make([]string, 0, len(r.Content))
	for _, block := range r.Content {
		if strings.TrimSpace(block.Type) != "text" {
			continue
		}
		text := strings.TrimSpace(block.Text)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n")
}

type anthropicModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}
