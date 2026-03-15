package notifications

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/platform/icons"
	"github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMountServesNotificationsGet(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: fakeGateway{listItems: []notificationsapp.NotificationSummary{testNotificationSummary("n1", false)}}, Base: notificationsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppNotifications, nil)
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
	if !strings.Contains(body, `data-app-side-menu-item="/app/notifications/n1"`) {
		t.Fatalf("body missing notifications side-menu item: %q", body)
	}
	if !strings.Contains(body, `href="#`+icons.LucideSymbolID("mail")+`"`) {
		t.Fatalf("body missing message icon: %q", body)
	}
	if strings.Contains(body, `data-notification-unread="true"`) {
		t.Fatalf("body should no longer render bespoke inbox rows: %q", body)
	}
}

func TestMountServesNotificationsHead(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: fakeGateway{listItems: []notificationsapp.NotificationSummary{testNotificationSummary("n1", false)}}, Base: notificationsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodHead, routepath.AppNotifications, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestModuleIDReturnsNotifications(t *testing.T) {
	t.Parallel()

	if got := New(Config{}).ID(); got != "notifications" {
		t.Fatalf("ID() = %q, want %q", got, "notifications")
	}
}

func TestModuleHealthyReflectsGatewayState(t *testing.T) {
	t.Parallel()

	if New(Config{}).Healthy() {
		t.Fatalf("New().Healthy() = true, want false for degraded module")
	}
	if !New(Config{Gateway: fakeGateway{}, Base: notificationsTestBase()}).Healthy() {
		t.Fatalf("New(Config{...}).Healthy() = false, want true")
	}
}

func TestMountReturnsServiceUnavailableWhenGatewayNotConfigured(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: nil, Base: notificationsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppNotifications, nil)
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

	m := New(Config{Gateway: fakeGateway{}, Base: notificationsTestBase()})
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPatch, routepath.Notifications+"inbox", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMountServesNotificationDetailRoute(t *testing.T) {
	t.Parallel()

	listItem := testNotificationSummary("n1", false)
	detailItem := notificationsapp.NotificationSummary{
		ID:          "n1",
		MessageType: "system.message.v1",
		PayloadJSON: mustNotificationPayloadJSON(notificationpayload.InAppPayload{
			Title: platformi18n.NewCopyRef("notification.campaign_invite.updated.title"),
			Body:  platformi18n.NewCopyRef("notification.campaign_invite.updated.body"),
		}),
		Read: false,
	}
	m := New(Config{Gateway: fakeGateway{
		listItems: []notificationsapp.NotificationSummary{listItem},
		getItem:   detailItem,
		openItem: notificationsapp.NotificationSummary{
			ID:          "n1",
			MessageType: "system.message.v1",
			PayloadJSON: mustNotificationPayloadJSON(notificationpayload.InAppPayload{
				Title: platformi18n.NewCopyRef("notification.campaign_invite.accepted.title"),
				Body:  platformi18n.NewCopyRef("notification.campaign_invite.accepted.body"),
			}),
			Read: true,
		},
	}, Base: notificationsTestBase()})
	mount, err := m.Mount()
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
	if !strings.Contains(body, "Invitation update") {
		t.Fatalf("body missing rendered detail title: %q", body)
	}
	// Invariant: GET detail remains read-only; only POST /open performs acknowledgment.
	if strings.Contains(body, "Invitation accepted") {
		t.Fatalf("body unexpectedly rendered open-route content on GET detail: %q", body)
	}
	if !strings.Contains(body, `class="menu-active" href="/app/notifications/n1"`) {
		t.Fatalf("body missing active side-menu item: %q", body)
	}
	if !strings.Contains(body, `href="#`+icons.LucideSymbolID("mail")+`"`) {
		t.Fatalf("body missing message icon: %q", body)
	}
}

func TestMountServesInviteNotificationDetailActions(t *testing.T) {
	t.Parallel()

	payloadJSON, err := json.Marshal(notificationpayload.InAppPayload{
		Title: platformi18n.NewCopyRef("notification.campaign_invite.created.title"),
		Body:  platformi18n.NewCopyRef("notification.campaign_invite.created.body_summary", "gm", "Skyfall"),
		Facts: []notificationpayload.PayloadFact{
			{Label: platformi18n.NewCopyRef("notification.fact.campaign"), Value: "Skyfall"},
			{Label: platformi18n.NewCopyRef("notification.fact.seat"), Value: "Scout"},
			{Label: platformi18n.NewCopyRef("notification.fact.invited_by"), Value: "@gm"},
		},
		Actions: []notificationpayload.PayloadAction{
			notificationpayload.ViewInvitationAction("inv-1"),
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	inviteItem := notificationsapp.NotificationSummary{
		ID:          "n1",
		MessageType: "campaign.invite.created.v1",
		PayloadJSON: string(payloadJSON),
		Read:        true,
	}
	m := New(Config{Gateway: fakeGateway{
		listItems: []notificationsapp.NotificationSummary{inviteItem},
		getItem:   inviteItem,
		openItem:  inviteItem,
	}, Base: notificationsTestBase()})
	mount, err := m.Mount()
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
	for _, needle := range []string{
		"Campaign invitation",
		"@gm invited you to join Skyfall.",
		"Skyfall",
		"Scout",
		"@gm",
		`href="/invite/inv-1"`,
		`View invitation`,
		`href="#` + icons.LucideSymbolID("user-plus") + `"`,
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("body missing %q: %q", needle, body)
		}
	}
}

func TestMountNotificationOpenRouteRedirectsToDetail(t *testing.T) {
	t.Parallel()

	m := New(Config{Gateway: fakeGateway{openItem: testNotificationSummary("n1", true)}, Base: notificationsTestBase()})
	mount, err := m.Mount()
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

func TestMountInviteNotificationOpenRouteStaysInNotifications(t *testing.T) {
	t.Parallel()

	payloadJSON, err := json.Marshal(notificationpayload.InAppPayload{
		Title: platformi18n.NewCopyRef("notification.campaign_invite.created.title"),
		Body:  platformi18n.NewCopyRef("notification.campaign_invite.created.body_summary", "gm", "Skyfall"),
		Actions: []notificationpayload.PayloadAction{
			notificationpayload.ViewInvitationAction("inv-1"),
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	m := New(Config{Gateway: fakeGateway{openItem: notificationsapp.NotificationSummary{
		ID:          "n1",
		MessageType: "campaign.invite.created.v1",
		PayloadJSON: string(payloadJSON),
		Read:        true,
	}}, Base: notificationsTestBase()})
	mount, err := m.Mount()
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

	m := New(Config{Gateway: fakeGateway{openItem: testNotificationSummary("n1", true)}, Base: notificationsTestBase()})
	mount, err := m.Mount()
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

	m := New(Config{Gateway: fakeGateway{listErr: apperrors.E(apperrors.KindUnauthorized, "missing session")}, Base: notificationsTestBase()})
	mount, err := m.Mount()
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

	m := New(Config{Gateway: fakeGateway{listErr: status.Error(codes.NotFound, "notification not found")}, Base: notificationsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppNotifications, nil)
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

	m := New(Config{Gateway: fakeGateway{listItems: []notificationsapp.NotificationSummary{testNotificationSummary("n1", false)}}, Base: notificationsTestBase()})
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.AppNotifications, nil)
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

	m := New(Config{Gateway: fakeGateway{}, Base: notificationsTestBase()})
	mount, err := m.Mount()
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

func notificationsTestBase() modulehandler.Base {
	return modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil)
}
