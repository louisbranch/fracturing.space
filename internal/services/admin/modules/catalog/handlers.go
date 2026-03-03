package catalog

import "net/http"

// Service defines catalog handlers consumed by this module's routes.
type Service interface {
	HandleCatalogPage(w http.ResponseWriter, r *http.Request)
	HandleCatalogSection(w http.ResponseWriter, r *http.Request, sectionID string)
	HandleCatalogSectionTable(w http.ResponseWriter, r *http.Request, sectionID string)
	HandleCatalogSectionDetail(w http.ResponseWriter, r *http.Request, sectionID string, entryID string)
}
