package publicauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newServiceWithGateway(nil), requestmeta.SchemePolicy{}))
}

func TestRegisterRoutesPublicPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newServiceWithGateway(nil), requestmeta.SchemePolicy{}))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{name: "root get", method: http.MethodGet, path: routepath.Root, wantStatus: http.StatusOK},
		{name: "root head", method: http.MethodHead, path: routepath.Root, wantStatus: http.StatusOK},
		{name: "login get", method: http.MethodGet, path: routepath.Login, wantStatus: http.StatusOK},
		{name: "login head", method: http.MethodHead, path: routepath.Login, wantStatus: http.StatusOK},
		{name: "health get", method: http.MethodGet, path: routepath.Health, wantStatus: http.StatusOK},
		{name: "logout get rejected", method: http.MethodGet, path: routepath.Logout, wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "logout post", method: http.MethodPost, path: routepath.Logout, wantStatus: http.StatusFound},
		{name: "passkey login start get rejected", method: http.MethodGet, path: routepath.PasskeyLoginStart, wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "passkey login start post", method: http.MethodPost, path: routepath.PasskeyLoginStart, wantStatus: http.StatusServiceUnavailable},
		{name: "unknown get path", method: http.MethodGet, path: "/unknown", wantStatus: http.StatusNotFound},
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
