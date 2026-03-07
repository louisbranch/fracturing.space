package systems

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func newRoutes(h Handlers) *http.ServeMux {
	mux := http.NewServeMux()
	if h == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppSystems, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.SystemsPrefix+"{$}", http.NotFound)
		return mux
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSystems, func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			h.HandleSystemsTable(w, r)
			return
		}
		h.HandleSystemsPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.SystemsPrefix+"{$}", func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			h.HandleSystemsTable(w, r)
			return
		}
		h.HandleSystemsPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSystemPattern, func(w http.ResponseWriter, r *http.Request) {
		systemID := strings.TrimSpace(r.PathValue("systemID"))
		if systemID == "" {
			http.NotFound(w, r)
			return
		}
		h.HandleSystemDetail(w, r, systemID)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.SystemsPrefix+"{systemID}/{rest...}", http.NotFound)
	return mux
}

func wantsRowsFragment(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get(routepath.FragmentQueryKey)), routepath.FragmentRows)
}
