package publicauth

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParsePasskeyRegisterStartInput(t *testing.T) {
	req := httptest.NewRequest("POST", "/passkeys/register/start", strings.NewReader(`{"username":" louis "}`))
	input, err := parsePasskeyRegisterStartInput(req)
	if err != nil {
		t.Fatalf("parsePasskeyRegisterStartInput() error = %v", err)
	}
	if input.Username != "louis" {
		t.Fatalf("username = %q, want %q", input.Username, "louis")
	}
}

func TestParsePasskeyLoginStartInput(t *testing.T) {
	req := httptest.NewRequest("POST", "/passkeys/login/start", strings.NewReader(`{"username":" louis "}`))
	input, err := parsePasskeyLoginStartInput(req)
	if err != nil {
		t.Fatalf("parsePasskeyLoginStartInput() error = %v", err)
	}
	if input.Username != "louis" {
		t.Fatalf("username = %q, want %q", input.Username, "louis")
	}
}
