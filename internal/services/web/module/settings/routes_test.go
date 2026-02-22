package settings

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeSettingsService struct {
	lastCall       string
	lastCredential string
}

func (f *fakeSettingsService) HandleSettings(http.ResponseWriter, *http.Request) {
	f.lastCall = "settings"
}

func (f *fakeSettingsService) HandleUserProfileSettings(http.ResponseWriter, *http.Request) {
	f.lastCall = "user_profile"
}

func (f *fakeSettingsService) HandleAIKeys(http.ResponseWriter, *http.Request) {
	f.lastCall = "ai_keys"
}

func (f *fakeSettingsService) HandleAIKeyRevoke(http.ResponseWriter, *http.Request, string) {
	f.lastCall = "ai_key_revoke"
}

func (f *fakeSettingsService) HandleAIKeyRevokeWithID(_ http.ResponseWriter, _ *http.Request, credentialID string) {
	f.lastCall = "ai_key_revoke"
	f.lastCredential = credentialID
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
		{path: "/app/settings/user-profile", wantCall: "user_profile"},
		{path: "/app/settings/ai-keys", wantCall: "ai_keys"},
		{path: "/app/settings/ai-keys/key-1/revoke", wantCall: "ai_key_revoke"},
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

func TestHandleSettingsSubpathDispatchesRevokeID(t *testing.T) {
	t.Parallel()

	svc := &fakeSettingsService{}
	req := httptest.NewRequest(http.MethodPost, "/app/settings/ai-keys/cred-42/revoke", nil)
	rec := httptest.NewRecorder()

	HandleSettingsSubpath(rec, req, settingsServiceWithRevokeID{s: svc})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if svc.lastCall != "ai_key_revoke" {
		t.Fatalf("lastCall = %q, want %q", svc.lastCall, "ai_key_revoke")
	}
	if svc.lastCredential != "cred-42" {
		t.Fatalf("lastCredential = %q, want %q", svc.lastCredential, "cred-42")
	}
}

type settingsServiceWithRevokeID struct {
	s *fakeSettingsService
}

func (s settingsServiceWithRevokeID) HandleSettings(w http.ResponseWriter, r *http.Request) {
	s.s.HandleSettings(w, r)
}

func (s settingsServiceWithRevokeID) HandleUserProfileSettings(w http.ResponseWriter, r *http.Request) {
	s.s.HandleUserProfileSettings(w, r)
}

func (s settingsServiceWithRevokeID) HandleAIKeys(w http.ResponseWriter, r *http.Request) {
	s.s.HandleAIKeys(w, r)
}

func (s settingsServiceWithRevokeID) HandleAIKeyRevoke(w http.ResponseWriter, r *http.Request, credentialID string) {
	s.s.HandleAIKeyRevokeWithID(w, r, credentialID)
}
