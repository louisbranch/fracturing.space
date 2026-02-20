package route

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectTrailingSlash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		wantOK   bool
		wantCode int
		wantLoc  string
	}{
		{
			name:     "no trailing slash",
			path:     "/campaigns",
			wantOK:   false,
			wantCode: 200,
		},
		{
			name:     "trailing slash",
			path:     "/campaigns/",
			wantOK:   true,
			wantCode: http.StatusMovedPermanently,
			wantLoc:  "/campaigns",
		},
		{
			name:     "campaign detail trailing slash",
			path:     "/campaigns/camp-1/",
			wantOK:   true,
			wantCode: http.StatusMovedPermanently,
			wantLoc:  "/campaigns/camp-1",
		},
		{
			name:     "root path",
			path:     "/",
			wantOK:   false,
			wantCode: 200,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			got := RedirectTrailingSlash(rec, req)
			if got != tc.wantOK {
				t.Fatalf("RedirectTrailingSlash = %v, want %v", got, tc.wantOK)
			}
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if got {
				if loc := rec.Header().Get("Location"); loc != tc.wantLoc {
					t.Fatalf("location = %q, want %q", loc, tc.wantLoc)
				}
				return
			}
		})
	}
}
