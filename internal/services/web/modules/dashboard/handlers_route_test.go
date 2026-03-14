package dashboard

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- handleIndex ---

func TestHandleIndexRendersDashboard(t *testing.T) {
	t.Parallel()

	gw := &fakeGateway{snapshot: dashboardapp.DashboardSnapshot{NeedsProfileCompletion: true}}
	h := newTestHandlers(gw)
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Dashboard") {
		t.Fatalf("body missing dashboard title")
	}
	if !strings.Contains(body, `data-dashboard-block="profile-pending"`) {
		t.Fatalf("body missing pending-profile block: %q", body)
	}
}

func TestHandleIndexHTMXRequestReturnsPartialResponse(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(&fakeGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, "<html") {
		t.Fatalf("HTMX response should not contain full HTML shell")
	}
}

func TestHandleIndexGracefullyDegradesWhenGatewayFails(t *testing.T) {
	t.Parallel()

	gw := &fakeGateway{err: errors.New("gateway down")}
	h := newTestHandlers(gw)
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Dashboard service swallows gateway errors and renders degraded view.
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (degraded render)", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `data-dashboard-status="unavailable"`) {
		t.Fatalf("body = %q, want unavailable dashboard status notice", rr.Body.String())
	}
}

func TestHandleIndexNilGatewayRendersDegradedDashboard(t *testing.T) {
	t.Parallel()

	h := newHandlers(dashboardapp.NewService(nil, nil, nil), dashboardTestBase())
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppDashboard, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `data-dashboard-status="unavailable"`) {
		t.Fatalf("body = %q, want unavailable dashboard status notice", rr.Body.String())
	}
}

// --- helpers ---

func newTestHandlers(gw *fakeGateway) handlers {
	return newHandlers(dashboardapp.NewService(gw, nil, nil), dashboardTestBase())
}

func dashboardTestBase() modulehandler.Base {
	return modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil)
}
