package users

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	lastCall string
	lastUser string
}

func (f *fakeService) HandleUsersPage(http.ResponseWriter, *http.Request) {
	f.lastCall = "users_page"
}

func (f *fakeService) HandleUsersTable(http.ResponseWriter, *http.Request) {
	f.lastCall = "users_table"
}

func (f *fakeService) HandleUserLookup(http.ResponseWriter, *http.Request) {
	f.lastCall = "users_lookup"
}

func (f *fakeService) HandleMagicLink(http.ResponseWriter, *http.Request) {
	f.lastCall = "users_magic_link"
}

func (f *fakeService) HandleUserDetail(_ http.ResponseWriter, _ *http.Request, userID string) {
	f.lastCall = "users_detail"
	f.lastUser = userID
}

func (f *fakeService) HandleUserInvites(_ http.ResponseWriter, _ *http.Request, userID string) {
	f.lastCall = "users_invites"
	f.lastUser = userID
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path     string
		method   string
		wantCode int
		wantCall string
		wantUser string
	}{
		{path: "/users", method: http.MethodGet, wantCode: http.StatusOK, wantCall: "users_page"},
		{path: "/users/table", method: http.MethodGet, wantCode: http.StatusOK, wantCall: "users_table"},
		{path: "/users/lookup", method: http.MethodGet, wantCode: http.StatusOK, wantCall: "users_lookup"},
		{path: "/users/magic-link", method: http.MethodPost, wantCode: http.StatusOK, wantCall: "users_magic_link"},
		{path: "/users/u-1", method: http.MethodGet, wantCode: http.StatusOK, wantCall: "users_detail", wantUser: "u-1"},
		{path: "/users/u-1/invites", method: http.MethodGet, wantCode: http.StatusOK, wantCall: "users_invites", wantUser: "u-1"},
		{path: "/users/create", method: http.MethodGet, wantCode: http.StatusNotFound},
		{path: "/users/u-1/invites/extra", method: http.MethodGet, wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastUser = ""

			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
			if svc.lastUser != tc.wantUser {
				t.Fatalf("lastUser = %q, want %q", svc.lastUser, tc.wantUser)
			}
		})
	}
}

func TestHandleUserPathRedirectsTrailingSlash(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodGet, "/users/u-1/", nil)
	rec := httptest.NewRecorder()

	HandleUserPath(rec, req, svc)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if location := rec.Header().Get("Location"); location != "/users/u-1" {
		t.Fatalf("location = %q, want %q", location, "/users/u-1")
	}
}
