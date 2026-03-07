package scenarios

import "net/http"

// Handlers defines scenario handler methods consumed by this module's routes.
type Handlers interface {
	HandleScenarios(w http.ResponseWriter, r *http.Request)
	HandleScenarioEvents(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleScenarioEventsTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleScenarioTimelineTable(w http.ResponseWriter, r *http.Request, campaignID string)
}
