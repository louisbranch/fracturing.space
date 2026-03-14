package discovery

import (
	"net/http"

	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	publichandler.Base
	service discoveryapp.Service
}

// newHandlers builds package wiring for this web seam.
func newHandlers(base publichandler.Base, service discoveryapp.Service) handlers {
	if service == nil {
		service = discoveryapp.NewService(nil)
	}
	return handlers{Base: base, service: service}
}

// handleIndex handles this route in the module transport layer.
func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, lang := webi18n.ResolveLocalizer(w, r, nil)
	page := h.service.LoadPage(r.Context())
	entries := mapEntriesToView(page.Entries)
	h.writeDiscoveryPage(w, r, loc, lang, entries)
}

// writeDiscoveryPage writes the discovery page shell and content fragment.
func (h handlers) writeDiscoveryPage(
	w http.ResponseWriter,
	r *http.Request,
	loc webtemplates.Localizer,
	lang string,
	entries []StarterEntryView,
) {
	h.WritePublicPage(
		w,
		r,
		webtemplates.T(loc, "web.discovery.title"),
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		http.StatusOK,
		DiscoveryFragment(entries, loc),
	)
}
