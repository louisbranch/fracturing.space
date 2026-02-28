package notifications

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newService(staticGateway{}), routeTestDependencies()))
}

func TestRegisterRoutesNotificationsPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(staticGateway{}), routeTestDependencies()))

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

func routeTestDependencies() module.Dependencies {
	return module.Dependencies{ResolveUserID: func(*http.Request) string { return "user-1" }}
}
