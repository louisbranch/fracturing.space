package systems

import (
	"net/http"
	"strings"

	sharedpath "github.com/louisbranch/fracturing.space/internal/services/admin/module/sharedpath"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
)

// Service defines systems route handlers consumed by this route module.
type Service interface {
	HandleSystemsPage(w http.ResponseWriter, r *http.Request)
	HandleSystemsTable(w http.ResponseWriter, r *http.Request)
	HandleSystemDetail(w http.ResponseWriter, r *http.Request, systemID string)
}

// RegisterRoutes wires system routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Systems, service.HandleSystemsPage)
	mux.HandleFunc(routepath.SystemsTable, service.HandleSystemsTable)
	mux.HandleFunc(routepath.SystemsPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleSystemPath(w, r, service)
	})
}

// HandleSystemPath parses dynamic system detail routes and dispatches to service handlers.
func HandleSystemPath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, routepath.SystemsPrefix)
	parts := sharedpath.SplitPathParts(path)
	if len(parts) == 1 {
		service.HandleSystemDetail(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}
