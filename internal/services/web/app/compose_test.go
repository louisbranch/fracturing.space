package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
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
	if got := rr.Header().Get("Location"); got != "/login?next=%2Fapp%2Fcampaigns%2F123" {
		t.Fatalf("Location = %q, want %q", got, "/login?next=%2Fapp%2Fcampaigns%2F123")
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
	if got := rr.Header().Get("HX-Redirect"); got != "/login?next=%2Fapp%2Fcampaigns%2F123" {
		t.Fatalf("HX-Redirect = %q, want %q", got, "/login?next=%2Fapp%2Fcampaigns%2F123")
	}
	if got := rr.Header().Get("Location"); got != "" {
		t.Fatalf("Location = %q, want empty", got)
	}
}

func TestComposeCanonicalizesSlashlessProtectedRootBeforePublicFallback(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return false },
		PublicModules: []module.Module{
			stubModule{id: "public", mount: module.Mount{Prefix: "/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})}},
		},
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", CanonicalRoot: true, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	if got := rr.Header().Get("Location"); got != "/login?next=%2Fapp%2Fcampaigns" {
		t.Fatalf("Location = %q, want %q", got, "/login?next=%2Fapp%2Fcampaigns")
	}
}

func TestComposeRedirectsSlashfulProtectedRootToCanonicalPath(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return true },
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", CanonicalRoot: true, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPermanentRedirect)
	}
	if got := rr.Header().Get("Location"); got != "/app/campaigns" {
		t.Fatalf("Location = %q, want %q", got, "/app/campaigns")
	}
}

func TestComposeRedirectsSlashfulNestedPathToCanonicalPath(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
		PublicModules: []module.Module{
			stubModule{id: "discover", mount: module.Mount{Prefix: "/discover/", CanonicalRoot: true, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns/?tab=starter", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPermanentRedirect)
	}
	if got := rr.Header().Get("Location"); got != "/discover/campaigns?tab=starter" {
		t.Fatalf("Location = %q, want %q", got, "/discover/campaigns?tab=starter")
	}
}

func TestComposeUsesHXRedirectForSlashfulHTMXGet(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
		PublicModules: []module.Module{
			stubModule{id: "discover", mount: module.Mount{Prefix: "/discover/", CanonicalRoot: true, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/discover/", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != "/discover" {
		t.Fatalf("HX-Redirect = %q, want %q", got, "/discover")
	}
}

func TestComposeUsesPermanentRedirectForSlashfulHTMXPost(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
		AuthRequired: func(*http.Request) bool { return true },
		ProtectedModules: []module.Module{
			stubModule{id: "campaigns", mount: module.Mount{Prefix: "/app/campaigns/", CanonicalRoot: true, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/c1/sessions/", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPermanentRedirect)
	}
	if got := rr.Header().Get("Location"); got != "/app/campaigns/c1/sessions" {
		t.Fatalf("Location = %q, want %q", got, "/app/campaigns/c1/sessions")
	}
	if got := rr.Header().Get("HX-Redirect"); got != "" {
		t.Fatalf("HX-Redirect = %q, want empty", got)
	}
}

func TestComposeLeavesSlashfulExactPrefixForRootlessModules(t *testing.T) {
	t.Parallel()

	h, err := Compose(ComposeInput{
		PublicModules: []module.Module{
			stubModule{id: "invite", mount: module.Mount{Prefix: "/invite/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})}},
		},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/invite/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if got := rr.Header().Get("Location"); got != "" {
		t.Fatalf("Location = %q, want empty", got)
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
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing styled error page: %q", body)
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

func TestComposeRejectsRootMountClaimingCanonicalRoot(t *testing.T) {
	t.Parallel()

	_, err := Compose(ComposeInput{
		PublicModules: []module.Module{
			stubModule{id: "public", mount: module.Mount{Prefix: "/", CanonicalRoot: true, Handler: http.NotFoundHandler()}},
		},
	})
	if err == nil {
		t.Fatalf("expected canonical root validation error")
	}
}

func TestMountModuleIgnoresNilInputs(t *testing.T) {
	t.Parallel()

	mount := module.Mount{
		Prefix:  "/discover/",
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	}

	if err := mountModule(nil, stubModule{id: "discover"}, mount, mount.Prefix, map[string]string{}, nil); err != nil {
		t.Fatalf("mountModule(nil root) error = %v", err)
	}
	if err := mountModule(http.NewServeMux(), nil, mount, mount.Prefix, map[string]string{}, nil); err != nil {
		t.Fatalf("mountModule(nil feature) error = %v", err)
	}
}

func TestMountModuleRejectsInvalidCanonicalRoot(t *testing.T) {
	t.Parallel()

	err := mountModule(
		http.NewServeMux(),
		stubModule{id: "public"},
		module.Mount{Prefix: "/", CanonicalRoot: true, Handler: http.NotFoundHandler()},
		"/",
		map[string]string{},
		nil,
	)
	if err == nil {
		t.Fatalf("expected invalid canonical root error")
	}
}

func TestMountModuleRejectsDuplicateCanonicalRootClaim(t *testing.T) {
	t.Parallel()

	err := mountModule(
		http.NewServeMux(),
		stubModule{id: "discover"},
		module.Mount{Prefix: "/discover/", CanonicalRoot: true, Handler: http.NotFoundHandler()},
		"/discover/",
		map[string]string{"/discover": "existing"},
		nil,
	)
	if err == nil {
		t.Fatalf("expected duplicate canonical root claim error")
	}
}

func TestMountProtectedModuleAllowsNilWrapper(t *testing.T) {
	t.Parallel()

	root := http.NewServeMux()
	err := mountProtectedModule(root, stubModule{
		id: "campaigns",
		mount: module.Mount{
			Prefix:  "/app/campaigns/",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		},
	}, map[string]string{}, nil)
	if err != nil {
		t.Fatalf("mountProtectedModule() error = %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/c1", nil)
	root.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestResolveMountValidatesModuleContracts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		feature module.Module
	}{
		{name: "nil module", feature: nil},
		{name: "mount error", feature: stubModule{id: "bad", err: http.ErrAbortHandler}},
		{name: "missing handler", feature: stubModule{id: "bad", mount: module.Mount{Prefix: "/discover/"}}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := resolveMount(tc.feature); err == nil {
				t.Fatalf("expected resolveMount error")
			}
		})
	}
}

func TestCanonicalizeTrailingSlashHandlesNilNextAndNilRequest(t *testing.T) {
	t.Parallel()

	h := canonicalizeTrailingSlash("/discover/", true, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestCanonicalizeTrailingSlashPassesThroughNilURLAndRootlessPrefix(t *testing.T) {
	t.Parallel()

	calls := 0
	h := canonicalizeTrailingSlash("/invite/", false, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}))

	nilURLReq := httptest.NewRequest(http.MethodGet, "/invite", nil)
	nilURLReq.URL = nil
	nilURLRR := httptest.NewRecorder()
	h.ServeHTTP(nilURLRR, nilURLReq)
	if nilURLRR.Code != http.StatusNoContent {
		t.Fatalf("nil URL status = %d, want %d", nilURLRR.Code, http.StatusNoContent)
	}

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRR := httptest.NewRecorder()
	h.ServeHTTP(rootRR, rootReq)
	if rootRR.Code != http.StatusNoContent {
		t.Fatalf("root status = %d, want %d", rootRR.Code, http.StatusNoContent)
	}

	exactPrefixReq := httptest.NewRequest(http.MethodGet, "/invite/", nil)
	exactPrefixRR := httptest.NewRecorder()
	h.ServeHTTP(exactPrefixRR, exactPrefixReq)
	if exactPrefixRR.Code != http.StatusNoContent {
		t.Fatalf("exact prefix status = %d, want %d", exactPrefixRR.Code, http.StatusNoContent)
	}

	if calls != 3 {
		t.Fatalf("calls = %d, want %d", calls, 3)
	}
}

func TestCanonicalizeTrailingSlashSkipsRootMountedHandlers(t *testing.T) {
	t.Parallel()

	h := canonicalizeTrailingSlash(routepath.Root, false, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if got := rr.Header().Get("Location"); got != "" {
		t.Fatalf("Location = %q, want empty", got)
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
