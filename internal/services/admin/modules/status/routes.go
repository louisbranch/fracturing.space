package status

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func newRoutes(service Service) *http.ServeMux {
	mux := http.NewServeMux()
	if service == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppStatus, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.StatusPrefix+"{$}", http.NotFound)
		return mux
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppStatus, func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			service.HandleStatusTable(w, r)
			return
		}
		service.HandleStatusPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.StatusPrefix+"{$}", func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			service.HandleStatusTable(w, r)
			return
		}
		service.HandleStatusPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.StatusPrefix+"{rest...}", http.NotFound)
	return mux
}

func wantsRowsFragment(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get(routepath.FragmentQueryKey)), routepath.FragmentRows)
}
