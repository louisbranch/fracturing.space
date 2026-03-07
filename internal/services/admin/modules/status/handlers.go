package status

import "net/http"

// Handlers defines status handler methods consumed by this module's routes.
type Handlers interface {
	HandleStatusPage(w http.ResponseWriter, r *http.Request)
	HandleStatusTable(w http.ResponseWriter, r *http.Request)
}
