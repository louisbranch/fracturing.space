package status

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

type fakeService struct {
	lastCall string
}

func (f *fakeService) HandleStatusPage(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "status_page"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleStatusTable(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "status_table"
	w.WriteHeader(http.StatusNoContent)
}

func TestMount(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	m, err := New(svc).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if m.Prefix != routepath.StatusPrefix {
		t.Fatalf("prefix = %q, want %q", m.Prefix, routepath.StatusPrefix)
	}

	tests := []struct {
		path     string
		wantCode int
		wantCall string
	}{
		{path: "/app/status", wantCode: http.StatusNoContent, wantCall: "status_page"},
		{path: "/app/status?fragment=rows", wantCode: http.StatusNoContent, wantCall: "status_table"},
		{path: "/app/status/unknown", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			m.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
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

	req := httptest.NewRequest(http.MethodGet, "/app/status?fragment=rows", nil)
	rec := httptest.NewRecorder()
	m.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
