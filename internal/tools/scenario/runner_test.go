package scenario

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.GRPCAddr != "localhost:8080" {
		t.Fatalf("grpc addr = %q, want %q", cfg.GRPCAddr, "localhost:8080")
	}
	if cfg.Timeout != 10*time.Second {
		t.Fatalf("timeout = %v, want %v", cfg.Timeout, 10*time.Second)
	}
	if cfg.Assertions != AssertionStrict {
		t.Fatalf("assertions = %v, want %v", cfg.Assertions, AssertionStrict)
	}
	if cfg.Verbose {
		t.Fatal("verbose = true, want false")
	}
	if !cfg.ValidateComments {
		t.Fatal("validate comments = false, want true")
	}
}

func TestNewRunnerRequiresGRPCAddr(t *testing.T) {
	_, err := NewRunner(context.Background(), Config{})
	if err == nil || !strings.Contains(err.Error(), "grpc address is required") {
		t.Fatalf("expected grpc address error, got %v", err)
	}
}

func TestRunFileReturnsRunnerSetupError(t *testing.T) {
	err := RunFile(context.Background(), Config{}, "ignored.lua")
	if err == nil || !strings.Contains(err.Error(), "grpc address is required") {
		t.Fatalf("expected grpc address error, got %v", err)
	}
}

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
	err    error
}

func (f *fakeAuthProvider) CreateUser(_ string) (string, error) {
	return f.userID, f.err
}
