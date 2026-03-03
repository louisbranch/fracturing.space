package catalog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
)

func TestCatalogServiceHandlersWithNilClient(t *testing.T) {
	svcIface := NewService(modulehandler.NewBase(nil))
	svc, ok := svcIface.(*service)
	if !ok {
		t.Fatalf("NewService() type = %T, want *service", svcIface)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/catalog", nil)
	rec := httptest.NewRecorder()
	svc.HandleCatalogPage(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleCatalogPage(GET) status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/app/catalog", nil)
	rec = httptest.NewRecorder()
	svc.HandleCatalogPage(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("HandleCatalogPage(POST) status = %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("HandleCatalogPage(POST) Allow = %q", allow)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/catalog/daggerheart/classes", nil)
	rec = httptest.NewRecorder()
	svc.HandleCatalogSection(rec, req, templates.CatalogSectionClasses)
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleCatalogSection(non-HTMX) status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/catalog/daggerheart/classes", nil)
	req.Header.Set("HX-Request", "true")
	rec = httptest.NewRecorder()
	svc.HandleCatalogSection(rec, req, templates.CatalogSectionClasses)
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleCatalogSection(HTMX) status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/catalog/daggerheart/classes?fragment=rows", nil)
	rec = httptest.NewRecorder()
	svc.HandleCatalogSectionTable(rec, req, templates.CatalogSectionClasses)
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleCatalogSectionTable(nil content client) status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/catalog/daggerheart/classes/class-1", nil)
	rec = httptest.NewRecorder()
	svc.HandleCatalogSectionDetail(rec, req, templates.CatalogSectionClasses, "class-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleCatalogSectionDetail(nil content client) status = %d", rec.Code)
	}
}

func TestCatalogServiceLocaleFromTag(t *testing.T) {
	if got := localeFromTag("pt-BR"); got != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("localeFromTag(pt-BR) = %v, want %v", got, commonv1.Locale_LOCALE_PT_BR)
	}
	if got := localeFromTag("not-a-locale"); got != platformi18n.DefaultLocale() {
		t.Fatalf("localeFromTag(invalid) = %v, want %v", got, platformi18n.DefaultLocale())
	}
}
