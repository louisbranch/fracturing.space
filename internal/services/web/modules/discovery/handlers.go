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

	var entries []webtemplates.StarterEntryView
	if h.gateway != nil {
		results, err := h.gateway.ListStarterEntries(r.Context())
		if err != nil {
			log.Printf("discovery: list starter entries: %v", err)
			// Render empty list on error — soft degradation.
		} else {
			entries = mapEntriesToView(results)
		}
	}

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

// mapEntriesToView converts gateway domain types to template view types.
func mapEntriesToView(entries []StarterEntry) []webtemplates.StarterEntryView {
	if len(entries) == 0 {
		return nil
	}
	views := make([]webtemplates.StarterEntryView, len(entries))
	for i, entry := range entries {
		views[i] = webtemplates.StarterEntryView{
			CampaignID:  entry.CampaignID,
			Title:       entry.Title,
			Description: entry.Description,
			Tags:        entry.Tags,
			Difficulty:  entry.Difficulty,
			Duration:    entry.Duration,
			GmMode:      entry.GmMode,
			System:      entry.System,
			Level:       entry.Level,
			Players:     entry.Players,
		}
	}
	return views
}
