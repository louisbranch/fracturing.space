package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

func TestComposeRejectsDuplicateModulePrefix(t *testing.T) {
	t.Parallel()

	_, err := Compose(ComposeInput{
		PublicModules: []module.Module{
			stubModule{id: "one", mount: module.Mount{Prefix: "/one/", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}},
			stubModule{id: "two", mount: module.Mount{Prefix: "/one/", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}},
		},
	})
	if err == nil {
		t.Fatalf("expected duplicate prefix error")
	}
}

func TestComposeRejectsInvalidPublicModulePrefixes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prefix string
	}{
		{name: "missing leading slash", prefix: "app/x"},
		{name: "missing trailing slash", prefix: "/app/x"},
		{name: "contains surrounding whitespace", prefix: "/app/x "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Compose(ComposeInput{
				PublicModules: []module.Module{
					stubModule{id: "bad", mount: module.Mount{Prefix: tc.prefix, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}},
				},
			})
			if err == nil {
				t.Fatalf("expected invalid prefix error")
			}
			if got := err.Error(); !strings.Contains(got, "invalid prefix") || !strings.Contains(got, tc.prefix) || !strings.Contains(got, "bad") {
				t.Fatalf("unexpected error = %q", got)
			}
		})
	}
}

func TestComposeRejectsNilPublicModule(t *testing.T) {
	t.Parallel()

	_, err := Compose(ComposeInput{
		PublicModules: []module.Module{nil},
	})
	if err == nil {
		t.Fatalf("expected nil public module error")
	}
}

func TestComposeRejectsNilProtectedModule(t *testing.T) {
	t.Parallel()

	_, err := Compose(ComposeInput{
		ProtectedModules: []module.Module{nil},
	})
	if err == nil {
		t.Fatalf("expected nil protected module error")
	}
}

func TestComposeWrapsProtectedModulesWithAuth(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
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

func TestComposeWrapsProtectedModulesWithAuthForHtmxRequest(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
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
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != "/login" {
		t.Fatalf("HX-Redirect = %q, want %q", got, "/login")
	}
	if got := rr.Header().Get("Location"); got != "" {
		t.Fatalf("Location = %q, want empty", got)
	}
}

func TestComposeProtectsSlashlessProtectedRootBeforePublicFallback(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return false },
		PublicModules: []module.Module{
			stubModule{id: "public", mount: module.Mount{Prefix: "/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})}},
		},
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
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

	h, err := Compose(ComposeInput{
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

	h, err := Compose(ComposeInput{
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
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestComposeAllowsCookieMutationWithSameOriginHeader(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
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
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestComposeRejectsCookieMutationWhenOriginSchemeDiffers(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
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
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestComposeRejectsCookieMutationWhenForwardedProtoMismatchesWithDefaultPolicy(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
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

	req := httptest.NewRequest(http.MethodPost, "http://app.example.test/app/campaigns/123/sessions/start", nil)
	req.Host = "app.example.test"
	req.Header.Set("Origin", "https://app.example.test")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestComposeAllowsCookieMutationWhenForwardedProtoTrustEnabled(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
		AuthRequired:        func(*http.Request) bool { return true },
		RequestSchemePolicy: requestmeta.SchemePolicy{TrustForwardedProto: true},
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://app.example.test/app/campaigns/123/sessions/start", nil)
	req.Host = "app.example.test"
	req.Header.Set("Origin", "https://app.example.test")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestComposeRejectsCookieMutationWhenOriginOmitsNonDefaultPort(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
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
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestComposeRejectsProtectedModuleOutsideAppPrefix(t *testing.T) {
	t.Parallel()

	_, err := Compose(ComposeInput{
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

	_, err := Compose(ComposeInput{
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

func (s stubModule) Mount() (module.Mount, error) {
	return s.mount, s.err
}
