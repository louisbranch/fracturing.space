package openai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

// InvokeConfig configures OpenAI provider behavior.
type InvokeConfig struct {
	ResponsesURL string
	BaseURL      string
	HTTPClient   *http.Client
}

// InvokeAdapter implements provider.InvocationAdapter, orchestration.Provider,
// and provider.ModelAdapter for the OpenAI Responses API.
type InvokeAdapter struct {
	cfg InvokeConfig
}

// NewInvokeAdapter builds an OpenAI invocation adapter.
func NewInvokeAdapter(cfg InvokeConfig) *InvokeAdapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if strings.TrimSpace(cfg.ResponsesURL) == "" {
		cfg.ResponsesURL = defaultResponsesURL()
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = openAIBaseURLFromResponsesURL(cfg.ResponsesURL)
	}
	return &InvokeAdapter{cfg: cfg}
}

func (a *InvokeAdapter) Invoke(ctx context.Context, input provider.InvokeInput) (provider.InvokeResult, error) {
	credentialSecret := strings.TrimSpace(input.CredentialSecret)
	model := strings.TrimSpace(input.Model)
	prompt := strings.TrimSpace(input.Input)
	if credentialSecret == "" {
		return provider.InvokeResult{}, fmt.Errorf("credential secret is required")
	}
	if model == "" {
		return provider.InvokeResult{}, fmt.Errorf("model is required")
	}
	if prompt == "" {
		return provider.InvokeResult{}, fmt.Errorf("input is required")
	}
	return a.invokeResponsesAPI(ctx, input)
}

// Run executes one OpenAI Responses API step with native tool calling.
func (a *InvokeAdapter) Run(ctx context.Context, input orchestration.ProviderInput) (orchestration.ProviderOutput, error) {
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
			"strict":      true,
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
