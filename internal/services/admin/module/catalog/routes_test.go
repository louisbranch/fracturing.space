package catalog

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	lastCall    string
	lastSection string
	lastEntry   string
}

func (f *fakeService) HandleCatalogPage(http.ResponseWriter, *http.Request) {
	f.lastCall = "catalog_page"
}

func (f *fakeService) HandleCatalogSection(_ http.ResponseWriter, _ *http.Request, sectionID string) {
	f.lastCall = "catalog_section"
	f.lastSection = sectionID
}

func (f *fakeService) HandleCatalogSectionTable(_ http.ResponseWriter, _ *http.Request, sectionID string) {
	f.lastCall = "catalog_section_table"
	f.lastSection = sectionID
}

func (f *fakeService) HandleCatalogSectionDetail(_ http.ResponseWriter, _ *http.Request, sectionID string, entryID string) {
	f.lastCall = "catalog_section_detail"
	f.lastSection = sectionID
	f.lastEntry = entryID
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path        string
		wantCode    int
		wantCall    string
		wantSection string
		wantEntry   string
	}{
		{path: "/catalog", wantCode: http.StatusOK, wantCall: "catalog_page"},
		{path: "/catalog/daggerheart/classes", wantCode: http.StatusOK, wantCall: "catalog_section", wantSection: "classes"},
		{path: "/catalog/daggerheart/classes/table", wantCode: http.StatusOK, wantCall: "catalog_section_table", wantSection: "classes"},
		{path: "/catalog/daggerheart/classes/class-1", wantCode: http.StatusOK, wantCall: "catalog_section_detail", wantSection: "classes", wantEntry: "class-1"},
		{path: "/catalog/daggerheart/not-a-real-section", wantCode: http.StatusNotFound},
		{path: "/catalog/unknown/classes", wantCode: http.StatusNotFound},
		{path: "/catalog/daggerheart", wantCode: http.StatusNotFound},
		{path: "/catalog/daggerheart/classes/class-1/extra", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastSection = ""
			svc.lastEntry = ""

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

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

func TestHandleCatalogPathRedirectsTrailingSlash(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodGet, "/catalog/daggerheart/classes/", nil)
	rec := httptest.NewRecorder()

	HandleCatalogPath(rec, req, svc)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if location := rec.Header().Get("Location"); location != "/catalog/daggerheart/classes" {
		t.Fatalf("location = %q, want %q", location, "/catalog/daggerheart/classes")
	}
}
