package web

import (
	"context"
	"errors"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

func (h *handler) handleAppCampaigns(w http.ResponseWriter, r *http.Request) {
	// Campaign list is the web entrypoint into the campaign read model and is
	// intentionally user-scoped before rendering.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if h.campaignClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Campaigns unavailable", "campaign service client is not configured")
		return
	}

	userID, err := h.sessionUserID(r.Context(), sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Campaigns unavailable", "failed to resolve current user")
		return
	}
	if userID == "" {
		h.renderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}

	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	resp, err := h.campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Campaigns unavailable", "failed to list campaigns")
		return
	}

	renderAppCampaignsPageWithAppName(w, h.resolvedAppName(), resp.GetCampaigns())
}

func (h *handler) handleAppCampaignCreate(w http.ResponseWriter, r *http.Request) {
	// Campaign create is the onboarding bridge from HTML form into typed
	// campaign service behavior.
	if r.Method == http.MethodGet {
		sess := sessionFromRequest(r, h.sessions)
		if sess == nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}
		renderAppCampaignsPageWithAppName(w, h.resolvedAppName(), nil)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if h.campaignClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Campaign create unavailable", "campaign service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Campaign create unavailable", "failed to parse campaign create form")
		return
	}
	campaignName := strings.TrimSpace(r.FormValue("name"))
	if campaignName == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Campaign create unavailable", "campaign name is required")
		return
	}
	systemValue := strings.TrimSpace(r.FormValue("system"))
	if systemValue == "" {
		systemValue = "daggerheart"
	}
	system, ok := parseAppGameSystem(systemValue)
	if !ok {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Campaign create unavailable", "campaign system is invalid")
		return
	}
	gmModeValue := strings.TrimSpace(r.FormValue("gm_mode"))
	if gmModeValue == "" {
		gmModeValue = "human"
	}
	gmMode, ok := parseAppGmMode(gmModeValue)
	if !ok {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Campaign create unavailable", "campaign gm mode is invalid")
		return
	}
	themePrompt := strings.TrimSpace(r.FormValue("theme_prompt"))
	creatorDisplayName := strings.TrimSpace(r.FormValue("creator_display_name"))
	if creatorDisplayName == "" {
		creatorDisplayName = strings.TrimSpace(sess.displayName)
	}

	userID, err := h.sessionUserID(r.Context(), sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Campaign create unavailable", "failed to resolve current user")
		return
	}
	if userID == "" {
		h.renderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}

	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	resp, err := h.campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:               campaignName,
		Locale:             commonv1.Locale_LOCALE_EN_US,
		System:             system,
		GmMode:             gmMode,
		ThemePrompt:        themePrompt,
		CreatorDisplayName: creatorDisplayName,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Campaign create unavailable", "failed to create campaign")
		return
	}
	campaignID := strings.TrimSpace(resp.GetCampaign().GetId())
	if campaignID == "" {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Campaign create unavailable", "created campaign id was empty")
		return
	}

	http.Redirect(w, r, "/campaigns/"+url.PathEscape(campaignID), http.StatusFound)
}

func (h *handler) sessionUserID(ctx context.Context, accessToken string) (string, error) {
	if h == nil || h.campaignAccess == nil {
		return "", errors.New("campaign access checker is not configured")
	}
	svc, ok := h.campaignAccess.(*campaignAccessService)
	if !ok {
		return "", errors.New("campaign access checker does not support user introspection")
	}
	return svc.introspectUserID(ctx, accessToken)
}

func renderAppCampaignsPage(w http.ResponseWriter, campaigns []*statev1.Campaign) {
	renderAppCampaignsPageWithAppName(w, "", campaigns)
}

func renderAppCampaignsPageWithAppName(w http.ResponseWriter, appName string, campaigns []*statev1.Campaign) {
	// renderAppCampaignsPage maps the list of campaign read models into links that
	// become the canonical campaign navigation point for this boundary.
	writeGamePageStart(w, "Campaigns", appName)
	_, _ = io.WriteString(w, "<h1>Campaigns</h1>")
	_, _ = io.WriteString(w, "<form method=\"post\" action=\"/campaigns/create\">")
	_, _ = io.WriteString(w, "<label>Campaign Name <input type=\"text\" name=\"name\" placeholder=\"campaign name\" required></label>")
	_, _ = io.WriteString(w, "<label>Game System <select name=\"system\"><option value=\"daggerheart\" selected>Daggerheart</option></select></label>")
	_, _ = io.WriteString(w, "<label>GM Mode <select name=\"gm_mode\"><option value=\"human\" selected>Human</option><option value=\"ai\">AI</option><option value=\"hybrid\">Hybrid</option></select></label>")
	_, _ = io.WriteString(w, "<label>Creator Display Name <input type=\"text\" name=\"creator_display_name\" placeholder=\"display name\"></label>")
	_, _ = io.WriteString(w, "<label>Theme Prompt <textarea name=\"theme_prompt\" rows=\"4\" placeholder=\"theme prompt\"></textarea></label>")
	_, _ = io.WriteString(w, "<button type=\"submit\">Create Campaign</button></form><ul>")
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		campaignID := strings.TrimSpace(campaign.GetId())
		name := strings.TrimSpace(campaign.GetName())
		if name == "" {
			name = campaignID
		}
		if campaignID != "" {
			_, _ = io.WriteString(w, "<li><a href=\"/campaigns/"+html.EscapeString(campaignID)+"\">"+html.EscapeString(name)+"</a></li>")
			continue
		}
		_, _ = io.WriteString(w, "<li>"+html.EscapeString(name)+"</li>")
	}
	_, _ = io.WriteString(w, "</ul>")
	writeGamePageEnd(w)
}

func parseAppGameSystem(value string) (commonv1.GameSystem, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "daggerheart", "game_system_daggerheart":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, true
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, false
	}
}

func parseAppGmMode(value string) (statev1.GmMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human":
		return statev1.GmMode_HUMAN, true
	case "ai":
		return statev1.GmMode_AI, true
	case "hybrid":
		return statev1.GmMode_HYBRID, true
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED, false
	}
}
