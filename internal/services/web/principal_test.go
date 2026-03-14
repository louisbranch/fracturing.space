package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// runWithPrincipalResolver executes a request inside the principal middleware so
// tests can assert per-request caching behavior.
func runWithPrincipalResolver(t *testing.T, resolver principal.Resolver, request *http.Request, fn func(*http.Request)) {
	t.Helper()
	if fn == nil {
		t.Fatalf("callback is required")
	}
	handler := resolver.Middleware()(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		fn(request)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), request)
}

func TestPrincipalResolverSharesAccountProfileLookupAcrossViewerAndLanguage(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{
			Profile: &authv1.AccountProfile{
				Username: "alice",
				Locale:   commonv1.Locale_LOCALE_PT_BR,
			},
		},
	}
	resolver := principal.New(principal.Dependencies{
		SessionClient: auth,
		AccountClient: account,
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, request, auth, "user-1")

	runWithPrincipalResolver(t, resolver, request, func(request *http.Request) {
		_ = resolver.ResolveViewer(request)
		_ = resolver.ResolveLanguage(request)
	})

	if account.getProfileCalls != 1 {
		t.Fatalf("GetProfile calls = %d, want %d", account.getProfileCalls, 1)
	}
}

func TestLegacyPrincipalResolverFilesAreRemoved(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"principal_resolver.go",
		"session_resolver.go",
		"viewer_resolver.go",
		"language_resolver.go",
	} {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			_, err := os.Stat(path)
			if err == nil {
				t.Fatalf("%q exists; principal resolution should stay owned by internal/services/web/principal", path)
			}
			if !os.IsNotExist(err) {
				t.Fatalf("Stat(%q) error = %v", path, err)
			}
		})
	}
}
