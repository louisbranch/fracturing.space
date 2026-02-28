package notifications

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- handleIndex ---

func TestHandleIndexRendersNotifications(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(staticGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppNotifications, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "notification-1") {
		t.Fatalf("body missing notification ID link")
	}
}

func TestHandleIndexReturnsErrorWhenGatewayFails(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(unavailableGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppNotifications, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleIndexHTMXRequestReturnsPartialResponse(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(staticGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppNotifications, nil)
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

// --- handleDetail ---

func TestHandleDetailRendersNotificationDetail(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(staticGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppNotification("notification-1"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "notification-1") {
		t.Fatalf("body missing notification detail content")
	}
}

func TestHandleDetailReturnsNotFoundForMissingNotification(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(staticGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppNotification("does-not-exist"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDetailReturnsErrorWhenGatewayFails(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(unavailableGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppNotification("notification-1"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

// --- handleOpen ---

func TestHandleOpenRedirectsAfterOpening(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(staticGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodPost, routepath.AppNotificationOpen("notification-1"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppNotification("notification-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppNotification("notification-1"))
	}
}

func TestHandleOpenReturnsNotFoundForMissingNotification(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(staticGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodPost, routepath.AppNotificationOpen("does-not-exist"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleOpenReturnsErrorWhenGatewayFails(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(unavailableGateway{})
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	req := httptest.NewRequest(http.MethodPost, routepath.AppNotificationOpen("notification-1"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

// --- helpers ---

func newTestHandlers(gw NotificationGateway) handlers {
	return newHandlers(newService(gw), routeTestBase())
}
