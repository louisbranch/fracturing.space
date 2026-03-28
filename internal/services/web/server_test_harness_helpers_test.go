// Test harness helpers for the web service root package.
//
// # When to use this harness vs webtest.Runtime
//
// Use the server_test_harness_* files (this harness) when you need:
//   - Fast unit-level tests that run in-process with fake gRPC clients
//   - Fine-grained control over which dependencies are present/absent
//   - Testing specific handler behavior, routing, or error paths
//
// Use webtest.Runtime (webtest/runtime.go) when you need:
//   - Integration tests against real downstream gRPC services
//   - Full HTTP server lifecycle with actual network connections
//   - Cross-service workflow validation
//
// # Harness file layout
//
//   - server_test_harness_helpers_test.go: shared test utilities (this file)
//   - server_test_harness_defaults_test.go: default config builders with realistic fakes
//   - server_test_harness_fakes_*_test.go: fake gRPC client implementations per service
package web

import (
	"context"
	"net/http"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

func assertPrimaryNavLinks(t *testing.T, body string) {
	t.Helper()
	for _, href := range []string{"/app/dashboard", "/app/campaigns", "/app/notifications", "/app/settings"} {
		if !strings.Contains(body, "href=\""+href+"\"") {
			t.Fatalf("body missing nav href %q", href)
		}
	}
	if !strings.Contains(body, `action="/logout"`) {
		t.Fatalf("body missing logout form action %q", "/logout")
	}
}

func attachSessionCookie(t *testing.T, req *http.Request, auth *fakeWebAuthClient, userID string) {
	t.Helper()
	if req == nil {
		t.Fatalf("request is required")
	}
	if auth == nil {
		t.Fatalf("auth client is required")
	}
	if strings.TrimSpace(userID) == "" {
		t.Fatalf("user id is required")
	}
	resp, err := auth.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: userID})
	if err != nil {
		t.Fatalf("CreateWebSession() error = %v", err)
	}
	sessionID := strings.TrimSpace(resp.GetSession().GetId())
	if sessionID == "" {
		t.Fatalf("expected non-empty session id")
	}
	req.AddCookie(&http.Cookie{Name: "web_session", Value: sessionID})
}

func newDependencyBundle(principalDeps principal.Dependencies, moduleDeps modules.Dependencies) *DependencyBundle {
	return &DependencyBundle{
		Principal: principalDeps,
		Modules:   moduleDeps,
	}
}

func newTestHandler(cfg Config) (http.Handler, error) {
	return composeHandler(cfg, snapshotDependencyBundle(cfg.Dependencies))
}

func newTestServer(cfg Config) (*Server, error) {
	handler, err := newTestHandler(cfg)
	if err != nil {
		return nil, err
	}
	return newServer(cfg, handler)
}

func newDefaultDependencyBundle(moduleDeps modules.Dependencies) *DependencyBundle {
	return newDependencyBundle(principal.Dependencies{}, moduleDeps)
}
