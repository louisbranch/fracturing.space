package web

import (
	"net/http"

	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// renderCampaignPage renders the shared campaign shell once access has been
// verified by route-level auth and campaign membership checks.
func (h *handler) renderCampaignPage(w http.ResponseWriter, r *http.Request, campaignID string) {
	printer, lang := localizer(w, r)
	page := webtemplates.PageContext{
		Lang:         lang,
		Loc:          printer,
		CurrentPath:  r.URL.Path,
		CurrentQuery: r.URL.RawQuery,
	}
	templ.Handler(webtemplates.CampaignPage(page, campaignID)).ServeHTTP(w, r)
}
