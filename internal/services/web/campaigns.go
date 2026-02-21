package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) handleAppCampaigns(w http.ResponseWriter, r *http.Request) {
	// Campaign list is the web entrypoint into the campaign read model and is
	// intentionally participant-only to keep campaign listing scoped by active
	// participant identity.
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

	if err := h.ensureCampaignClients(r.Context()); err != nil {
		renderAppCampaignsListPageWithAppName(w, r, h.resolvedAppName(), []*statev1.Campaign{})
		return
	}
	if h.campaignClient == nil || h.campaignAccess == nil {
		renderAppCampaignsListPageWithAppName(w, r, h.resolvedAppName(), []*statev1.Campaign{})
		return
	}

	requestCtx := r.Context()
	participantID, err := h.sessionParticipantID(requestCtx, sess.accessToken)
	listCampaigns := func(ctx context.Context) ([]*statev1.Campaign, error) {
		resp, err := h.campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
		if err != nil {
			return nil, err
		}
		return resp.GetCampaigns(), nil
	}

	if err != nil || strings.TrimSpace(participantID) == "" {
		renderAppCampaignsListPageWithAppName(w, r, h.resolvedAppName(), []*statev1.Campaign{})
		return
	}

	requestCtx = grpcauthctx.WithParticipantID(requestCtx, participantID)
	campaigns, err := listCampaigns(requestCtx)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Campaigns unavailable", "failed to list campaigns")
		return
	}
	renderAppCampaignsListPageWithAppName(w, r, h.resolvedAppName(), campaigns)
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
		renderAppCampaignCreatePageWithAppName(w, r, h.resolvedAppName())
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
	if err := h.ensureCampaignClients(r.Context()); err != nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Campaign create unavailable", "campaign service client is not configured")
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

func (h *handler) sessionParticipantID(ctx context.Context, accessToken string) (string, error) {
	if h == nil || h.campaignAccess == nil {
		return "", errors.New("campaign access checker is not configured")
	}
	svc, ok := h.campaignAccess.(*campaignAccessService)
	if !ok {
		return "", errors.New("campaign access checker does not support participant introspection")
	}
	return svc.introspectParticipantID(ctx, accessToken)
}

func (h *handler) ensureCampaignClients(ctx context.Context) error {
	if h == nil {
		return errors.New("web handler is not configured")
	}
	if h.campaignClient != nil {
		return nil
	}

	h.clientInitMu.Lock()
	defer h.clientInitMu.Unlock()

	if h.campaignClient != nil {
		return nil
	}

	_, participantClient, campaignClient, sessionClient, characterClient, inviteClient, err := dialGameGRPC(ctx, h.config)
	if err != nil {
		return err
	}
	if campaignClient == nil || sessionClient == nil || participantClient == nil || characterClient == nil || inviteClient == nil {
		return errors.New("campaign service client is not configured")
	}

	h.participantClient = participantClient
	h.campaignClient = campaignClient
	h.sessionClient = sessionClient
	h.characterClient = characterClient
	h.inviteClient = inviteClient
	h.campaignAccess = newCampaignAccessChecker(h.config, participantClient)
	return nil
}

func renderAppCampaignsPage(w http.ResponseWriter, r *http.Request, campaigns []*statev1.Campaign) {
	renderAppCampaignsListPageWithAppName(w, r, "", campaigns)
}

func renderAppCampaignsListPageWithAppName(w http.ResponseWriter, r *http.Request, appName string, campaigns []*statev1.Campaign) {
	// renderAppCampaignsListPageWithAppName maps the list of campaign read models into
	// links that become the canonical campaign navigation point for this boundary.
	normalized := make([]webtemplates.CampaignListItem, 0, len(campaigns))
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		campaignID := strings.TrimSpace(campaign.GetId())
		name := strings.TrimSpace(campaign.GetName())
		if name == "" {
			name = campaignID
		}
		normalized = append(normalized, webtemplates.CampaignListItem{
			ID:   campaignID,
			Name: name,
		})
	}
	writeGameContentType(w)
	if err := webtemplates.CampaignsListPage(appName, normalized).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render campaigns list page", http.StatusInternalServerError)
	}
}

func renderAppCampaignCreatePage(w http.ResponseWriter, r *http.Request) {
	renderAppCampaignCreatePageWithAppName(w, r, "")
}

func renderAppCampaignCreatePageWithAppName(w http.ResponseWriter, r *http.Request, appName string) {
	// renderAppCampaignCreatePageWithAppName renders the campaign creation form used by
	// the create flow.
	writeGameContentType(w)
	if err := webtemplates.CampaignCreatePage(appName).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render campaign create page", http.StatusInternalServerError)
	}
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
