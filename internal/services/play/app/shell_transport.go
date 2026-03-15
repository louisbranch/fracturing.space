package app

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playorigin"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// shellAccess captures the resolved browser handoff decision for one campaign
// shell request.
type shellAccess struct {
	UserID        string
	RedirectToWeb bool
}

// handleRootShell renders the SPA shell for the root placeholder surface
// without requiring play-session bootstrap or campaign runtime state.
func (s *Server) handleRootShell(w http.ResponseWriter, _ *http.Request) {
	if err := s.writeShell(w, shellRenderInput{}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to render play shell")
	}
}

// handleCampaignShell resolves browser handoff and renders the SPA shell only
// once the request is anchored to a valid play session.
func (s *Server) handleCampaignShell(w http.ResponseWriter, r *http.Request, launchGrantCfg playlaunchgrant.Config) {
	campaign, ok := requireCampaignRequest(w, r)
	if !ok {
		return
	}
	access, handled := s.resolveCampaignShellAccess(w, r, campaign.CampaignID, launchGrantCfg)
	if handled {
		return
	}
	if access.RedirectToWeb {
		http.Redirect(w, r, playorigin.WebURL(r, s.requestSchemePolicy, s.webFallbackPort, routepath.AppCampaignGame(campaign.CampaignID)), http.StatusSeeOther)
		return
	}
	if _, err := s.application().bootstrap(r.Context(), playRequest{
		campaignRequest: campaign,
		UserID:          access.UserID,
	}); err != nil {
		writeRPCError(w, err)
		return
	}
	if err := s.writeCampaignShell(w, campaign.CampaignID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to render play shell")
	}
}

// writeCampaignShell keeps shell rendering isolated from session and bootstrap
// gating so shell failures stay transport-only.
func (s *Server) writeCampaignShell(w http.ResponseWriter, campaignID string) error {
	return s.writeShell(w, shellRenderInput{
		CampaignID:    campaignID,
		BootstrapPath: pathForCampaignAPI(campaignID, "bootstrap"),
		RealtimePath:  "/realtime",
		BackURL:       routepath.AppCampaignGame(campaignID),
	})
}

// writeShell keeps shell rendering isolated from transport-specific gating so
// root and campaign surfaces can share one shell writer.
func (s *Server) writeShell(w http.ResponseWriter, input shellRenderInput) error {
	html, err := s.shellAssets.renderHTML(input)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(html)
	return nil
}
