package dashboard

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func newRoutes(h Handlers) *http.ServeMux {
	mux := http.NewServeMux()
	if h == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppDashboard, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.DashboardPrefix+"{$}", http.NotFound)
		return mux
	}

	mux.HandleFunc(http.MethodGet+" "+routepath.AppDashboard, func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			h.HandleDashboardContent(w, r)
			return
		}
		h.HandleDashboard(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.DashboardPrefix+"{$}", func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			h.HandleDashboardContent(w, r)
			return
		}
		h.HandleDashboard(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.DashboardPrefix+"{rest...}", http.NotFound)

	return mux
}

func wantsRowsFragment(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get(routepath.FragmentQueryKey)), routepath.FragmentRows)
}
