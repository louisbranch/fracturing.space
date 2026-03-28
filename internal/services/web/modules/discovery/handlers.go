package discovery

import (
	"net/http"

	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	publichandler.Base
	service discoveryapp.Service
}

// newHandlersWithBase keeps package tests free to inject a custom base while
// module roots stay transport-only.
func newHandlersWithBase(base publichandler.Base, service discoveryapp.Service) handlers {
	if service == nil {
		service = discoveryapp.NewService(nil, nil)
	}
	return handlers{Base: base, service: service}
}

// newHandlers builds package wiring for the production discovery module seam.
func newHandlers(service discoveryapp.Service) handlers {
	return newHandlersWithBase(publichandler.NewBase(), service)
}

// handleIndex handles this route in the module transport layer.
func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.PageLocalizer(w, r)
	page := h.service.LoadPage(r.Context())
	view := mapPageToView(page)
	h.writeDiscoveryPage(w, r, loc, lang, view)
}

// writeDiscoveryPage writes the discovery page shell and content fragment.
func (h handlers) writeDiscoveryPage(
	w http.ResponseWriter,
	r *http.Request,
	loc webtemplates.Localizer,
	lang string,
	view DiscoveryPageView,
) {
	h.WritePublicPage(w, r, pagerender.PublicPage{
		Title:      webtemplates.T(loc, "web.discovery.title"),
		MetaDesc:   webtemplates.T(loc, "layout.meta_description"),
		Language:   lang,
		StatusCode: http.StatusOK,
		Body:       DiscoveryFragment(view, loc),
	})
}
