package settings

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeSettingsService struct {
	lastCall string
}

func (f *fakeSettingsService) HandleSettings(http.ResponseWriter, *http.Request) {
	f.lastCall = "settings"
}

func (f *fakeSettingsService) HandleSettingsSubroutes(http.ResponseWriter, *http.Request) {
	f.lastCall = "settings_subroutes"
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeSettingsService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path     string
		wantCall string
	}{
		{path: "/app/settings", wantCall: "settings"},
		{path: "/app/settings/username", wantCall: "settings_subroutes"},
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
