package systems

import "net/http"

// Handlers defines systems handler methods consumed by this module's routes.
type Handlers interface {
	HandleSystemsPage(w http.ResponseWriter, r *http.Request)
	HandleSystemsTable(w http.ResponseWriter, r *http.Request)
	HandleSystemDetail(w http.ResponseWriter, r *http.Request, systemID string)
}
