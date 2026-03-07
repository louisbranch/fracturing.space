package catalog

import "net/http"

// Handlers defines catalog handler methods consumed by this module's routes.
type Handlers interface {
	HandleCatalogPage(w http.ResponseWriter, r *http.Request)
	HandleCatalogSection(w http.ResponseWriter, r *http.Request, sectionID string)
	HandleCatalogSectionTable(w http.ResponseWriter, r *http.Request, sectionID string)
	HandleCatalogSectionDetail(w http.ResponseWriter, r *http.Request, sectionID string, entryID string)
}
