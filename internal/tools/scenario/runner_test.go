package scenario

import (
	"strings"
	"testing"
	"time"
)

func TestNewRunnerWithDeps_EmptyUserID(t *testing.T) {
	auth := &fakeAuthProvider{userID: ""}
	deps := runnerDeps{
		env:  scenarioEnv{},
		auth: auth,
	}
	cfg := Config{Assertions: AssertionStrict, Timeout: 5 * time.Second}
	_, err := newRunnerWithDeps(cfg, deps)
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
	if !strings.Contains(err.Error(), "empty user id") {
		t.Fatalf("expected empty user id error, got: %v", err)
	}
}

func TestNewRunnerWithDeps_DefaultLogger(t *testing.T) {
	auth := &fakeAuthProvider{userID: "user-1"}
	deps := runnerDeps{
		env:  scenarioEnv{},
		auth: auth,
	}
	cfg := Config{Assertions: AssertionStrict, Timeout: 5 * time.Second}
	r, err := newRunnerWithDeps(cfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.logger == nil {
		t.Fatal("expected default logger to be set")
	}
}

func TestNewRunnerWithDeps_DefaultTimeout(t *testing.T) {
	auth := &fakeAuthProvider{userID: "user-1"}
	deps := runnerDeps{
		env:  scenarioEnv{},
		auth: auth,
	}
	cfg := Config{Assertions: AssertionStrict}
	r, err := newRunnerWithDeps(cfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.timeout != 10*time.Second {
		t.Fatalf("expected default timeout 10s, got %v", r.timeout)
	}
}

// fakeAuthProvider is a test double that satisfies authProvider.
type fakeAuthProvider struct {
	userID string
}

func (f *fakeAuthProvider) CreateUser(_ string) string {
	return f.userID
}
