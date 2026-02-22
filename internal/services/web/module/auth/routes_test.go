package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakePublicService struct {
	lastCall string
}

func (f *fakePublicService) HandleRoot(http.ResponseWriter, *http.Request) {
	f.lastCall = "root"
}

func (f *fakePublicService) HandleLogin(http.ResponseWriter, *http.Request) {
	f.lastCall = "login"
}

func (f *fakePublicService) HandleAuthLogin(http.ResponseWriter, *http.Request) {
	f.lastCall = "auth_login"
}

func (f *fakePublicService) HandleAuthCallback(http.ResponseWriter, *http.Request) {
	f.lastCall = "auth_callback"
}

func (f *fakePublicService) HandleAuthLogout(http.ResponseWriter, *http.Request) {
	f.lastCall = "auth_logout"
}

func (f *fakePublicService) HandleMagicLink(http.ResponseWriter, *http.Request) {
	f.lastCall = "magic"
}

func (f *fakePublicService) HandlePasskeyRegisterStart(http.ResponseWriter, *http.Request) {
	f.lastCall = "passkey_register_start"
}

func (f *fakePublicService) HandlePasskeyRegisterFinish(http.ResponseWriter, *http.Request) {
	f.lastCall = "passkey_register_finish"
}

func (f *fakePublicService) HandlePasskeyLoginStart(http.ResponseWriter, *http.Request) {
	f.lastCall = "passkey_login_start"
}

func (f *fakePublicService) HandlePasskeyLoginFinish(http.ResponseWriter, *http.Request) {
	f.lastCall = "passkey_login_finish"
}

func (f *fakePublicService) HandleHealth(http.ResponseWriter, *http.Request) {
	f.lastCall = "up"
}

func TestRegisterPublicRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakePublicService{}
	mux := http.NewServeMux()
	RegisterPublicRoutes(mux, svc)

	tests := []struct {
		path     string
		wantCall string
	}{
		{path: "/", wantCall: "root"},
		{path: "/login", wantCall: "login"},
		{path: "/auth/login", wantCall: "auth_login"},
		{path: "/auth/callback", wantCall: "auth_callback"},
		{path: "/auth/logout", wantCall: "auth_logout"},
		{path: "/magic", wantCall: "magic"},
		{path: "/passkeys/register/start", wantCall: "passkey_register_start"},
		{path: "/passkeys/register/finish", wantCall: "passkey_register_finish"},
		{path: "/passkeys/login/start", wantCall: "passkey_login_start"},
		{path: "/passkeys/login/finish", wantCall: "passkey_login_finish"},
		{path: "/up", wantCall: "up"},
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
