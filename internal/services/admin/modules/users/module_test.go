package users

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

type fakeService struct {
	lastCall string
	lastUser string
}

func (f *fakeService) HandleUsersPage(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "users_page"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleUsersTable(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "users_table"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleUserLookup(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "users_lookup"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleUserDetail(w http.ResponseWriter, _ *http.Request, userID string) {
	f.lastCall = "users_detail"
	f.lastUser = userID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleUserInvites(w http.ResponseWriter, _ *http.Request, userID string) {
	f.lastCall = "users_invites"
	f.lastUser = userID
	w.WriteHeader(http.StatusNoContent)
}

func TestMount(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	m, err := New(svc).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if m.Prefix != routepath.UsersPrefix {
		t.Fatalf("prefix = %q, want %q", m.Prefix, routepath.UsersPrefix)
	}

	tests := []struct {
		path     string
		method   string
		wantCode int
		wantCall string
		wantUser string
	}{
		{path: "/users", method: http.MethodGet, wantCode: http.StatusNoContent, wantCall: "users_page"},
		{path: "/users/_rows", method: http.MethodGet, wantCode: http.StatusNoContent, wantCall: "users_table"},
		{path: "/users/lookup", method: http.MethodGet, wantCode: http.StatusNoContent, wantCall: "users_lookup"},
		{path: "/users/u-1", method: http.MethodGet, wantCode: http.StatusNoContent, wantCall: "users_detail", wantUser: "u-1"},
		{path: "/users/u-1/invites", method: http.MethodGet, wantCode: http.StatusNoContent, wantCall: "users_invites", wantUser: "u-1"},
		{path: "/users/magic-link", method: http.MethodPost, wantCode: http.StatusNotFound},
		{path: "/users/table", method: http.MethodGet, wantCode: http.StatusNotFound},
		{path: "/users/create", method: http.MethodGet, wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastUser = ""
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			m.Handler.ServeHTTP(rec, req)
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

func TestMountNilService(t *testing.T) {
	t.Parallel()

	m, err := New(nil).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/users/_rows", nil)
	rec := httptest.NewRecorder()
	m.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
