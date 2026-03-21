package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

func TestInvokeAdapterInvokeNon2xxReadError(t *testing.T) {
	adapter := &invokeAdapter{cfg: InvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       failingReadCloser{},
				}, nil
			}),
		},
	}}

	_, err := adapter.Invoke(context.Background(), provider.InvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err == nil || !strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("error = %v, want read error", err)
	}
}

func TestNewInvokeAdapterDefaults(t *testing.T) {
	adapter := NewInvokeAdapter(InvokeConfig{})
	typed, ok := adapter.(*invokeAdapter)
	if !ok {
		t.Fatalf("adapter type = %T, want *invokeAdapter", adapter)
	}
	if typed.cfg.HTTPClient == nil {
		t.Fatal("expected non-nil HTTP client")
	}
	if typed.cfg.ResponsesURL != "https://api.openai.com/v1/responses" {
		t.Fatalf("responses_url = %q", typed.cfg.ResponsesURL)
	}
}

func TestOpenAIBaseURLFromResponsesURL(t *testing.T) {
	tests := []struct {
		name         string
		responsesURL string
		want         string
	}{
		{name: "default base url", responsesURL: "", want: "https://api.openai.com/v1"},
		{name: "responses path trimmed", responsesURL: "https://provider.example.com/v1/responses", want: "https://provider.example.com/v1"},
		{name: "trailing slash trimmed", responsesURL: "https://provider.example.com/v1/responses/", want: "https://provider.example.com/v1"},
		{name: "custom endpoint without responses suffix", responsesURL: "https://provider.example.com/custom", want: "https://provider.example.com/custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := openAIBaseURLFromResponsesURL(tt.responsesURL); got != tt.want {
				t.Fatalf("base url = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInvokeAdapterInvokeValidation(t *testing.T) {
	tests := []struct {
		name  string
		input provider.InvokeInput
		want  string
	}{
		{name: "missing credential secret", input: provider.InvokeInput{Model: "gpt-4o-mini", Input: "hello"}, want: "credential secret is required"},
		{name: "missing model", input: provider.InvokeInput{Input: "hello", CredentialSecret: "sk-1"}, want: "model is required"},
		{name: "missing input", input: provider.InvokeInput{Model: "gpt-4o-mini", CredentialSecret: "sk-1"}, want: "input is required"},
	}

	adapter := &invokeAdapter{cfg: InvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("round trip should not execute for validation failure: %v", req.URL)
				return nil, nil
			}),
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.Invoke(context.Background(), tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestInvokeAdapterInvokeProviderError(t *testing.T) {
	adapter := &invokeAdapter{cfg: InvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, io.ErrUnexpectedEOF
			}),
		},
	}}

	_, err := adapter.Invoke(context.Background(), provider.InvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err == nil || !strings.Contains(err.Error(), "invoke request failed") || !strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("error = %v, want provider error", err)
	}
}

func TestInvokeAdapterInvokeDecodeAndOutputErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "invalid json", body: "{bad json", want: "decode invoke response"},
		{name: "missing output", body: `{}`, want: "invoke response missing output text"},
		{name: "blank output", body: `{"output_text":" "}`, want: "invoke response missing output text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &invokeAdapter{cfg: InvokeConfig{
				ResponsesURL: "https://provider.example.com/v1/responses",
				HTTPClient: &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						return response(http.StatusOK, tt.body), nil
					}),
				},
			}}

			if _, err := adapter.Invoke(context.Background(), provider.InvokeInput{
				Model:            "gpt-4o-mini",
				Input:            "Say hello",
				CredentialSecret: "sk-1",
			}); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestInvokeAdapterInvokeAndListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-1" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/responses":
			var body struct {
				Model        string         `json:"model"`
				Input        string         `json:"input"`
				Instructions string         `json:"instructions"`
				Reasoning    map[string]any `json:"reasoning"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if body.Model != "gpt-4o-mini" {
				t.Fatalf("model = %q, want %q", body.Model, "gpt-4o-mini")
			}
			if body.Instructions != "Stay in character." {
				t.Fatalf("instructions = %q, want %q", body.Instructions, "Stay in character.")
			}
			if body.Input != "Say hello" {
				t.Fatalf("input = %q, want %q", body.Input, "Say hello")
			}
			if got := body.Reasoning["effort"]; got != "low" {
				t.Fatalf("reasoning.effort = %#v, want %q", got, "low")
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"output_text": "Hello from OpenAI",
				"usage": map[string]any{
					"input_tokens":  12,
					"output_tokens": 7,
					"total_tokens":  19,
					"output_tokens_details": map[string]any{
						"reasoning_tokens": 3,
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list",
				"data": []map[string]any{
					{"id": "gpt-4o-mini", "object": "model", "created": 1, "owned_by": "openai"},
					{"id": "gpt-4o", "object": "model", "created": 1, "owned_by": "openai"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewInvokeAdapter(InvokeConfig{
		ResponsesURL: server.URL + "/v1/responses",
	})
	got, err := adapter.Invoke(context.Background(), provider.InvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		Instructions:     "Stay in character.",
		ReasoningEffort:  "low",
		CredentialSecret: "sk-1",
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if got.OutputText != "Hello from OpenAI" {
		t.Fatalf("output_text = %q, want %q", got.OutputText, "Hello from OpenAI")
	}
	if got.Usage != (provider.Usage{InputTokens: 12, OutputTokens: 7, ReasoningTokens: 3, TotalTokens: 19}) {
		t.Fatalf("usage = %#v", got.Usage)
	}

	modelAdapter, ok := adapter.(provider.ModelAdapter)
	if !ok {
		t.Fatalf("adapter type %T does not implement provider.ModelAdapter", adapter)
	}
	models, err := modelAdapter.ListModels(context.Background(), provider.ListModelsInput{CredentialSecret: "sk-1"})
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("models len = %d, want 2", len(models))
	}
	if models[0].ID != "gpt-4o-mini" || models[1].ID != "gpt-4o" {
		t.Fatalf("models = %#v, want gpt-4o-mini and gpt-4o", models)
	}
	if models[0].Created != 1 || models[1].Created != 1 {
		t.Fatalf("created values = %#v, want provider-created timestamps preserved", models)
	}
}

func TestInvokeAdapterListModelsValidationAndError(t *testing.T) {
	adapter := &invokeAdapter{cfg: InvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return response(http.StatusUnauthorized, "bad credential"), nil
			}),
		},
	}}

	if _, err := adapter.ListModels(context.Background(), provider.ListModelsInput{}); err == nil || !strings.Contains(err.Error(), "credential secret is required") {
		t.Fatalf("error = %v, want missing credential secret", err)
	}
	if _, err := adapter.ListModels(context.Background(), provider.ListModelsInput{CredentialSecret: "sk-1"}); err == nil || !strings.Contains(err.Error(), "list models") {
		t.Fatalf("error = %v, want list models provider error", err)
	}
}

func TestInvokeAdapterRunNormalizesZeroArgToolSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Tools []struct {
				Name       string         `json:"name"`
				Parameters map[string]any `json:"parameters"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if len(body.Tools) != 2 {
			t.Fatalf("tools len = %d, want 2", len(body.Tools))
		}
		if body.Tools[0].Name != "duality_rules_version" {
			t.Fatalf("tool[0].name = %q", body.Tools[0].Name)
		}
		if body.Tools[0].Parameters["type"] != "object" {
			t.Fatalf("tool[0].parameters.type = %#v", body.Tools[0].Parameters["type"])
		}
		props, ok := body.Tools[0].Parameters["properties"].(map[string]any)
		if !ok {
			t.Fatalf("tool[0].parameters.properties type = %T", body.Tools[0].Parameters["properties"])
		}
		if len(props) != 0 {
			t.Fatalf("tool[0].parameters.properties = %#v, want empty object", props)
		}
		if body.Tools[0].Parameters["additionalProperties"] != false {
			t.Fatalf("tool[0].parameters.additionalProperties = %#v", body.Tools[0].Parameters["additionalProperties"])
		}
		props, ok = body.Tools[1].Parameters["properties"].(map[string]any)
		if !ok || len(props) != 1 {
			t.Fatalf("tool[1].parameters.properties = %#v", body.Tools[1].Parameters["properties"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "resp-1",
			"output_text": "Scene established.",
			"usage": map[string]any{
				"input_tokens":  11,
				"output_tokens": 5,
				"total_tokens":  16,
				"output_tokens_details": map[string]any{
					"reasoning_tokens": 2,
				},
			},
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{"type": "output_text", "text": "Scene established."},
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &invokeAdapter{cfg: InvokeConfig{
		ResponsesURL: server.URL + "/v1/responses",
		HTTPClient:   server.Client(),
	}}
	res, err := adapter.Run(context.Background(), orchestration.ProviderInput{
		Model:            "gpt-4.1-mini",
		Prompt:           "Start the scene.",
		CredentialSecret: "sk-1",
		Tools: []orchestration.Tool{
			{Name: "duality_rules_version", Description: "Describe the ruleset", InputSchema: map[string]any{"type": "object"}},
			{
				Name:        "scene_create",
				Description: "Create a scene",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.OutputText != "Scene established." {
		t.Fatalf("output_text = %q", res.OutputText)
	}
	if res.Usage != (provider.Usage{InputTokens: 11, OutputTokens: 5, ReasoningTokens: 2, TotalTokens: 16}) {
		t.Fatalf("usage = %#v", res.Usage)
	}
}

func TestInvokeAdapterRunIncludesReasoningEffort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Model     string         `json:"model"`
			Reasoning map[string]any `json:"reasoning"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Model != "gpt-5.4" {
			t.Fatalf("model = %q, want %q", body.Model, "gpt-5.4")
		}
		if got := body.Reasoning["effort"]; got != "medium" {
			t.Fatalf("reasoning.effort = %#v, want %q", got, "medium")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "resp-1",
			"output_text": "Scene established.",
			"usage": map[string]any{
				"input_tokens":  9,
				"output_tokens": 4,
				"total_tokens":  13,
				"output_tokens_details": map[string]any{
					"reasoning_tokens": 1,
				},
			},
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{"type": "output_text", "text": "Scene established."},
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &invokeAdapter{cfg: InvokeConfig{
		ResponsesURL: server.URL + "/v1/responses",
		HTTPClient:   server.Client(),
	}}
	res, err := adapter.Run(context.Background(), orchestration.ProviderInput{
		Model:            "gpt-5.4",
		ReasoningEffort:  "medium",
		Prompt:           "Start the scene.",
		CredentialSecret: "sk-1",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.OutputText != "Scene established." {
		t.Fatalf("output_text = %q", res.OutputText)
	}
	if res.Usage != (provider.Usage{InputTokens: 9, OutputTokens: 4, ReasoningTokens: 1, TotalTokens: 13}) {
		t.Fatalf("usage = %#v", res.Usage)
	}
}
