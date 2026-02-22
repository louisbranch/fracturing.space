package icons

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	lastCall string
}

func (f *fakeService) HandleIconsPage(http.ResponseWriter, *http.Request) {
	f.lastCall = "icons_page"
}

func (f *fakeService) HandleIconsTable(http.ResponseWriter, *http.Request) {
	f.lastCall = "icons_table"
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path     string
		wantCall string
	}{
		{path: "/icons", wantCall: "icons_page"},
		{path: "/icons/table", wantCall: "icons_table"},
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
