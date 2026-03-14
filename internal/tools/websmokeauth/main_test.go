package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSanitizeEmailPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "web-smoke"},
		{name: "spaces only", input: "   ", want: "web-smoke"},
		{name: "alpha numeric", input: "Web Smoke 123", want: "web-smoke-123"},
		{name: "symbol collapse", input: "a***b___c", want: "a-b-c"},
		{name: "leading trailing symbols", input: "---smoke---", want: "smoke"},
		{name: "no ascii letters", input: "测试", want: "web-smoke"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeEmailPrefix(tc.input); got != tc.want {
				t.Fatalf("sanitizeEmailPrefix(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestRunAuthAddrRequired(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-auth-addr", "   "}, &stdout, &stderr)
	if got := exitCode(err); got != 1 {
		t.Fatalf("exitCode(run) = %d, want 1 (err=%v)", got, err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "websmokeauth: auth address is required") {
		t.Fatalf("stderr = %q, want auth address message", stderr.String())
	}
}

func TestRunTTLMustBePositive(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-ttl-seconds", "0", "-username", "alpha", "-recipient-username", "beta"}, &stdout, &stderr)
	if got := exitCode(err); got != 1 {
		t.Fatalf("exitCode(run) = %d, want 1 (err=%v)", got, err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "websmokeauth: ttl-seconds must be > 0") {
		t.Fatalf("stderr = %q, want ttl validation message", stderr.String())
	}
}

func TestRunTimeoutMustBePositive(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-timeout", "0s", "-username", "alpha", "-recipient-username", "beta"}, &stdout, &stderr)
	if got := exitCode(err); got != 1 {
		t.Fatalf("exitCode(run) = %d, want 1 (err=%v)", got, err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "websmokeauth: timeout must be > 0") {
		t.Fatalf("stderr = %q, want timeout validation message", stderr.String())
	}
}

func TestRunUnknownFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-unknown"}, &stdout, &stderr)
	if got := exitCode(err); got != 2 {
		t.Fatalf("exitCode(run) = %d, want 2 (err=%v)", got, err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr = %q, want flag parse error", stderr.String())
	}
}

func TestRunUsernameRequired(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-username", "   ", "-recipient-username", "beta"}, &stdout, &stderr)
	if got := exitCode(err); got != 1 {
		t.Fatalf("exitCode(run) = %d, want 1 (err=%v)", got, err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "websmokeauth: username is required") {
		t.Fatalf("stderr = %q, want username validation message", stderr.String())
	}
}

func TestRunRecipientUsernameRequired(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-username", "alpha", "-recipient-username", "   "}, &stdout, &stderr)
	if got := exitCode(err); got != 1 {
		t.Fatalf("exitCode(run) = %d, want 1 (err=%v)", got, err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "websmokeauth: recipient-username is required") {
		t.Fatalf("stderr = %q, want recipient username validation message", stderr.String())
	}
}
