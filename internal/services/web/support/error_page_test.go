package support

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"

	"github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func TestRenderErrorPage_LocalizesAndRendersPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/app?foo=bar", nil)
	writer := httptest.NewRecorder()
	page := templates.PageContext{}

	var gotContentType bool
	var gotLocalizedTitle, gotLocalizedMessage string
	var gotComposedTitle string
	var gotPageTitle string
	var gotPageComponent templ.Component
	var gotWriteErr error

	RenderErrorPage(
		writer,
		req,
		http.StatusUnauthorized,
		"Access denied",
		"failed to load profile",
		page,
		ErrorPageRenderer{
			WriteContentType: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				gotContentType = true
			},
			WritePage: func(w http.ResponseWriter, _ *http.Request, component templ.Component, title string) error {
				gotPageComponent = component
				gotPageTitle = title
				_, err := w.Write([]byte("rendered"))
				gotWriteErr = err
				return err
			},
			ComposeTitle: func(_ templates.PageContext, title string, _ ...any) string {
				gotComposedTitle = "COMPOSE:" + title
				return gotComposedTitle
			},
			LocalizeText: func(_ templates.Localizer, raw string, _ map[string]string) string {
				if raw == "Access denied" {
					gotLocalizedTitle = "Acesso negado"
					return gotLocalizedTitle
				}
				if raw == "failed to load profile" {
					gotLocalizedMessage = "Falha ao carregar perfil"
					return gotLocalizedMessage
				}
				return raw
			},
			LocalizeHTTP: func(http.ResponseWriter, *http.Request, int, string, ...any) {},
		},
	)

	if !gotContentType {
		t.Fatalf("expected WriteContentType to be invoked")
	}
	if writer.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", writer.Result().StatusCode, http.StatusUnauthorized)
	}
	if gotLocalizedTitle == "" {
		t.Fatalf("expected localized title to be produced")
	}
	if gotLocalizedMessage == "" {
		t.Fatalf("expected localized message to be produced")
	}
	if gotPageTitle != gotComposedTitle {
		t.Fatalf("write page title = %q, want %q", gotPageTitle, gotComposedTitle)
	}
	if gotPageComponent == nil {
		t.Fatalf("expected non-nil error page component")
	}
	if gotWriteErr != nil {
		t.Fatalf("unexpected write error while rendering page: %v", gotWriteErr)
	}
}

func TestRenderErrorPage_LocalizeHTTPOnWritePageError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	writer := httptest.NewRecorder()

	var calledLocalizeHTTP bool
	RenderErrorPage(
		writer,
		req,
		http.StatusBadRequest,
		"Campaign unavailable",
		"failed to load campaign",
		templates.PageContext{},
		ErrorPageRenderer{
			WriteContentType: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			},
			WritePage: func(http.ResponseWriter, *http.Request, templ.Component, string) error {
				return errors.New("template write failure")
			},
			LocalizeText: func(_ templates.Localizer, raw string, _ map[string]string) string {
				return raw
			},
			LocalizeHTTP: func(_ http.ResponseWriter, _ *http.Request, status int, key string, _ ...any) {
				calledLocalizeHTTP = status == http.StatusInternalServerError && key == "error.http.web_handler_unavailable"
			},
		},
	)

	if !calledLocalizeHTTP {
		t.Fatalf("expected fallback LocalizeHTTP call with web handler unavailable key")
	}
}
