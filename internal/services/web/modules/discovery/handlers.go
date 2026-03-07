package discovery

import (
	"context"
	"log"
	"net/http"

	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	publichandler.Base
	gateway Gateway
}

// newHandlers builds package wiring for this web seam.
func newHandlers(base publichandler.Base, gw Gateway) handlers {
	return handlers{Base: base, gateway: gw}
}

// handleIndex handles this route in the module transport layer.
func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, lang := webi18n.ResolveLocalizer(w, r, nil)
	entries := h.loadStarterEntriesView(r.Context())
	h.writeDiscoveryPage(w, r, loc, lang, entries)
}

// loadStarterEntriesView loads discovery entries and soft-degrades to an empty
// list when the gateway is unavailable.
func (h handlers) loadStarterEntriesView(ctx context.Context) []webtemplates.StarterEntryView {
	if h.gateway == nil {
		return nil
	}
	results, err := h.gateway.ListStarterEntries(ctx)
	if err != nil {
		log.Printf("discovery: list starter entries: %v", err)
		// Render empty list on error — soft degradation.
		return nil
	}
	return mapEntriesToView(results)
}

// writeDiscoveryPage writes the discovery page shell and content fragment.
func (h handlers) writeDiscoveryPage(
	w http.ResponseWriter,
	r *http.Request,
	loc webtemplates.Localizer,
	lang string,
	entries []webtemplates.StarterEntryView,
) {
	h.WritePublicPage(
		w,
		r,
		webtemplates.T(loc, "web.discovery.title"),
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		http.StatusOK,
		webtemplates.DiscoveryFragment(entries, loc),
	)
}
