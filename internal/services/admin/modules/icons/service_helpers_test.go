package icons

import (
	"net/http"
	"net/http/httptest"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformicons "github.com/louisbranch/fracturing.space/internal/platform/icons"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
)

func TestBuildIconRows(t *testing.T) {
	rows := buildIconRows([]platformicons.Definition{
		{ID: commonv1.IconId_ICON_ID_GENERIC, Name: "One", Description: "first"},
		{ID: commonv1.IconId_ICON_ID_CHAT, Name: "Two", Description: "second"},
	})
	if len(rows) != 2 {
		t.Fatalf("buildIconRows() len = %d", len(rows))
	}
	if rows[0].ID != commonv1.IconId_ICON_ID_GENERIC || rows[0].Name != "One" {
		t.Fatalf("buildIconRows() first row = %#v", rows[0])
	}
	if rows[0].LucideName == "" {
		t.Fatalf("buildIconRows() missing lucide name: %#v", rows[0])
	}
}

func TestIconsServiceHandlers(t *testing.T) {
	svc := service{base: modulehandler.NewBase(nil)}

	rec := httptest.NewRecorder()
	svc.HandleIconsPage(rec, httptest.NewRequest(http.MethodGet, "/app/icons", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleIconsPage() status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.HandleIconsTable(rec, httptest.NewRequest(http.MethodGet, "/app/icons?fragment=rows", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleIconsTable() status = %d", rec.Code)
	}
}
