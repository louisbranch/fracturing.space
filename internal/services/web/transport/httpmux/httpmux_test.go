package httpmux

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestMountAppAndPublicRoutesUsesCanonicalAppPrefix(t *testing.T) {
	t.Parallel()

	rootMux := http.NewServeMux()
	appMux := http.NewServeMux()
	publicMux := http.NewServeMux()

	appMux.HandleFunc("/app/campaigns", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("app-campaigns"))
	})
	publicMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("public-root"))
	})

	MountAppAndPublicRoutes(rootMux, appMux, publicMux)

	appReq := httptest.NewRequest(http.MethodGet, "/app/campaigns", nil)
	appRec := httptest.NewRecorder()
	rootMux.ServeHTTP(appRec, appReq)
	if appRec.Code != http.StatusOK {
		t.Fatalf("/app/campaigns status = %d, want %d", appRec.Code, http.StatusOK)
	}
	if body := appRec.Body.String(); body != "app-campaigns" {
		t.Fatalf("/app/campaigns body = %q, want %q", body, "app-campaigns")
	}

	legacyReq := httptest.NewRequest(http.MethodGet, "/campaigns", nil)
	legacyRec := httptest.NewRecorder()
	rootMux.ServeHTTP(legacyRec, legacyReq)
	if legacyRec.Code != http.StatusNotFound {
		t.Fatalf("/campaigns status = %d, want %d", legacyRec.Code, http.StatusNotFound)
	}
}

func TestMountStaticServesStaticPrefix(t *testing.T) {
	t.Parallel()

	rootMux := http.NewServeMux()
	staticFS := fstest.MapFS{
		"app.js": &fstest.MapFile{Data: []byte("console.log('ok');")},
	}
	MountStatic(rootMux, staticFS, nil)

	req := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	rec := httptest.NewRecorder()
	rootMux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMountStaticNoopOnNilInputs(t *testing.T) {
	t.Parallel()

	rootMux := http.NewServeMux()
	MountStatic(nil, fstest.MapFS{}, nil)
	MountStatic(rootMux, fs.FS(nil), nil)
}
