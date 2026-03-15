package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

type stubGateway struct{}

func (stubGateway) ListStarterEntries(context.Context) ([]discoveryapp.StarterEntry, error) {
	return nil, nil
}

func TestMountServesDiscoveryGet(t *testing.T) {
	t.Parallel()

	m := New(Config{})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.DiscoverPrefix+"campaigns", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", rr.Header().Get("Content-Type"), "text/html; charset=utf-8")
	}
	if got := rr.Body.String(); !strings.Contains(got, "discover-root") {
		t.Fatalf("body = %q", got)
	}
}

func TestMountServesDiscoveryHead(t *testing.T) {
	t.Parallel()

	m := New(Config{})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodHead, routepath.DiscoverPrefix+"campaigns", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestModuleIDReturnsDiscovery(t *testing.T) {
	t.Parallel()

	if got := New(Config{}).ID(); got != "discovery" {
		t.Fatalf("ID() = %q, want %q", got, "discovery")
	}
}

func TestMountRejectsDiscoveryNonGet(t *testing.T) {
	t.Parallel()

	m := New(Config{})
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodDelete, routepath.DiscoverPrefix+"campaigns", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
