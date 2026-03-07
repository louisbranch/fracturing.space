package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func TestRoutesRedirectRootToDashboard(t *testing.T) {
	t.Parallel()

	handler := NewServiceHandler(nil, "", nil, nil).routes()
	req := httptest.NewRequest(http.MethodGet, routepath.Root, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if location := rec.Header().Get("Location"); location != routepath.AppDashboard {
		t.Fatalf("location = %q, want %q", location, routepath.AppDashboard)
	}
}
