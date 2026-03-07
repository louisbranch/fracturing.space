package icons

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func newRoutes(h Handlers) *http.ServeMux {
	mux := http.NewServeMux()
	if h == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppIcons, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.IconsPrefix+"{$}", http.NotFound)
		return mux
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppIcons, func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			h.HandleIconsTable(w, r)
			return
		}
		h.HandleIconsPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.IconsPrefix+"{$}", func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			h.HandleIconsTable(w, r)
			return
		}
		h.HandleIconsPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.IconsPrefix+"{rest...}", http.NotFound)
	return mux
}

func wantsRowsFragment(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get(routepath.FragmentQueryKey)), routepath.FragmentRows)
}
