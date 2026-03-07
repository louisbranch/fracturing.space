package icons

import "net/http"

// Handlers defines icon handler methods consumed by this module's routes.
type Handlers interface {
	HandleIconsPage(w http.ResponseWriter, r *http.Request)
	HandleIconsTable(w http.ResponseWriter, r *http.Request)
}
