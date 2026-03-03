package icons

import "net/http"

// Service defines icon handlers consumed by this module's routes.
type Service interface {
	HandleIconsPage(w http.ResponseWriter, r *http.Request)
	HandleIconsTable(w http.ResponseWriter, r *http.Request)
}
