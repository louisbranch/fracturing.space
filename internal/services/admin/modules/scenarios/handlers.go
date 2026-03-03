package scenarios

import "net/http"

// Service defines scenario handlers consumed by this module's routes.
type Service interface {
	HandleScenarios(w http.ResponseWriter, r *http.Request)
	HandleScenarioEvents(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleScenarioEventsTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleScenarioTimelineTable(w http.ResponseWriter, r *http.Request, campaignID string)
}
