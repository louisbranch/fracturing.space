package notifications

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeNotificationsService struct {
	lastCall string
}

func (f *fakeNotificationsService) HandleNotifications(http.ResponseWriter, *http.Request) {
	f.lastCall = "notifications"
}

func (f *fakeNotificationsService) HandleNotificationsSubroutes(http.ResponseWriter, *http.Request) {
	f.lastCall = "notifications_subroutes"
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeNotificationsService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path     string
		wantCall string
	}{
		{path: "/app/notifications", wantCall: "notifications"},
		{path: "/app/notifications/item-1", wantCall: "notifications_subroutes"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			svc.lastCall = ""

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
		})
	}
}
