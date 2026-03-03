package scenarios

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func newRoutes(service Service) *http.ServeMux {
	mux := http.NewServeMux()
	if service == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppScenarios, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.ScenariosPrefix+"{$}", http.NotFound)
		return mux
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppScenarios, service.HandleScenarios)
	mux.HandleFunc(http.MethodGet+" "+routepath.ScenariosPrefix+"{$}", service.HandleScenarios)
	mux.HandleFunc(http.MethodGet+" "+routepath.ScenariosRun, func(w http.ResponseWriter, r *http.Request) {
		methodNotAllowed(w, r, http.MethodPost)
	})
	mux.HandleFunc(http.MethodPost+" "+routepath.AppScenarios, service.HandleScenarios)
	mux.HandleFunc(http.MethodPost+" "+routepath.ScenariosRun, service.HandleScenarios)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppScenarioEventsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			service.HandleScenarioEventsTable(w, r, campaignID)
			return
		}
		service.HandleScenarioEvents(w, r, campaignID)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.AppScenarioTimelinePattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		service.HandleScenarioTimelineTable(w, r, campaignID)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.ScenariosPrefix+"{campaignID}/{rest...}", http.NotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.ScenariosPrefix+"{campaignID}/{rest...}", http.NotFound)
	return mux
}

func wantsRowsFragment(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get(routepath.FragmentQueryKey)), routepath.FragmentRows)
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request, allowedMethods ...string) {
	if len(allowedMethods) > 0 {
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	}
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}
