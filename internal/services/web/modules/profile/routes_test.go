package profile

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newService(staticGateway{}), module.Dependencies{}))
}

func TestRegisterRoutesProfilePathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(staticGateway{}), module.Dependencies{}))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantLoc    string
		wantAllow  string
	}{
		{name: "app profile get", method: http.MethodGet, path: routepath.AppProfile, wantStatus: http.StatusOK},
		{name: "app profile head", method: http.MethodHead, path: routepath.AppProfile, wantStatus: http.StatusOK},
		{name: "profile prefix get", method: http.MethodGet, path: routepath.ProfilePrefix, wantStatus: http.StatusOK},
		{name: "profile unknown subpath", method: http.MethodGet, path: routepath.ProfilePrefix + "other", wantStatus: http.StatusNotFound},
		{name: "profile prefix post rejected", method: http.MethodPost, path: routepath.ProfilePrefix, wantStatus: http.StatusMethodNotAllowed, wantAllow: "GET, HEAD"},
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
			if tc.wantLoc != "" {
				if got := rr.Header().Get("Location"); got != tc.wantLoc {
					t.Fatalf("Location = %q, want %q", got, tc.wantLoc)
				}
			}
			if tc.wantAllow != "" {
				if got := rr.Header().Get("Allow"); got != tc.wantAllow {
					t.Fatalf("Allow = %q, want %q", got, tc.wantAllow)
				}
			}
		})
	}
}
