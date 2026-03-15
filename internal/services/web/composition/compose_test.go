package composition

import (
	"net/http"
	"net/http/httptest"
	"testing"

	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
)

func TestComposeAppHandlerBuildsRegistryInputAndRoutes(t *testing.T) {
	t.Parallel()

	reg := &stubRegistry{
		output: modules.RegistryOutput{
			Public: []module.Module{
				stubModule{
					id: "public",
					mount: module.Mount{
						Prefix: "/",
						Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(http.StatusNoContent)
						}),
					},
				},
			},
			Protected: []module.Module{
				stubModule{
					id: "campaigns",
					mount: module.Mount{
						Prefix: "/app/campaigns/",
						Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(http.StatusNoContent)
						}),
					},
				},
			},
		},
	}

	h, err := ComposeAppHandler(ComposeInput{
		Principal: requestresolver.NewPrincipal(
			func(*http.Request) bool { return true },
			func(*http.Request) bool { return true },
			func(*http.Request) string { return "user-1" },
			func(*http.Request) string { return "en" },
			func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "Ada"} },
		),
		PlayHTTPAddr:        "127.0.0.1:9004",
		RequestSchemePolicy: requestmeta.SchemePolicy{TrustForwardedProto: true},
		RegistryBuilder:     reg,
	})
	if err != nil {
		t.Fatalf("ComposeAppHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}

	if !reg.input.ProtectedOptions.RequestSchemePolicy.TrustForwardedProto {
		t.Fatalf("ProtectedOptions.RequestSchemePolicy.TrustForwardedProto = false, want true")
	}
	if !reg.input.Principal.AuthRequired(req) {
		t.Fatalf("RegistryInput.Principal.AuthRequired() = false, want true")
	}
	if !reg.input.Principal.ResolveSignedIn(req) {
		t.Fatalf("RegistryInput.Principal.ResolveSignedIn() = false, want true")
	}
	if got := reg.input.Principal.ResolveUserID(req); got != "user-1" {
		t.Fatalf("RegistryInput.Principal.ResolveUserID() = %q, want %q", got, "user-1")
	}
	if got := reg.input.Principal.ResolveRequestLanguage(req); got != "en" {
		t.Fatalf("RegistryInput.Principal.ResolveRequestLanguage() = %q, want %q", got, "en")
	}
	if got := reg.input.Principal.ResolveRequestViewer(req).DisplayName; got != "Ada" {
		t.Fatalf("RegistryInput.Principal.ResolveRequestViewer().DisplayName = %q, want %q", got, "Ada")
	}
	wantPlayPort := websupport.ResolveHTTPFallbackPort("127.0.0.1:9004")
	if reg.input.ProtectedOptions.PlayFallbackPort != wantPlayPort {
		t.Fatalf("ProtectedOptions.PlayFallbackPort = %q, want %q", reg.input.ProtectedOptions.PlayFallbackPort, wantPlayPort)
	}
}

func TestComposeAppHandlerDefaultsAuthToFalseWhenNil(t *testing.T) {
	t.Parallel()

	reg := &stubRegistry{
		output: modules.RegistryOutput{
			Protected: []module.Module{
				stubModule{
					id: "campaigns",
					mount: module.Mount{
						Prefix: "/app/campaigns/",
						Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.WriteHeader(http.StatusNoContent)
						}),
					},
				},
			},
		},
	}

	h, err := ComposeAppHandler(ComposeInput{
		RegistryBuilder: reg,
	})
	if err != nil {
		t.Fatalf("ComposeAppHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/1", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/login?next=%2Fapp%2Fcampaigns%2F1" {
		t.Fatalf("Location = %q, want %q", got, "/login?next=%2Fapp%2Fcampaigns%2F1")
	}
}

func TestComposeAppHandlerUsesDefaultRegistryBuilderWhenNil(t *testing.T) {
	t.Parallel()

	h, err := ComposeAppHandler(ComposeInput{})
	if err != nil {
		t.Fatalf("ComposeAppHandler() error = %v", err)
	}
	if h == nil {
		t.Fatal("ComposeAppHandler() handler = nil, want non-nil")
	}
}

func TestComposeAppHandlerReturnsComposeError(t *testing.T) {
	t.Parallel()

	_, err := ComposeAppHandler(ComposeInput{
		RegistryBuilder: &stubRegistry{
			output: modules.RegistryOutput{
				Public: []module.Module{
					stubModule{
						id: "broken",
						mount: module.Mount{
							Prefix:  "/app/broken/",
							Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
						},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("ComposeAppHandler() error = nil, want non-nil")
	}
}

type stubRegistry struct {
	input  modules.RegistryInput
	output modules.RegistryOutput
}

func (s *stubRegistry) Build(input modules.RegistryInput) modules.RegistryOutput {
	s.input = input
	return s.output
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
