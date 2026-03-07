package scenarios

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func newRoutes(h Handlers) *http.ServeMux {
	mux := http.NewServeMux()
	if h == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppScenarios, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.ScenariosPrefix+"{$}", http.NotFound)
		return mux
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppScenarios, h.HandleScenarios)
	mux.HandleFunc(http.MethodGet+" "+routepath.ScenariosPrefix+"{$}", h.HandleScenarios)
	mux.HandleFunc(http.MethodGet+" "+routepath.ScenariosRun, func(w http.ResponseWriter, r *http.Request) {
		methodNotAllowed(w, r, http.MethodPost)
	})
	mux.HandleFunc(http.MethodPost+" "+routepath.AppScenarios, h.HandleScenarios)
	mux.HandleFunc(http.MethodPost+" "+routepath.ScenariosRun, h.HandleScenarios)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppScenarioEventsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			h.HandleScenarioEventsTable(w, r, campaignID)
			return
		}
		h.HandleScenarioEvents(w, r, campaignID)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.AppScenarioTimelinePattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		h.HandleScenarioTimelineTable(w, r, campaignID)
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
