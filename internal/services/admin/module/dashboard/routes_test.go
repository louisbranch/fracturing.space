package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	lastCall string
}

func (f *fakeService) HandleDashboard(http.ResponseWriter, *http.Request) {
	f.lastCall = "dashboard"
}

func (f *fakeService) HandleDashboardContent(http.ResponseWriter, *http.Request) {
	f.lastCall = "dashboard_content"
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path     string
		wantCall string
	}{
		{path: "/", wantCall: "dashboard"},
		{path: "/dashboard/content", wantCall: "dashboard_content"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			svc.lastCall = ""

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
		})
	}
}
