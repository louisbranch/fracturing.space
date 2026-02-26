package settings

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newService(staticGateway{}), settingsTestDependencies()))
}

func TestRegisterRoutesSettingsPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(staticGateway{}), settingsTestDependencies()))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{name: "settings root", method: http.MethodGet, path: routepath.AppSettings, wantStatus: http.StatusFound},
		{name: "settings slash root", method: http.MethodGet, path: routepath.SettingsPrefix, wantStatus: http.StatusFound},
		{name: "profile get", method: http.MethodGet, path: routepath.AppSettingsProfile, wantStatus: http.StatusOK},
		{name: "profile head", method: http.MethodHead, path: routepath.AppSettingsProfile, wantStatus: http.StatusOK},
		{name: "profile delete rejected", method: http.MethodDelete, path: routepath.AppSettingsProfile, wantStatus: http.StatusMethodNotAllowed},
		{name: "ai key revoke get rejected", method: http.MethodGet, path: routepath.AppSettingsAIKeyRevoke("cred-1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "unknown subpath", method: http.MethodGet, path: routepath.SettingsPrefix + "unknown", wantStatus: http.StatusNotFound},
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
