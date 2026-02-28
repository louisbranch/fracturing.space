package notifications

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMountServesNotificationsGet(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{listItems: []NotificationSummary{{ID: "n1", MessageType: "auth.onboarding.welcome", Read: false}}})
	mount, err := m.Mount(notificationsTestDependencies())
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
	body := rr.Body.String()
	if !strings.Contains(body, "notifications-root") {
		t.Fatalf("body missing notifications root marker: %q", body)
	}
	if !strings.Contains(body, "Welcome to Fracturing Space") {
		t.Fatalf("body missing rendered notification title: %q", body)
	}
}

func TestMountServesNotificationsHead(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{listItems: []NotificationSummary{{ID: "n1", MessageType: "auth.onboarding.welcome", Read: false}}})
	mount, err := m.Mount(notificationsTestDependencies())
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
	mount, err := m.Mount(notificationsTestDependencies())
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
		t.Fatalf("body unexpectedly rendered notifications surface without backend: %q", body)
	}
}

func TestMountRejectsNotificationsNonGet(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{})
	mount, _ := m.Mount(notificationsTestDependencies())
	req := httptest.NewRequest(http.MethodPatch, routepath.Notifications+"inbox", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMountServesNotificationDetailRoute(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{getItem: NotificationSummary{ID: "n1", MessageType: "auth.onboarding.welcome", Read: true}})
	mount, err := m.Mount(notificationsTestDependencies())
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppNotification("n1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "notification-open") {
		t.Fatalf("body missing detail container marker: %q", body)
	}
	if !strings.Contains(body, "Welcome to Fracturing Space") {
		t.Fatalf("body missing rendered detail title: %q", body)
	}
}

func TestMountNotificationOpenRouteRedirectsToDetail(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{openItem: NotificationSummary{ID: "n1", MessageType: "auth.onboarding.welcome", Read: true}})
	mount, err := m.Mount(notificationsTestDependencies())
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppNotificationOpen("n1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppNotification("n1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppNotification("n1"))
	}
}

func TestMountNotificationOpenRouteHTMXRedirects(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{openItem: NotificationSummary{ID: "n1", MessageType: "auth.onboarding.welcome", Read: true}})
	mount, err := m.Mount(notificationsTestDependencies())
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, routepath.AppNotificationOpen("n1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppNotification("n1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppNotification("n1"))
	}
}

func TestMountMapsNotificationGatewayErrorToHTTPStatus(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{listErr: apperrors.E(apperrors.KindUnauthorized, "missing session")})
	mount, err := m.Mount(notificationsTestDependencies())
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

	m := NewWithGateway(fakeGateway{listErr: status.Error(codes.NotFound, "notification not found")})
	mount, err := m.Mount(notificationsTestDependencies())
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

func TestMountNotificationsHTMXReturnsFragmentWithoutDocumentWrapper(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{listItems: []NotificationSummary{{ID: "n1", MessageType: "auth.onboarding.welcome", Read: false}}})
	mount, err := m.Mount(notificationsTestDependencies())
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

	m := NewWithGateway(fakeGateway{})
	mount, err := m.Mount(notificationsTestDependencies())
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

func notificationsTestDependencies() module.Dependencies {
	return module.Dependencies{ResolveUserID: func(*http.Request) string { return "user-1" }}
}

type fakeGateway struct {
	listItems []NotificationSummary
	listErr   error
	getItem   NotificationSummary
	getErr    error
	openItem  NotificationSummary
	openErr   error
}

func (f fakeGateway) ListNotifications(context.Context, string) ([]NotificationSummary, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listItems == nil {
		return []NotificationSummary{{ID: "n1", MessageType: "auth.onboarding.welcome", Read: false}}, nil
	}
	return f.listItems, nil
}

func (f fakeGateway) GetNotification(context.Context, string, string) (NotificationSummary, error) {
	if f.getErr != nil {
		return NotificationSummary{}, f.getErr
	}
	if f.getItem != (NotificationSummary{}) {
		return f.getItem, nil
	}
	if len(f.listItems) > 0 {
		return f.listItems[0], nil
	}
	return NotificationSummary{ID: "n1", MessageType: "auth.onboarding.welcome", Read: false}, nil
}

func (f fakeGateway) OpenNotification(context.Context, string, string) (NotificationSummary, error) {
	if f.openErr != nil {
		return NotificationSummary{}, f.openErr
	}
	if f.openItem != (NotificationSummary{}) {
		return f.openItem, nil
	}
	return NotificationSummary{ID: "n1", MessageType: "auth.onboarding.welcome", Read: true}, nil
}

var _ NotificationGateway = fakeGateway{}
