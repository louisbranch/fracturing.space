package notifications

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newService(staticGateway{}), routeTestBase()))
}

func TestRegisterRoutesNotificationsPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(staticGateway{}), routeTestBase()))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{name: "notifications root", method: http.MethodGet, path: routepath.AppNotifications, wantStatus: http.StatusOK},
		{name: "notifications slash root", method: http.MethodGet, path: routepath.Notifications, wantStatus: http.StatusOK},
		{name: "notifications head", method: http.MethodHead, path: routepath.AppNotifications, wantStatus: http.StatusOK},
		{name: "notifications post rejected", method: http.MethodPost, path: routepath.AppNotifications, wantStatus: http.StatusMethodNotAllowed},
		{name: "notification detail", method: http.MethodGet, path: routepath.AppNotification("notification-1"), wantStatus: http.StatusOK},
		{name: "notification open post", method: http.MethodPost, path: routepath.AppNotificationOpen("notification-1"), wantStatus: http.StatusFound},
		{name: "notification open get rejected", method: http.MethodGet, path: routepath.AppNotificationOpen("notification-1"), wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "notification unknown subpath", method: http.MethodGet, path: routepath.AppNotification("notification-1") + "/other", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantAllow != "" {
				if got := rr.Header().Get("Allow"); got != tc.wantAllow {
					t.Fatalf("Allow = %q, want %q", got, tc.wantAllow)
				}
			}
		})
	}
}

func routeTestBase() modulehandler.Base {
	return modulehandler.NewBase(func(*http.Request) string { return "user-1" }, nil, nil)
}

// staticGateway returns canned notification data for route-level tests.
type staticGateway struct{}

var staticGatewayFixedTime = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func (staticGateway) ListNotifications(context.Context, string) ([]NotificationSummary, error) {
	return []NotificationSummary{{
		ID:          "notification-1",
		MessageType: "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":"passkey"}`,
		Source:      "system",
		Read:        false,
		CreatedAt:   staticGatewayFixedTime,
		UpdatedAt:   staticGatewayFixedTime,
	}}, nil
}

func (g staticGateway) GetNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	items, err := g.ListNotifications(ctx, userID)
	if err != nil {
		return NotificationSummary{}, err
	}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == strings.TrimSpace(notificationID) {
			return item, nil
		}
	}
	return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
}

func (g staticGateway) OpenNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	item, err := g.GetNotification(ctx, userID, notificationID)
	if err != nil {
		return NotificationSummary{}, err
	}
	readAt := staticGatewayFixedTime
	item.Read = true
	item.ReadAt = &readAt
	item.UpdatedAt = staticGatewayFixedTime
	return item, nil
}
