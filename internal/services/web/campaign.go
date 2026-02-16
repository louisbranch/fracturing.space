package web

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) handleCampaignPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/campaigns/")
	path = strings.Trim(path, "/")
	if path == "" || strings.Contains(path, "/") {
		http.NotFound(w, r)
		return
	}

	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if h.campaignAccess == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Campaign unavailable", "campaign access checker is not configured")
		return
	}
	allowed, err := h.campaignAccess.IsCampaignParticipant(r.Context(), path, sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Campaign unavailable", "failed to verify campaign access")
		return
	}
	if !allowed {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant access required")
		return
	}
	h.renderCampaignPage(w, r, path)
}

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
