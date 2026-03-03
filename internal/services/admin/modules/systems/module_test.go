package systems

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

type fakeService struct {
	lastCall   string
	lastSystem string
}

func (f *fakeService) HandleSystemsPage(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "systems_page"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleSystemsTable(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "systems_table"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleSystemDetail(w http.ResponseWriter, _ *http.Request, systemID string) {
	f.lastCall = "systems_detail"
	f.lastSystem = systemID
	w.WriteHeader(http.StatusNoContent)
}

func TestMount(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	m, err := New(svc).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if m.Prefix != routepath.SystemsPrefix {
		t.Fatalf("prefix = %q, want %q", m.Prefix, routepath.SystemsPrefix)
	}

	tests := []struct {
		path       string
		wantCode   int
		wantCall   string
		wantSystem string
	}{
		{path: "/app/systems", wantCode: http.StatusNoContent, wantCall: "systems_page"},
		{path: "/app/systems?fragment=rows", wantCode: http.StatusNoContent, wantCall: "systems_table"},
		{path: "/app/systems/daggerheart", wantCode: http.StatusNoContent, wantCall: "systems_detail", wantSystem: "daggerheart"},
		{path: "/app/systems/daggerheart/version", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastSystem = ""
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			m.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
			if svc.lastSystem != tc.wantSystem {
				t.Fatalf("lastSystem = %q, want %q", svc.lastSystem, tc.wantSystem)
			}
		})
	}
}

func TestMountNilService(t *testing.T) {
	t.Parallel()

	m, err := New(nil).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/systems?fragment=rows", nil)
	rec := httptest.NewRecorder()
	m.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
