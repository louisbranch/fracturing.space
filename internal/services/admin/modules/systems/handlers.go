package systems

import "net/http"

// Service defines systems handlers consumed by this module's routes.
type Service interface {
	HandleSystemsPage(w http.ResponseWriter, r *http.Request)
	HandleSystemsTable(w http.ResponseWriter, r *http.Request)
	HandleSystemDetail(w http.ResponseWriter, r *http.Request, systemID string)
}
