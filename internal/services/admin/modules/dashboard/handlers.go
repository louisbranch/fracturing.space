package dashboard

import "net/http"

// Handlers defines dashboard handler methods consumed by this module's routes.
type Handlers interface {
	HandleDashboard(w http.ResponseWriter, r *http.Request)
	HandleDashboardContent(w http.ResponseWriter, r *http.Request)
}
