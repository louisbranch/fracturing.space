package pagerender

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
)

func TestWriteModulePageRendersHTMXFragmentWithStatus(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	err := WriteModulePage(rr, req, module.Dependencies{}, ModulePage{
		Title:      "Settings",
		StatusCode: http.StatusCreated,
		Fragment:   textComponent(`<section id="fragment-root">ok</section>`),
	})
	if err != nil {
		t.Fatalf("WriteModulePage() error = %v", err)
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="fragment-root"`) {
		t.Fatalf("body missing fragment marker: %q", body)
	}
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx fragment without full document wrapper")
	}
}

func TestWriteModulePageRendersFullPageWithAppShell(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	rr := httptest.NewRecorder()

	err := WriteModulePage(rr, req, module.Dependencies{}, ModulePage{
		Title:      "Settings",
		StatusCode: http.StatusAccepted,
		Fragment:   textComponent(`<section id="fragment-root">ok</section>`),
	})
	if err != nil {
		t.Fatalf("WriteModulePage() error = %v", err)
	}
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	body := rr.Body.String()
	for _, marker := range []string{`id="main"`, `id="fragment-root"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing marker %q: %q", marker, body)
		}
	}
}

type textComponent string

func (c textComponent) Render(_ context.Context, w io.Writer) error {
	_, err := io.WriteString(w, string(c))
	return err
}
