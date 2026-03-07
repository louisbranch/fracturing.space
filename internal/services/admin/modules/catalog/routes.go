package catalog

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
)

func newRoutes(h Handlers) *http.ServeMux {
	mux := http.NewServeMux()
	if h == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppCatalog, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.CatalogPrefix+"{$}", http.NotFound)
		return mux
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCatalog, h.HandleCatalogPage)
	mux.HandleFunc(http.MethodGet+" "+routepath.CatalogPrefix+"{$}", h.HandleCatalogPage)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCatalogSectionPattern, func(w http.ResponseWriter, r *http.Request) {
		systemID := strings.TrimSpace(r.PathValue("systemID"))
		sectionID := strings.TrimSpace(r.PathValue("sectionID"))
		if !isSupportedSection(systemID, sectionID) {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			h.HandleCatalogSectionTable(w, r, sectionID)
			return
		}
		h.HandleCatalogSection(w, r, sectionID)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCatalogEntryPattern, func(w http.ResponseWriter, r *http.Request) {
		systemID := strings.TrimSpace(r.PathValue("systemID"))
		sectionID := strings.TrimSpace(r.PathValue("sectionID"))
		entryID := strings.TrimSpace(r.PathValue("entryID"))
		if !isSupportedSection(systemID, sectionID) || entryID == "" || isLegacyTableSegment(entryID) {
			http.NotFound(w, r)
			return
		}
		h.HandleCatalogSectionDetail(w, r, sectionID, entryID)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.CatalogPrefix+"{systemID}/{sectionID}/{rest...}", http.NotFound)
	return mux
}

func isSupportedSection(systemID string, sectionID string) bool {
	if !strings.EqualFold(strings.TrimSpace(systemID), DaggerheartSystemID) {
		return false
	}
	return templates.IsDaggerheartCatalogSection(strings.TrimSpace(sectionID))
}

func wantsRowsFragment(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get(routepath.FragmentQueryKey)), routepath.FragmentRows)
}

func isLegacyTableSegment(segment string) bool {
	switch strings.ToLower(strings.TrimSpace(segment)) {
	case "table", "_rows":
		return true
	default:
		return false
	}
}
