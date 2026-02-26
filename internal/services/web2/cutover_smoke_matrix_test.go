package web2

import (
	"net/http"
	"net/http/httptest"
	"testing"

	legacyweb "github.com/louisbranch/fracturing.space/internal/services/web"
)

func TestCutoverSmokeMatrixUnauthenticatedRoutes(t *testing.T) {
	t.Parallel()

	legacyHandler := legacyweb.NewHandler(legacyweb.Config{AuthBaseURL: "http://auth.local"}, nil)
	modernHandler, err := NewHandler(defaultStableProtectedConfig(newFakeWeb2AuthClient()))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	testCases := []struct {
		name         string
		method       string
		legacyPath   string
		modernPath   string
		legacyStatus int
		modernStatus int
	}{
		{
			name:         "campaigns route enforces session",
			method:       http.MethodGet,
			legacyPath:   "/app/campaigns",
			modernPath:   "/app/campaigns/",
			legacyStatus: http.StatusFound,
			modernStatus: http.StatusFound,
		},
		{
			name:         "settings route enforces session",
			method:       http.MethodGet,
			legacyPath:   "/app/settings",
			modernPath:   "/app/settings/",
			legacyStatus: http.StatusFound,
			modernStatus: http.StatusFound,
		},
		{
			name:         "invites route parity gap is explicit",
			method:       http.MethodGet,
			legacyPath:   "/app/invites",
			modernPath:   "/app/invites",
			legacyStatus: http.StatusFound,
			modernStatus: http.StatusNotFound,
		},
		{
			name:         "notifications route parity gap is explicit",
			method:       http.MethodGet,
			legacyPath:   "/app/notifications",
			modernPath:   "/app/notifications",
			legacyStatus: http.StatusFound,
			modernStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			legacyReq := httptest.NewRequest(tc.method, tc.legacyPath, nil)
			legacyRR := httptest.NewRecorder()
			legacyHandler.ServeHTTP(legacyRR, legacyReq)
			if legacyRR.Code != tc.legacyStatus {
				t.Fatalf("legacy status for %q = %d, want %d", tc.legacyPath, legacyRR.Code, tc.legacyStatus)
			}

			modernReq := httptest.NewRequest(tc.method, tc.modernPath, nil)
			modernRR := httptest.NewRecorder()
			modernHandler.ServeHTTP(modernRR, modernReq)
			if modernRR.Code != tc.modernStatus {
				t.Fatalf("modern status for %q = %d, want %d", tc.modernPath, modernRR.Code, tc.modernStatus)
			}
		})
	}
}
