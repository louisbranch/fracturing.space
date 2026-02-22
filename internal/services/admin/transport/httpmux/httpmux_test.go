package httpmux

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestMountStaticServesAssets(t *testing.T) {
	t.Parallel()

	rootMux := http.NewServeMux()
	staticFS := fstest.MapFS{
		"theme.css": &fstest.MapFile{Data: []byte("body{}")},
	}
	MountStatic(rootMux, staticFS, nil)

	req := httptest.NewRequest(http.MethodGet, "/static/theme.css", nil)
	rec := httptest.NewRecorder()
	rootMux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMountAdminRoutesMountsRoot(t *testing.T) {
	t.Parallel()

	rootMux := http.NewServeMux()
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/users", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("users"))
	})

	MountAdminRoutes(rootMux, adminMux)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	rootMux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != "users" {
		t.Fatalf("body = %q, want %q", body, "users")
	}
}

func TestMountNoopsOnNilInputs(t *testing.T) {
	t.Parallel()

	rootMux := http.NewServeMux()
	MountStatic(nil, fstest.MapFS{}, nil)
	MountStatic(rootMux, fs.FS(nil), nil)
	MountAdminRoutes(nil, http.NewServeMux())
	MountAdminRoutes(rootMux, nil)
}
