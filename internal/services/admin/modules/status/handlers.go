package status

import "net/http"

// Service defines status handlers consumed by this module's routes.
type Service interface {
	HandleStatusPage(w http.ResponseWriter, r *http.Request)
	HandleStatusTable(w http.ResponseWriter, r *http.Request)
}
