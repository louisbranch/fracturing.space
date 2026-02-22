package catalog

import (
	"net/http"
	"strings"

	sharedpath "github.com/louisbranch/fracturing.space/internal/services/admin/module/sharedpath"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
)

const (
	// DaggerheartSystemID is the currently supported system slug in admin catalog routes.
	DaggerheartSystemID = "daggerheart"
)

// Service defines catalog route handlers consumed by this route module.
type Service interface {
	HandleCatalogPage(w http.ResponseWriter, r *http.Request)
	HandleCatalogSection(w http.ResponseWriter, r *http.Request, sectionID string)
	HandleCatalogSectionTable(w http.ResponseWriter, r *http.Request, sectionID string)
	HandleCatalogSectionDetail(w http.ResponseWriter, r *http.Request, sectionID string, entryID string)
}

// RegisterRoutes wires catalog routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Catalog, service.HandleCatalogPage)
	mux.HandleFunc(routepath.CatalogPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleCatalogPath(w, r, service)
	})
}

// HandleCatalogPath parses catalog subroutes and dispatches to service handlers.
func HandleCatalogPath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, routepath.CatalogPrefix)
	parts := sharedpath.SplitPathParts(path)

	if len(parts) == 2 && parts[0] == DaggerheartSystemID && templates.IsDaggerheartCatalogSection(parts[1]) {
		service.HandleCatalogSection(w, r, parts[1])
		return
	}
	if len(parts) == 3 && parts[0] == DaggerheartSystemID && parts[2] == "table" && templates.IsDaggerheartCatalogSection(parts[1]) {
		service.HandleCatalogSectionTable(w, r, parts[1])
		return
	}
	if len(parts) == 3 && parts[0] == DaggerheartSystemID && templates.IsDaggerheartCatalogSection(parts[1]) {
		service.HandleCatalogSectionDetail(w, r, parts[1], parts[2])
		return
	}
	http.NotFound(w, r)
}
