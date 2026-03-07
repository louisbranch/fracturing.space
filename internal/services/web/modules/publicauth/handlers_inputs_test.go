package publicauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestParsePasskeyCredentialInputTrimsSessionID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, routepathPasskeysTestPath, strings.NewReader(`{"session_id":" session-1 ","credential":{"id":"cred-1"}}`))
	input, err := parsePasskeyCredentialInput(req)
	if err != nil {
		t.Fatalf("parsePasskeyCredentialInput() error = %v", err)
	}
	if input.SessionID != "session-1" {
		t.Fatalf("SessionID = %q, want %q", input.SessionID, "session-1")
	}
	if len(input.Credential) == 0 {
		t.Fatalf("Credential should not be empty")
	}
	var credential map[string]any
	if err := json.Unmarshal(input.Credential, &credential); err != nil {
		t.Fatalf("json.Unmarshal(credential) error = %v", err)
	}
	if got := asString(credential["id"]); got != "cred-1" {
		t.Fatalf("credential.id = %q, want %q", got, "cred-1")
	}
}

func TestParsePasskeyRegisterStartInputTrimsEmail(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, routepathPasskeysTestPath, strings.NewReader(`{"email":" user@example.com "}`))
	input, err := parsePasskeyRegisterStartInput(req)
	if err != nil {
		t.Fatalf("parsePasskeyRegisterStartInput() error = %v", err)
	}
	if input.Email != "user@example.com" {
		t.Fatalf("Email = %q, want %q", input.Email, "user@example.com")
	}
}

func TestPasskeyParsersRejectInvalidJSONPayloads(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		body  string
		parse func(*http.Request) error
	}{
		{
			name: "malformed json",
			body: `{`,
			parse: func(r *http.Request) error {
				_, err := parsePasskeyCredentialInput(r)
				return err
			},
		},
		{
			name: "unknown field",
			body: `{"session_id":"session-1","credential":{"id":"cred-1"},"extra":"nope"}`,
			parse: func(r *http.Request) error {
				_, err := parsePasskeyCredentialInput(r)
				return err
			},
		},
		{
			name: "multiple json payloads",
			body: `{"email":"user@example.com"}{"email":"ignored@example.com"}`,
			parse: func(r *http.Request) error {
				_, err := parsePasskeyRegisterStartInput(r)
				return err
			},
		},
		{
			name: "empty body",
			body: ``,
			parse: func(r *http.Request) error {
				_, err := parsePasskeyRegisterStartInput(r)
				return err
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, routepathPasskeysTestPath, strings.NewReader(tc.body))
			err := tc.parse(req)
			if err == nil {
				t.Fatalf("expected parse error")
			}
			if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
			}
		})
	}
}

func TestPasskeyParsersRejectOversizedPayload(t *testing.T) {
	t.Parallel()

	oversized := `{"email":"` + strings.Repeat("a", maxJSONBodyBytes+1) + `"}`
	req := httptest.NewRequest(http.MethodPost, routepathPasskeysTestPath, strings.NewReader(oversized))
	_, err := parsePasskeyRegisterStartInput(req)
	if err == nil {
		t.Fatalf("expected oversize parse error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

const routepathPasskeysTestPath = "/auth/passkeys/test"
