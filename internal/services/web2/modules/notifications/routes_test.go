package notifications

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(newService(staticGateway{}), module.Dependencies{}))
}

func TestRegisterRoutesNotificationsPathAndMethodContracts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(newService(staticGateway{}), module.Dependencies{}))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "notifications root", method: http.MethodGet, path: routepath.AppNotifications, wantStatus: http.StatusOK},
		{name: "notifications slash root", method: http.MethodGet, path: routepath.Notifications, wantStatus: http.StatusOK},
		{name: "notifications head", method: http.MethodHead, path: routepath.AppNotifications, wantStatus: http.StatusOK},
		{name: "notifications post rejected", method: http.MethodPost, path: routepath.AppNotifications, wantStatus: http.StatusMethodNotAllowed},
		{name: "notification open", method: http.MethodGet, path: routepath.AppNotificationsOpen("n1"), wantStatus: http.StatusOK},
		{name: "notification unknown subpath", method: http.MethodGet, path: routepath.AppNotificationsOpen("n1") + "/other", wantStatus: http.StatusNotFound},
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
		})
	}
}
