package composition

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules"
)

func TestComposeAppHandlerBuildsRegistryModules(t *testing.T) {
	reg := &stubRegistry{output: modules.BuildOutput{Modules: []modules.Module{
		stubModule{id: "ok", mount: mod.Mount{Prefix: "/app/ok/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })}},
	}}}
	h, err := ComposeAppHandler(ComposeInput{Registry: reg})
	if err != nil {
		t.Fatalf("ComposeAppHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/ok/example", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

type stubRegistry struct {
	output modules.BuildOutput
}

func (s *stubRegistry) Build(modules.BuildInput) modules.BuildOutput { return s.output }

type stubModule struct {
	id    string
	mount mod.Mount
	err   error
}

func (s stubModule) ID() string { return s.id }

func (s stubModule) Mount() (mod.Mount, error) { return s.mount, s.err }
