package catalog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

type fakeService struct {
	lastCall    string
	lastSection string
	lastEntry   string
}

func (f *fakeService) HandleCatalogPage(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "catalog_page"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleCatalogSection(w http.ResponseWriter, _ *http.Request, sectionID string) {
	f.lastCall = "catalog_section"
	f.lastSection = sectionID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleCatalogSectionTable(w http.ResponseWriter, _ *http.Request, sectionID string) {
	f.lastCall = "catalog_section_table"
	f.lastSection = sectionID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleCatalogSectionDetail(w http.ResponseWriter, _ *http.Request, sectionID string, entryID string) {
	f.lastCall = "catalog_section_detail"
	f.lastSection = sectionID
	f.lastEntry = entryID
	w.WriteHeader(http.StatusNoContent)
}

func TestMount(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	m, err := New(svc).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if m.Prefix != routepath.CatalogPrefix {
		t.Fatalf("prefix = %q, want %q", m.Prefix, routepath.CatalogPrefix)
	}

	tests := []struct {
		path        string
		wantCode    int
		wantCall    string
		wantSection string
		wantEntry   string
	}{
		{path: "/catalog", wantCode: http.StatusNoContent, wantCall: "catalog_page"},
		{path: "/catalog/daggerheart/classes", wantCode: http.StatusNoContent, wantCall: "catalog_section", wantSection: "classes"},
		{path: "/catalog/daggerheart/classes/_rows", wantCode: http.StatusNoContent, wantCall: "catalog_section_table", wantSection: "classes"},
		{path: "/catalog/daggerheart/classes/class-1", wantCode: http.StatusNoContent, wantCall: "catalog_section_detail", wantSection: "classes", wantEntry: "class-1"},
		{path: "/catalog/unknown/classes/_rows", wantCode: http.StatusNotFound},
		{path: "/catalog/daggerheart/classes/table", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastSection = ""
			svc.lastEntry = ""
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			m.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
			if svc.lastSection != tc.wantSection {
				t.Fatalf("lastSection = %q, want %q", svc.lastSection, tc.wantSection)
			}
			if svc.lastEntry != tc.wantEntry {
				t.Fatalf("lastEntry = %q, want %q", svc.lastEntry, tc.wantEntry)
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

	req := httptest.NewRequest(http.MethodGet, "/catalog/daggerheart/classes/_rows", nil)
	rec := httptest.NewRecorder()
	m.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
