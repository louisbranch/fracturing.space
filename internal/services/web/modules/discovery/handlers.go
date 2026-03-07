package discovery

import (
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

	var listings []webtemplates.StarterListingView
	if h.gateway != nil {
		results, err := h.gateway.ListStarterListings(r.Context())
		if err != nil {
			log.Printf("discovery: list starter listings: %v", err)
			// Render empty list on error — soft degradation.
		} else {
			listings = mapListingsToView(results)
		}
	}

	h.WritePublicPage(
		w,
		r,
		webtemplates.T(loc, "web.discovery.title"),
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		http.StatusOK,
		webtemplates.DiscoveryFragment(listings, loc),
	)
}

// mapListingsToView converts gateway domain types to template view types.
func mapListingsToView(listings []StarterListing) []webtemplates.StarterListingView {
	if len(listings) == 0 {
		return nil
	}
	views := make([]webtemplates.StarterListingView, len(listings))
	for i, l := range listings {
		views[i] = webtemplates.StarterListingView{
			CampaignID:  l.CampaignID,
			Title:       l.Title,
			Description: l.Description,
			Tags:        l.Tags,
			Difficulty:  l.Difficulty,
			Duration:    l.Duration,
			GmMode:      l.GmMode,
			System:      l.System,
			Level:       l.Level,
			Players:     l.Players,
		}
	}
	return views
}
