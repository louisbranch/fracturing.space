package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
)

func TestComposeRejectsDuplicateModulePrefix(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	_, err := composer.Compose(ComposeInput{
		PublicModules: []module.Module{
			stubModule{id: "one", mount: module.Mount{Prefix: "/one/", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}},
			stubModule{id: "two", mount: module.Mount{Prefix: "/one/", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}},
		},
	})
	if err == nil {
		t.Fatalf("expected duplicate prefix error")
	}
}

func TestComposeRejectsNilPublicModule(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	_, err := composer.Compose(ComposeInput{
		PublicModules: []module.Module{nil},
	})
	if err == nil {
		t.Fatalf("expected nil public module error")
	}
}

func TestComposeRejectsNilProtectedModule(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	_, err := composer.Compose(ComposeInput{
		ProtectedModules: []module.Module{nil},
	})
	if err == nil {
		t.Fatalf("expected nil protected module error")
	}
}

func TestComposeWrapsProtectedModulesWithAuth(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	h, err := composer.Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return false },
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/123", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}
}

func TestComposeMountsPublicModulesWithoutAuth(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	h, err := composer.Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return false },
		PublicModules: []module.Module{
			stubModule{id: "discover", mount: module.Mount{Prefix: "/discover/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestComposeRejectsCookieMutationWithoutSameOriginProof(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	h, err := composer.Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return true },
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/123/sessions/start", nil)
	req.AddCookie(&http.Cookie{Name: "web2_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestComposeAllowsCookieMutationWithSameOriginHeader(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	h, err := composer.Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return true },
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "https://app.example.test/app/campaigns/123/sessions/start", nil)
	req.Host = "app.example.test"
	req.Header.Set("Origin", "https://app.example.test")
	req.AddCookie(&http.Cookie{Name: "web2_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestComposeRejectsCookieMutationWhenOriginSchemeDiffers(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	h, err := composer.Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return true },
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "https://app.example.test/app/campaigns/123/sessions/start", nil)
	req.Host = "app.example.test"
	req.Header.Set("Origin", "http://app.example.test")
	req.AddCookie(&http.Cookie{Name: "web2_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestComposeRejectsCookieMutationWhenOriginOmitsNonDefaultPort(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	h, err := composer.Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return true },
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "https://app.example.test:8443/app/campaigns/123/sessions/start", nil)
	req.Host = "app.example.test:8443"
	req.Header.Set("Origin", "https://app.example.test")
	req.AddCookie(&http.Cookie{Name: "web2_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestComposeRejectsProtectedModuleOutsideAppPrefix(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	_, err := composer.Compose(ComposeInput{
		ProtectedModules: []module.Module{
			stubModule{id: "bad", mount: module.Mount{Prefix: "/discover/", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}},
		},
	})
	if err == nil {
		t.Fatalf("expected protected module prefix policy error")
	}
}

func TestComposeRejectsPublicModuleInsideAppPrefix(t *testing.T) {
	t.Parallel()

	composer := Composer{}
	_, err := composer.Compose(ComposeInput{
		PublicModules: []module.Module{
			stubModule{id: "bad", mount: module.Mount{Prefix: "/app/bad/", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}},
		},
	})
	if err == nil {
		t.Fatalf("expected public module prefix policy error")
	}
}

type stubModule struct {
	id    string
	mount module.Mount
	err   error
}

func (s stubModule) ID() string {
	return s.id
}

func (s stubModule) Mount(module.Dependencies) (module.Mount, error) {
	return s.mount, s.err
}
