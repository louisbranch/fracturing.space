package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func response(status int, body string) *http.Response {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type failingReadCloser struct{}

func (f failingReadCloser) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (f failingReadCloser) Close() error {
	return nil
}

func TestOpenAIInvokeAdapterInvokeNon2xxReadError(t *testing.T) {
	adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
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

	_, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err == nil || !strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("error = %v, want read error", err)
	}
}

func TestNewOpenAIInvokeAdapterDefaults(t *testing.T) {
	adapter := NewOpenAIInvokeAdapter(OpenAIInvokeConfig{})
	typed, ok := adapter.(*openAIInvokeAdapter)
	if !ok {
		t.Fatalf("adapter type = %T, want *openAIInvokeAdapter", adapter)
	}
	if typed.cfg.HTTPClient == nil {
		t.Fatal("expected non-nil HTTP client")
	}
	if typed.cfg.ResponsesURL != "https://api.openai.com/v1/responses" {
		t.Fatalf("responses_url = %q", typed.cfg.ResponsesURL)
	}
}

func TestSetOpenAIInvocationAdapterNoopOnNilInputs(t *testing.T) {
	var svc *Service
	svc.SetOpenAIInvocationAdapter(&fakeProviderInvocationAdapter{})

	svc = &Service{}
	svc.SetOpenAIInvocationAdapter(nil)
	if svc.providerInvocationAdapters != nil {
		t.Fatalf("provider invocation adapters = %v, want nil", svc.providerInvocationAdapters)
	}
}

func TestSetOpenAIInvocationAdapterStoresAdapter(t *testing.T) {
	svc := &Service{}
	adapter := &fakeProviderInvocationAdapter{}
	svc.SetOpenAIInvocationAdapter(adapter)
	if got := svc.providerInvocationAdapters[providergrant.ProviderOpenAI]; got != adapter {
		t.Fatalf("stored adapter = %v, want %v", got, adapter)
	}
	if got := svc.providerModelAdapters[providergrant.ProviderOpenAI]; got != adapter {
		t.Fatalf("stored model adapter = %v, want %v", got, adapter)
	}
}

func TestOpenAIBaseURLFromResponsesURL(t *testing.T) {
	tests := []struct {
		name         string
		responsesURL string
		want         string
	}{
		{
			name:         "default base url",
			responsesURL: "",
			want:         "https://api.openai.com/v1",
		},
		{
			name:         "responses path trimmed",
			responsesURL: "https://provider.example.com/v1/responses",
			want:         "https://provider.example.com/v1",
		},
		{
			name:         "trailing slash trimmed",
			responsesURL: "https://provider.example.com/v1/responses/",
			want:         "https://provider.example.com/v1",
		},
		{
			name:         "custom endpoint without responses suffix",
			responsesURL: "https://provider.example.com/custom",
			want:         "https://provider.example.com/custom",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := openAIBaseURLFromResponsesURL(tt.responsesURL); got != tt.want {
				t.Fatalf("base url = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenAIInvokeAdapterInvokeValidation(t *testing.T) {
	tests := []struct {
		name  string
		input ProviderInvokeInput
		want  string
	}{
		{
			name:  "missing credential secret",
			input: ProviderInvokeInput{Model: "gpt-4o-mini", Input: "hello"},
			want:  "credential secret is required",
		},
		{
			name:  "missing model",
			input: ProviderInvokeInput{Input: "hello", CredentialSecret: "sk-1"},
			want:  "model is required",
		},
		{
			name:  "missing input",
			input: ProviderInvokeInput{Model: "gpt-4o-mini", CredentialSecret: "sk-1"},
			want:  "input is required",
		},
	}

	adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				t.Fatalf("round trip should not execute for validation failure: %v", req.URL)
				return nil, nil
			}),
		},
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.Invoke(context.Background(), tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestOpenAIInvokeAdapterInvokeProviderError(t *testing.T) {
	adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, io.ErrUnexpectedEOF
			}),
		},
	}}

	_, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err == nil || !strings.Contains(err.Error(), "invoke request failed") || !strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("error = %v, want provider error", err)
	}
}

func TestOpenAIInvokeAdapterInvokeDecodeAndOutputErrors(t *testing.T) {
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
				ResponsesURL: "https://provider.example.com/v1/responses",
				HTTPClient: &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						return response(http.StatusOK, tt.body), nil
					}),
				},
			}}

			if _, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
				Model:            "gpt-4o-mini",
				Input:            "Say hello",
				CredentialSecret: "sk-1",
			}); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestOpenAIInvokeAdapterInvokeAndListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-1" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/responses":
			var body struct {
				Model        string `json:"model"`
				Input        string `json:"input"`
				Instructions string `json:"instructions"`
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
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"output_text": "Hello from OpenAI",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list",
				"data": []map[string]any{
					{
						"id":       "gpt-4o-mini",
						"object":   "model",
						"created":  1,
						"owned_by": "openai",
					},
					{
						"id":       "gpt-4o",
						"object":   "model",
						"created":  1,
						"owned_by": "openai",
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewOpenAIInvokeAdapter(OpenAIInvokeConfig{
		ResponsesURL: server.URL + "/v1/responses",
	})
	got, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		Instructions:     "Stay in character.",
		CredentialSecret: "sk-1",
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if got.OutputText != "Hello from OpenAI" {
		t.Fatalf("output_text = %q, want %q", got.OutputText, "Hello from OpenAI")
	}

	modelAdapter, ok := adapter.(ProviderModelAdapter)
	if !ok {
		t.Fatalf("adapter type %T does not implement ProviderModelAdapter", adapter)
	}
	models, err := modelAdapter.ListModels(context.Background(), ProviderListModelsInput{CredentialSecret: "sk-1"})
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

func TestOpenAIInvokeAdapterListModelsValidationAndError(t *testing.T) {
	adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return response(http.StatusUnauthorized, "bad credential"), nil
			}),
		},
	}}

	if _, err := adapter.ListModels(context.Background(), ProviderListModelsInput{}); err == nil || !strings.Contains(err.Error(), "credential secret is required") {
		t.Fatalf("error = %v, want missing credential secret", err)
	}
	if _, err := adapter.ListModels(context.Background(), ProviderListModelsInput{CredentialSecret: "sk-1"}); err == nil || !strings.Contains(err.Error(), "list models") {
		t.Fatalf("error = %v, want list models provider error", err)
	}
}

func TestOpenAIInvokeAdapterRunNormalizesZeroArgToolSchema(t *testing.T) {
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
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{
							"type": "output_text",
							"text": "Scene established.",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
		ResponsesURL: server.URL + "/v1/responses",
		HTTPClient:   server.Client(),
	}}
	res, err := adapter.Run(context.Background(), orchestration.ProviderInput{
		Model:            "gpt-4.1-mini",
		Prompt:           "Start the scene.",
		CredentialSecret: "sk-1",
		Tools: []orchestration.Tool{
			{
				Name:        "duality_rules_version",
				Description: "Describe the ruleset",
				InputSchema: map[string]any{"type": "object"},
			},
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
}

func TestOpenAIInvokeAdapterRunIncludesReasoningEffort(t *testing.T) {
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
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{
							"type": "output_text",
							"text": "Scene established.",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
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
}
