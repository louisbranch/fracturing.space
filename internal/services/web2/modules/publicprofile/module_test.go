package publicprofile

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
)

func TestMountServesPublicProfileGet(t *testing.T) {
	t.Parallel()

	m := New()
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.UserProfilePrefix+"alice", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMountServesPublicProfileHead(t *testing.T) {
	t.Parallel()

	m := New()
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodHead, routepath.UserProfilePrefix+"alice", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestModuleIDReturnsPublicProfile(t *testing.T) {
	t.Parallel()

	if got := New().ID(); got != "publicprofile" {
		t.Fatalf("ID() = %q, want %q", got, "publicprofile")
	}
}

func TestMountRejectsPublicProfileNonGet(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodDelete, routepath.UserProfilePrefix+"alice", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
