package systems

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	lastCall   string
	lastSystem string
}

func (f *fakeService) HandleSystemsPage(http.ResponseWriter, *http.Request) {
	f.lastCall = "systems_page"
}

func (f *fakeService) HandleSystemsTable(http.ResponseWriter, *http.Request) {
	f.lastCall = "systems_table"
}

func (f *fakeService) HandleSystemDetail(_ http.ResponseWriter, _ *http.Request, systemID string) {
	f.lastCall = "systems_detail"
	f.lastSystem = systemID
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path       string
		wantCode   int
		wantCall   string
		wantSystem string
	}{
		{path: "/systems", wantCode: http.StatusOK, wantCall: "systems_page"},
		{path: "/systems/table", wantCode: http.StatusOK, wantCall: "systems_table"},
		{path: "/systems/daggerheart", wantCode: http.StatusOK, wantCall: "systems_detail", wantSystem: "daggerheart"},
		{path: "/systems/daggerheart/version", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastSystem = ""

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

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

func TestHandleSystemPathRedirectsTrailingSlash(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodGet, "/systems/daggerheart/", nil)
	rec := httptest.NewRecorder()

	HandleSystemPath(rec, req, svc)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if location := rec.Header().Get("Location"); location != "/systems/daggerheart" {
		t.Fatalf("location = %q, want %q", location, "/systems/daggerheart")
	}
}
