package invites

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeInvitesService struct {
	lastCall string
}

func (f *fakeInvitesService) HandleInvites(http.ResponseWriter, *http.Request) {
	f.lastCall = "invites"
}

func (f *fakeInvitesService) HandleInviteClaim(http.ResponseWriter, *http.Request) {
	f.lastCall = "claim"
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeInvitesService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path     string
		wantCall string
	}{
		{path: "/app/invites", wantCall: "invites"},
		{path: "/app/invites/claim", wantCall: "claim"},
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
