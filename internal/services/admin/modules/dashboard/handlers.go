package dashboard

import "net/http"

// Service defines dashboard handlers consumed by this module's routes.
type Service interface {
	HandleDashboard(w http.ResponseWriter, r *http.Request)
	HandleDashboardContent(w http.ResponseWriter, r *http.Request)
}
