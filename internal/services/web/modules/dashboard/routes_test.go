package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newService(nil), module.Dependencies{}))
}

func TestRegisterRoutesDashboardPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(nil), module.Dependencies{}))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{name: "app dashboard get", method: http.MethodGet, path: routepath.AppDashboard, wantStatus: http.StatusOK},
		{name: "app dashboard head", method: http.MethodHead, path: routepath.AppDashboard, wantStatus: http.StatusOK},
		{name: "dashboard prefix get", method: http.MethodGet, path: routepath.DashboardPrefix, wantStatus: http.StatusOK},
		{name: "dashboard unknown subpath", method: http.MethodGet, path: routepath.DashboardPrefix + "other", wantStatus: http.StatusNotFound},
		{name: "dashboard post rejected", method: http.MethodPost, path: routepath.DashboardPrefix, wantStatus: http.StatusMethodNotAllowed, wantAllow: "GET, HEAD"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantAllow != "" {
				if got := rr.Header().Get("Allow"); got != tc.wantAllow {
					t.Fatalf("Allow = %q, want %q", got, tc.wantAllow)
				}
			}
		})
	}
}
