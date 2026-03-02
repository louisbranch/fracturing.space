package campaigns

import (
	"net/http"
	"strings"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	campaignsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides campaigns routes.
type Module struct {
	service campaignsmodule.Service
}

// New returns a campaigns module.
func New(service campaignsmodule.Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "campaigns" }

// Mount wires campaigns routes.
func (m Module) Mount() (mod.Mount, error) {
	mux := http.NewServeMux()
	if m.service == nil {
		mux.HandleFunc(routepath.CampaignsPrefix, http.NotFound)
		return mod.Mount{Prefix: routepath.CampaignsPrefix, Handler: mux}, nil
	}

	mux.HandleFunc(routepath.Campaigns, m.service.HandleCampaignsPage)
	mux.HandleFunc(routepath.CampaignsRows, m.service.HandleCampaignsTable)
	mux.HandleFunc(routepath.CampaignsPrefix, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/table") {
			http.NotFound(w, r)
			return
		}
		campaignsmodule.HandleCampaignPath(w, r, m.service)
	})
	return mod.Mount{Prefix: routepath.CampaignsPrefix, Handler: mux}, nil
}
