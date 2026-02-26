package notifications

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web2/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMountServesNotificationsGet(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []NotificationSummary{{ID: "n1", Title: "Welcome", Read: false}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.Notifications, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	if body := rr.Body.String(); !strings.Contains(body, "web2-scaffold-page") || !strings.Contains(body, "notifications-root") {
		t.Fatalf("body = %q, want minimal scaffold notifications page", body)
	}
}

func TestMountServesNotificationsHead(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []NotificationSummary{{ID: "n1", Title: "Welcome", Read: false}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodHead, routepath.Notifications, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestModuleIDReturnsNotifications(t *testing.T) {
	t.Parallel()

	if got := New().ID(); got != "notifications" {
		t.Fatalf("ID() = %q, want %q", got, "notifications")
	}
}

func TestMountReturnsServiceUnavailableWhenGatewayNotConfigured(t *testing.T) {
	t.Parallel()

	m := New()
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.Notifications, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: default module wiring must fail closed when notifications backend is absent.
	if strings.Contains(body, "notifications-root") {
		t.Fatalf("body unexpectedly rendered notifications scaffold without backend: %q", body)
	}
}

func TestMountRejectsNotificationsNonGet(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodPatch, routepath.Notifications+"inbox", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMountServesNotificationOpenRoute(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []NotificationSummary{{ID: "n1", Title: "Welcome", Read: false}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppNotificationsOpen("n1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if body := rr.Body.String(); !strings.Contains(body, "web2-scaffold-page") || !strings.Contains(body, "notification-open") {
		t.Fatalf("body = %q, want minimal scaffold notification detail page", body)
	}
}

func TestMountMapsNotificationGatewayErrorToHTTPStatus(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: apperrors.E(apperrors.KindUnauthorized, "missing session")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.Notifications+"inbox", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMountNotificationsGRPCNotFoundRendersAppErrorPage(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: status.Error(codes.NotFound, "notification not found")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.Notifications, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: backend transport errors must never leak raw gRPC strings to user-facing pages.
	if strings.Contains(body, "rpc error:") {
		t.Fatalf("body leaked raw grpc error: %q", body)
	}
}

func TestMountNotificationsGRPCNotFoundHTMXRendersErrorFragment(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: status.Error(codes.NotFound, "notification not found")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.Notifications, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: HTMX failures must swap a fragment and not a full document.
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx error fragment without document wrapper")
	}
}

func TestMountNotificationsHTMXReturnsFragmentWithoutDocumentWrapper(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []NotificationSummary{{ID: "n1", Title: "Welcome", Read: false}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.Notifications, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "notifications-root") {
		t.Fatalf("body = %q, want notifications marker", body)
	}
	// Invariant: HTMX requests must receive partial content, never a full document envelope.
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx fragment without document wrapper")
	}
}

func TestMountNotificationsUnknownSubpathRendersSharedNotFoundPage(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{items: []NotificationSummary{{ID: "n1", Title: "Welcome", Read: false}}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.Notifications+"n1/extra", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: unknown app routes should use the shared not-found page, not net/http plain text.
	if strings.Contains(body, "404 page not found") {
		t.Fatalf("body unexpectedly rendered plain 404 text: %q", body)
	}
}

type fakeGateway struct {
	items []NotificationSummary
	err   error
}

func (f fakeGateway) ListNotifications(context.Context) ([]NotificationSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.items) == 0 {
		return nil, errors.New("no notifications")
	}
	return f.items, nil
}

func (f fakeGateway) OpenNotification(context.Context, string) (NotificationSummary, error) {
	if f.err != nil {
		return NotificationSummary{}, f.err
	}
	if len(f.items) == 0 {
		return NotificationSummary{}, errors.New("no notification")
	}
	return f.items[0], nil
}
