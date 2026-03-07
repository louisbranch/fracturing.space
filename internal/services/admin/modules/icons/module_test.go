package icons

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

type fakeHandlers struct {
	lastCall string
}

func (f *fakeHandlers) HandleIconsPage(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "icons_page"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeHandlers) HandleIconsTable(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "icons_table"
	w.WriteHeader(http.StatusNoContent)
}

func TestMount(t *testing.T) {
	t.Parallel()

	svc := &fakeHandlers{}
	m, err := New(svc).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if m.Prefix != routepath.IconsPrefix {
		t.Fatalf("prefix = %q, want %q", m.Prefix, routepath.IconsPrefix)
	}

	tests := []struct {
		path     string
		wantCode int
		wantCall string
	}{
		{path: "/app/icons", wantCode: http.StatusNoContent, wantCall: "icons_page"},
		{path: "/app/icons?fragment=rows", wantCode: http.StatusNoContent, wantCall: "icons_table"},
		{path: "/app/icons/unknown", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			m.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
		})
	}
}

func TestMountNilService(t *testing.T) {
	t.Parallel()

	m, err := New(nil).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/icons?fragment=rows", nil)
	rec := httptest.NewRecorder()
	m.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
