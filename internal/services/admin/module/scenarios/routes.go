package scenarios

import (
	"net/http"
	"strings"

	sharedpath "github.com/louisbranch/fracturing.space/internal/services/admin/module/sharedpath"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
)

// Service defines scenario route handlers consumed by this route module.
type Service interface {
	HandleScenarios(w http.ResponseWriter, r *http.Request)
	HandleScenarioEvents(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleScenarioEventsTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleScenarioTimelineTable(w http.ResponseWriter, r *http.Request, campaignID string)
}

// RegisterRoutes wires scenario routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Scenarios, service.HandleScenarios)
	mux.HandleFunc(routepath.ScenariosPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleScenarioPath(w, r, service)
	})
}

// HandleScenarioPath parses scenario subroutes and dispatches to service handlers.
func HandleScenarioPath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, routepath.ScenariosPrefix)
	parts := sharedpath.SplitPathParts(path)

	if len(parts) == 2 && parts[1] == "events" {
		service.HandleScenarioEvents(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "events" && parts[2] == "table" {
		service.HandleScenarioEventsTable(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "timeline" && parts[2] == "table" {
		service.HandleScenarioTimelineTable(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}
