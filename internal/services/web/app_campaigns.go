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

	renderAppCampaignsPage(w, resp.GetCampaigns())
}

func (h *handler) handleAppCampaignCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
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
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             statev1.GmMode_HUMAN,
		CreatorDisplayName: strings.TrimSpace(sess.displayName),
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

	http.Redirect(w, r, "/app/campaigns/"+url.PathEscape(campaignID), http.StatusFound)
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, "<!doctype html><html><head><title>Campaigns</title></head><body><h1>Campaigns</h1><form method=\"post\" action=\"/app/campaigns/create\"><input type=\"text\" name=\"name\" placeholder=\"campaign name\" required><button type=\"submit\">Create Campaign</button></form><ul>")
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
			_, _ = io.WriteString(w, "<li><a href=\"/app/campaigns/"+html.EscapeString(campaignID)+"\">"+html.EscapeString(name)+"</a></li>")
			continue
		}
		_, _ = io.WriteString(w, "<li>"+html.EscapeString(name)+"</li>")
	}
	_, _ = io.WriteString(w, "</ul></body></html>")
}
