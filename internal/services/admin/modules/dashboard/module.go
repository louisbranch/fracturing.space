package dashboard

import (
	"net/http"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	dashboardmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides dashboard routes.
type Module struct {
	service dashboardmodule.Service
}

// New returns a dashboard module.
func New(service dashboardmodule.Service) Module {
	return Module{service: service}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "dashboard" }

// Mount wires dashboard routes.
func (m Module) Mount() (mod.Mount, error) {
	mux := http.NewServeMux()
	if m.service == nil {
		mux.HandleFunc(routepath.Root, http.NotFound)
		return mod.Mount{Prefix: routepath.Root, Handler: mux}, nil
	}

	mux.HandleFunc(routepath.Root, func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != routepath.Root {
			http.NotFound(w, r)
			return
		}
		routeWithPath(w, r, routepath.Root, m.service.HandleDashboard)
	})
	mux.HandleFunc(routepath.Dashboard, func(w http.ResponseWriter, r *http.Request) {
		routeWithPath(w, r, "/", m.service.HandleDashboard)
	})
	mux.HandleFunc(routepath.DashboardAlt, func(w http.ResponseWriter, r *http.Request) {
		routeWithPath(w, r, "/", m.service.HandleDashboard)
	})
	mux.HandleFunc(routepath.DashboardStats, m.service.HandleDashboardContent)

	return mod.Mount{Prefix: routepath.Root, Handler: mux}, nil
}

func routeWithPath(w http.ResponseWriter, r *http.Request, path string, next func(http.ResponseWriter, *http.Request)) {
	if next == nil {
		http.NotFound(w, r)
		return
	}
	if r == nil {
		next(w, r)
		return
	}
	clone := r.Clone(r.Context())
	if clone.URL != nil {
		urlCopy := *clone.URL
		urlCopy.Path = path
		clone.URL = &urlCopy
	}
	next(w, clone)
}
