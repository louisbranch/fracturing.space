package web

import (
	"net/http"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignsmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/campaigns"
)

// handleAppCampaignDetail parses campaign workspace routes and dispatches each
// subpath to the ownership/authorization-aware leaf handler.
func (h *handler) handleAppCampaignDetail(w http.ResponseWriter, r *http.Request) {
	campaignsmodule.HandleCampaignDetailPath(w, r, buildCampaignModuleService(h))
}

func (h *handler) handleAppCampaignOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	readCtx, _, ok := h.campaignReadContext(w, r, "Campaign unavailable")
	if !ok {
		return
	}

	if h.campaignClient != nil {
		resp, err := h.campaignClient.GetCampaign(readCtx, &statev1.GetCampaignRequest{CampaignId: campaignID})
		if err != nil {
			h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Campaign unavailable", "failed to load campaign")
			return
		}
		if resp == nil || resp.GetCampaign() == nil {
			h.renderErrorPage(w, r, http.StatusNotFound, "Campaign unavailable", "campaign not found")
			return
		}
		h.setCampaignCache(readCtx, resp.GetCampaign())
	}

	h.renderCampaignPage(w, r.WithContext(readCtx), campaignID)
}
