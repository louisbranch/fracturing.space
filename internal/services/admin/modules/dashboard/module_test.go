package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

type fakeHandlers struct {
	lastCall string
	lastPath string
}

func (f *fakeHandlers) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	f.lastCall = "dashboard"
	if r != nil && r.URL != nil {
		f.lastPath = r.URL.Path
	}
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeHandlers) HandleDashboardContent(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "dashboard_content"
	w.WriteHeader(http.StatusNoContent)
}

func TestMount(t *testing.T) {
	t.Parallel()

	svc := &fakeHandlers{}
	m, err := New(svc).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if m.Prefix != routepath.DashboardPrefix {
		t.Fatalf("prefix = %q, want %q", m.Prefix, routepath.DashboardPrefix)
	}

	tests := []struct {
		path     string
		wantCode int
		wantCall string
		wantPath string
	}{
		{path: "/app/dashboard", wantCode: http.StatusNoContent, wantCall: "dashboard", wantPath: "/app/dashboard"},
		{path: "/app/dashboard/", wantCode: http.StatusNoContent, wantCall: "dashboard", wantPath: "/app/dashboard/"},
		{path: "/app/dashboard?fragment=rows", wantCode: http.StatusNoContent, wantCall: "dashboard_content"},
		{path: "/app/dashboard/unknown", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastPath = ""
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			m.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
			if tc.wantPath != "" && svc.lastPath != tc.wantPath {
				t.Fatalf("lastPath = %q, want %q", svc.lastPath, tc.wantPath)
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

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	rec := httptest.NewRecorder()
	m.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
