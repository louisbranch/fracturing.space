package scenarios

import (
	"net/http"
	"strings"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	scenariosmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/scenarios"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides scenarios routes.
type Module struct {
	service scenariosmodule.Service
}

// New returns a scenarios module.
func New(service scenariosmodule.Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "scenarios" }

// Mount wires scenarios routes.
func (m Module) Mount() (mod.Mount, error) {
	mux := http.NewServeMux()
	if m.service == nil {
		mux.HandleFunc(routepath.ScenariosPrefix, http.NotFound)
		return mod.Mount{Prefix: routepath.ScenariosPrefix, Handler: mux}, nil
	}
	mux.HandleFunc(routepath.Scenarios, m.service.HandleScenarios)
	mux.HandleFunc(routepath.ScenariosRun, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		routeWithPath(w, r, routepath.Scenarios, m.service.HandleScenarios)
	})
	mux.HandleFunc(routepath.ScenariosPrefix, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/table") {
			http.NotFound(w, r)
			return
		}
		scenariosmodule.HandleScenarioPath(w, r, m.service)
	})
	return mod.Mount{Prefix: routepath.ScenariosPrefix, Handler: mux}, nil
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
