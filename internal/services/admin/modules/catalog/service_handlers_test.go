package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testUnavailableConn implements grpc.ClientConnInterface and returns
// codes.Unavailable for every RPC, simulating a disconnected backend.
type testUnavailableConn struct{}

func (testUnavailableConn) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	return status.Error(codes.Unavailable, "test: service not connected")
}

func (testUnavailableConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Error(codes.Unavailable, "test: service not connected")
}

func TestCatalogServiceHandlersWithUnavailableClient(t *testing.T) {
	var conn testUnavailableConn
	svcIface := NewHandlers(modulehandler.NewBase(), daggerheartv1.NewDaggerheartContentServiceClient(conn))
	svc, ok := svcIface.(*handlers)
	if !ok {
		t.Fatalf("NewHandlers() type = %T, want *handlers", svcIface)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/catalog", nil)
	rec := httptest.NewRecorder()
	svc.HandleCatalogPage(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleCatalogPage(GET) status = %d", rec.Code)
	}

	// Method enforcement is handled by the mux (routes registered with method prefix).

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
