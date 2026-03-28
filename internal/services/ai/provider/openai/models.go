package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	anyllm "github.com/mozilla-ai/any-llm-go"
	anyllmopenai "github.com/mozilla-ai/any-llm-go/providers/openai"
)

func (a *InvokeAdapter) ListModels(ctx context.Context, input provider.ListModelsInput) ([]provider.Model, error) {
	authToken := strings.TrimSpace(input.AuthToken)
	if authToken == "" {
		return nil, fmt.Errorf("auth token is required")
	}
	openAIProvider, err := a.providerClient(authToken)
	if err != nil {
		return nil, err
	}
	resp, err := openAIProvider.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	models := make([]provider.Model, 0, len(resp.Data))
	for _, model := range resp.Data {
		modelID := strings.TrimSpace(model.ID)
		if modelID == "" {
			continue
		}
		models = append(models, provider.Model{ID: modelID})
	}
	return models, nil
}

func (a *InvokeAdapter) providerClient(authToken string) (*anyllmopenai.Provider, error) {
	opts := []anyllm.Option{
		anyllm.WithAPIKey(authToken),
		anyllm.WithHTTPClient(a.cfg.HTTPClient),
	}
	baseURL := strings.TrimSpace(a.cfg.BaseURL)
	if baseURL != "" {
		opts = append(opts, anyllm.WithBaseURL(baseURL))
	}
	providerClient, err := anyllmopenai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("build openai provider: %w", err)
	}
	return providerClient, nil
}

func openAIBaseURLFromResponsesURL(responsesURL string) string {
	responsesURL = strings.TrimSpace(responsesURL)
	if responsesURL == "" {
		return defaultBaseURL()
	}
	trimmed := strings.TrimSuffix(responsesURL, "/")
	trimmed = strings.TrimSuffix(trimmed, "/responses")
	return strings.TrimSpace(trimmed)
}

func defaultResponsesURL() string {
	return "https://api.openai.com/v1/responses"
}

func defaultBaseURL() string {
	return "https://api.openai.com/v1"
}
