package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
)

func TestComposeMountsSlashAndAlias(t *testing.T) {
	h, err := Compose(ComposeInput{Modules: []mod.Module{
		stubModule{id: "campaigns", mount: mod.Mount{Prefix: "/campaigns/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })}},
	}})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	for _, path := range []string{"/campaigns", "/campaigns/", "/campaigns/123"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("%s status = %d, want %d", path, rr.Code, http.StatusNoContent)
		}
	}
}

func TestComposeRejectsDuplicatePrefixes(t *testing.T) {
	_, err := Compose(ComposeInput{Modules: []mod.Module{
		stubModule{id: "a", mount: mod.Mount{Prefix: "/campaigns/", Handler: http.NotFoundHandler()}},
		stubModule{id: "b", mount: mod.Mount{Prefix: "/campaigns/", Handler: http.NotFoundHandler()}},
	}})
	if err == nil {
		t.Fatal("expected duplicate prefix error")
	}
}

type stubModule struct {
	id    string
	mount mod.Mount
	err   error
}

func (s stubModule) ID() string { return s.id }

func (s stubModule) Mount() (mod.Mount, error) { return s.mount, s.err }
