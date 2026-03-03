package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
)

func TestComposeMountsSlashAndAlias(t *testing.T) {
	h, err := Compose(ComposeInput{Modules: []mod.Module{
		stubModule{id: "campaigns", mount: mod.Mount{Prefix: "/app/campaigns/", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })}},
	}})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	tests := []struct {
		path         string
		wantCode     int
		wantLocation string
	}{
		{path: "/app/campaigns", wantCode: http.StatusTemporaryRedirect, wantLocation: "/app/campaigns/"},
		{path: "/app/campaigns/", wantCode: http.StatusNoContent},
		{path: "/app/campaigns/123", wantCode: http.StatusNoContent},
	}
	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != tc.wantCode {
			t.Fatalf("%s status = %d, want %d", tc.path, rr.Code, tc.wantCode)
		}
		if tc.wantLocation != "" && rr.Header().Get("Location") != tc.wantLocation {
			t.Fatalf("%s location = %q, want %q", tc.path, rr.Header().Get("Location"), tc.wantLocation)
		}
	}
}

func TestComposeRejectsDuplicatePrefixes(t *testing.T) {
	_, err := Compose(ComposeInput{Modules: []mod.Module{
		stubModule{id: "a", mount: mod.Mount{Prefix: "/app/campaigns/", Handler: http.NotFoundHandler()}},
		stubModule{id: "b", mount: mod.Mount{Prefix: "/app/campaigns/", Handler: http.NotFoundHandler()}},
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
