package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

func TestNewAdapterDefaults(t *testing.T) {
	adapter := NewAdapter(Config{})
	if adapter.cfg.HTTPClient == nil {
		t.Fatal("expected non-nil HTTP client")
	}
	if adapter.cfg.BaseURL != defaultBaseURL {
		t.Fatalf("base url = %q, want %q", adapter.cfg.BaseURL, defaultBaseURL)
	}
	if adapter.cfg.APIVersion != defaultAPIVersion {
		t.Fatalf("api version = %q, want %q", adapter.cfg.APIVersion, defaultAPIVersion)
	}
	if adapter.cfg.MaxTokens != defaultMaxTokens {
		t.Fatalf("max tokens = %d, want %d", adapter.cfg.MaxTokens, defaultMaxTokens)
	}
}

func TestAdapterInvokeValidation(t *testing.T) {
	tests := []struct {
		name  string
		input provider.InvokeInput
		want  string
	}{
		{name: "missing auth token", input: provider.InvokeInput{Model: "claude-sonnet", Input: "hello"}, want: "auth token is required"},
		{name: "missing model", input: provider.InvokeInput{Input: "hello", AuthToken: "sk-ant-1"}, want: "model is required"},
		{name: "missing input", input: provider.InvokeInput{Model: "claude-sonnet", AuthToken: "sk-ant-1"}, want: "input is required"},
	}

	adapter := &Adapter{cfg: Config{
		BaseURL: defaultBaseURL,
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

func TestAdapterInvokeAndListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-ant-1" {
			t.Fatalf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != defaultAPIVersion {
			t.Fatalf("anthropic-version = %q, want %q", r.Header.Get("anthropic-version"), defaultAPIVersion)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/messages":
			var body struct {
				Model     string `json:"model"`
				MaxTokens int32  `json:"max_tokens"`
				System    string `json:"system"`
				Messages  []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if body.Model != "claude-sonnet-4-5" {
				t.Fatalf("model = %q, want %q", body.Model, "claude-sonnet-4-5")
			}
			if body.MaxTokens != 2048 {
				t.Fatalf("max_tokens = %d, want %d", body.MaxTokens, 2048)
			}
			if body.System != "Stay in character." {
				t.Fatalf("system = %q, want %q", body.System, "Stay in character.")
			}
			if len(body.Messages) != 1 || body.Messages[0].Role != "user" || body.Messages[0].Content != "Say hello" {
				t.Fatalf("messages = %#v", body.Messages)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": "Hello from Anthropic"},
				},
				"usage": map[string]any{
					"input_tokens":  12,
					"output_tokens": 9,
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "claude-sonnet-4-5"},
					{"id": "claude-haiku-4-5"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewAdapter(Config{
		BaseURL:   server.URL,
		MaxTokens: 2048,
	})
	got, err := adapter.Invoke(context.Background(), provider.InvokeInput{
		Model:        "claude-sonnet-4-5",
		Input:        "Say hello",
		Instructions: "Stay in character.",
		AuthToken:    "sk-ant-1",
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if got.OutputText != "Hello from Anthropic" {
		t.Fatalf("output_text = %q, want %q", got.OutputText, "Hello from Anthropic")
	}
	if got.Usage != (provider.Usage{InputTokens: 12, OutputTokens: 9, TotalTokens: 21}) {
		t.Fatalf("usage = %#v", got.Usage)
	}

	models, err := adapter.ListModels(context.Background(), provider.ListModelsInput{AuthToken: "sk-ant-1"})
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("models len = %d, want 2", len(models))
	}
	if models[0].ID != "claude-sonnet-4-5" || models[1].ID != "claude-haiku-4-5" {
		t.Fatalf("models = %#v", models)
	}
}

func TestAdapterInvokeProviderErrors(t *testing.T) {
	adapter := &Adapter{cfg: Config{
		BaseURL: defaultBaseURL,
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, io.ErrUnexpectedEOF
			}),
		},
	}}

	_, err := adapter.Invoke(context.Background(), provider.InvokeInput{
		Model:     "claude-sonnet-4-5",
		Input:     "Say hello",
		AuthToken: "sk-ant-1",
	})
	if err == nil || !strings.Contains(err.Error(), "invoke request failed") {
		t.Fatalf("error = %v, want invoke request failure", err)
	}
}

func TestAdapterListModelsValidationAndError(t *testing.T) {
	adapter := &Adapter{cfg: Config{
		BaseURL: defaultBaseURL,
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return response(http.StatusUnauthorized, "bad credential"), nil
			}),
		},
	}}

	if _, err := adapter.ListModels(context.Background(), provider.ListModelsInput{}); err == nil || !strings.Contains(err.Error(), "auth token is required") {
		t.Fatalf("error = %v, want missing auth token", err)
	}
	if _, err := adapter.ListModels(context.Background(), provider.ListModelsInput{AuthToken: "sk-ant-1"}); err == nil || !strings.Contains(err.Error(), "list models request status") {
		t.Fatalf("error = %v, want list models provider error", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
