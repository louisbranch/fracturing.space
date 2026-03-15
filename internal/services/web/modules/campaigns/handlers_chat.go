package campaigns

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playorigin"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
)

// handleGame redirects the campaign game route into the dedicated play surface.
func (h sessionHandlers) handleGame(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, err := h.loadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	userID := strings.TrimSpace(h.RequestUserID(r))
	if userID == "" {
		h.WriteError(w, r, http.ErrNoCookie)
		return
	}
	grant, _, err := playlaunchgrant.Issue(h.playLaunchGrant, playlaunchgrant.IssueInput{
		GrantID:    strconv.FormatInt(h.now().UnixNano(), 10),
		CampaignID: strings.TrimSpace(campaignID),
		UserID:     userID,
	})
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	_ = ctx
	_ = page
	target := playorigin.PlayURL(
		r,
		h.requestMeta,
		h.playFallbackPort,
		"/campaigns/"+url.PathEscape(campaignID)+"?launch="+url.QueryEscape(grant),
	)
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, target)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
