package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

func TestBuildRootHandlerInjectsDefaultDependencies(t *testing.T) {
	t.Parallel()

	inspector := &captureDepsModule{
		id:     "inspect",
		prefix: "/inspect/",
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	h, err := BuildRootHandler(Config{PublicModules: []module.Module{inspector}}, nil)
	if err != nil {
		t.Fatalf("BuildRootHandler() error = %v", err)
	}
	if inspector.captured.ResolveLanguage == nil {
		t.Fatalf("expected ResolveLanguage default")
	}
	if inspector.captured.ResolveViewer == nil {
		t.Fatalf("expected ResolveViewer default")
	}

	req := httptest.NewRequest(http.MethodGet, "/inspect/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestBuildRootHandlerAppliesAuthToProtectedModules(t *testing.T) {
	t.Parallel()

	protected := &captureDepsModule{
		id:     "protected",
		prefix: "/app/protected/",
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	authRequired := func(r *http.Request) bool {
		return r.Header.Get("X-Allow") == "yes"
	}

	h, err := BuildRootHandler(Config{ProtectedModules: []module.Module{protected}}, authRequired)
	if err != nil {
		t.Fatalf("BuildRootHandler() error = %v", err)
	}

	blockedReq := httptest.NewRequest(http.MethodGet, "/app/protected/a", nil)
	blockedRR := httptest.NewRecorder()
	h.ServeHTTP(blockedRR, blockedReq)
	if blockedRR.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", blockedRR.Code, http.StatusFound)
	}
	if got := blockedRR.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}

	allowedReq := httptest.NewRequest(http.MethodGet, "/app/protected/a", nil)
	allowedReq.Header.Set("X-Allow", "yes")
	allowedRR := httptest.NewRecorder()
	h.ServeHTTP(allowedRR, allowedReq)
	if allowedRR.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", allowedRR.Code, http.StatusNoContent)
	}
}

type captureDepsModule struct {
	id       string
	prefix   string
	handler  http.Handler
	captured module.Dependencies
}

func (m *captureDepsModule) ID() string { return m.id }

func (m *captureDepsModule) Mount(deps module.Dependencies) (module.Mount, error) {
	m.captured = deps
	return module.Mount{Prefix: m.prefix, Handler: m.handler}, nil
}
