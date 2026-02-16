package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
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
}

func TestOpenAIInvokeAdapterInvokeValidation(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			t.Fatalf("round trip should not execute for validation failure: %v", req.URL)
			return nil, nil
		}),
	}

	tests := []struct {
		name    string
		adapter *openAIInvokeAdapter
		input   ProviderInvokeInput
	}{
		{
			name: "missing responses url",
			adapter: &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
				ResponsesURL: "",
				HTTPClient:   client,
			}},
			input: ProviderInvokeInput{Model: "gpt-4o-mini", Input: "hello", CredentialSecret: "sk-1"},
		},
		{
			name: "missing credential secret",
			adapter: &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
				ResponsesURL: "https://provider.example.com/v1/responses",
				HTTPClient:   client,
			}},
			input: ProviderInvokeInput{Model: "gpt-4o-mini", Input: "hello", CredentialSecret: ""},
		},
		{
			name: "missing model",
			adapter: &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
				ResponsesURL: "https://provider.example.com/v1/responses",
				HTTPClient:   client,
			}},
			input: ProviderInvokeInput{Model: "", Input: "hello", CredentialSecret: "sk-1"},
		},
		{
			name: "missing input",
			adapter: &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
				ResponsesURL: "https://provider.example.com/v1/responses",
				HTTPClient:   client,
			}},
			input: ProviderInvokeInput{Model: "gpt-4o-mini", Input: "", CredentialSecret: "sk-1"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.adapter.Invoke(context.Background(), tt.input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestOpenAIInvokeAdapterInvokeRoundTripError(t *testing.T) {
	adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("dial timeout")
			}),
		},
	}}

	_, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err == nil || !strings.Contains(err.Error(), "invoke request failed") {
		t.Fatalf("error = %v, want invoke request failed", err)
	}
}

func TestOpenAIInvokeAdapterInvokeSuccessWithOutputText(t *testing.T) {
	adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Authorization") != "Bearer sk-1" {
					t.Fatalf("authorization = %q", req.Header.Get("Authorization"))
				}
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				if !strings.Contains(string(body), "\"model\":\"gpt-4o-mini\"") {
					t.Fatalf("request body = %s", string(body))
				}
				if !strings.Contains(string(body), "\"input\":\"Say hello\"") {
					t.Fatalf("request body = %s", string(body))
				}
				return response(http.StatusOK, `{"output_text":"Hello from OpenAI"}`), nil
			}),
		},
	}}

	got, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if got.OutputText != "Hello from OpenAI" {
		t.Fatalf("output_text = %q, want %q", got.OutputText, "Hello from OpenAI")
	}
}

func TestOpenAIInvokeAdapterInvokeDecodeAndOutputErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: "{bad json"},
		{name: "missing output", body: "{}"},
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
			}); err == nil {
				t.Fatal("expected invoke error")
			}
		})
	}
}

func TestOpenAIInvokeAdapterInvokeNon2xx(t *testing.T) {
	adapter := &openAIInvokeAdapter{cfg: OpenAIInvokeConfig{
		ResponsesURL: "https://provider.example.com/v1/responses",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return response(http.StatusUnauthorized, "bad credential"), nil
			}),
		},
	}}

	_, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err == nil || !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("error = %v, want status 401", err)
	}
}
