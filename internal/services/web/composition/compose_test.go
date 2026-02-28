package composition

import (
	"net/http"
	"net/http/httptest"
	"testing"

	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

func TestComposeAppHandlerBuildsRegistryInputAndRoutes(t *testing.T) {
	t.Parallel()

	reg := &stubRegistry{
		output: modules.BuildOutput{
			Public: []modules.Module{
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
			Protected: []modules.Module{
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
		Principal: PrincipalResolvers{
			AuthRequired:    func(*http.Request) bool { return true },
			ResolveViewer:   func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "Ada"} },
			ResolveSignedIn: func(*http.Request) bool { return true },
			ResolveUserID:   func(*http.Request) string { return "user-1" },
			ResolveLanguage: func(*http.Request) string { return "en" },
		},
		EnableExperimentalModules: true,
		ChatHTTPAddr:              "127.0.0.1:9002",
		RequestSchemePolicy:       requestmeta.SchemePolicy{TrustForwardedProto: true},
		Registry:                  reg,
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

	if !reg.input.EnableExperimentalModules {
		t.Fatalf("EnableExperimentalModules = false, want true")
	}
	if !reg.input.ProtectedOptions.RequestSchemePolicy.TrustForwardedProto {
		t.Fatalf("ProtectedOptions.RequestSchemePolicy.TrustForwardedProto = false, want true")
	}
	wantPort := websupport.ResolveChatFallbackPort("127.0.0.1:9002")
	if reg.input.ProtectedOptions.ChatFallbackPort != wantPort {
		t.Fatalf("ProtectedOptions.ChatFallbackPort = %q, want %q", reg.input.ProtectedOptions.ChatFallbackPort, wantPort)
	}
}

func TestComposeAppHandlerDefaultsAuthToFalseWhenNil(t *testing.T) {
	t.Parallel()

	reg := &stubRegistry{
		output: modules.BuildOutput{
			Protected: []modules.Module{
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
		Registry: reg,
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
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}
}

type stubRegistry struct {
	input  modules.BuildInput
	output modules.BuildOutput
}

func (s *stubRegistry) Build(input modules.BuildInput) modules.BuildOutput {
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
