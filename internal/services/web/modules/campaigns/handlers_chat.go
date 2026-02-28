package campaigns

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// --- Campaign chat route ---

func (h handlers) handleGame(w http.ResponseWriter, r *http.Request, campaignID string) {
	_, page, err := h.loadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := webtemplates.CampaignChatView{
		CampaignID:       campaignID,
		CampaignName:     page.workspace.Name,
		BackURL:          routepath.AppCampaign(campaignID),
		ChatFallbackPort: strings.TrimSpace(h.chatFallbackPort),
	}
	h.writeCampaignChatHTML(w, r, view, page.lang, page.loc)
}

func (h handlers) writeCampaignChatHTML(
	w http.ResponseWriter,
	r *http.Request,
	view webtemplates.CampaignChatView,
	lang string,
	loc webtemplates.Localizer,
) {
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, routepath.AppCampaignGame(view.CampaignID))
		return
	}
	if err := webtemplates.CampaignChatPage(view, lang, loc).Render(r.Context(), w); err != nil {
		h.WriteError(w, r, err)
	}
}
