package web

import (
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) handleAppCampaignSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignSessions renders active/archived campaign sessions so
	// participants can reason about current play context.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	participant, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if h.sessionClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Sessions unavailable", "session service client is not configured")
		return
	}
	canManageSessions := canManageCampaignSessions(participant.GetCampaignAccess())
	if sessions, ok := h.cachedCampaignSessions(r.Context(), campaignID); ok {
		renderAppCampaignSessionsPage(w, r, h.pageContextForCampaign(w, r, campaignID), campaignID, sessions, canManageSessions)
		return
	}

	resp, err := h.sessionClient.ListSessions(r.Context(), &statev1.ListSessionsRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Sessions unavailable", "failed to list sessions")
		return
	}

	sessions := resp.GetSessions()
	h.setCampaignSessionsCache(r.Context(), campaignID, sessions)
	renderAppCampaignSessionsPage(w, r, h.pageContextForCampaign(w, r, campaignID), campaignID, sessions, canManageSessions)
}

func (h *handler) handleAppCampaignSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	// handleAppCampaignSessionDetail returns one session record from the session
	// read model for the campaign workspace.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if _, ok := h.requireCampaignActor(w, r, campaignID); !ok {
		return
	}
	if h.sessionClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Session unavailable", "session service client is not configured")
		return
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Session unavailable", "session id is required")
		return
	}

	resp, err := h.sessionClient.GetSession(r.Context(), &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Session unavailable", "failed to load session")
		return
	}
	if resp.GetSession() == nil {
		h.renderErrorPage(w, r, http.StatusNotFound, "Session unavailable", "session not found")
		return
	}

	renderAppCampaignSessionDetailPage(w, r, h.pageContextForCampaign(w, r, campaignID), campaignID, resp.GetSession())
}

func (h *handler) handleAppCampaignSessionStart(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignSessionStart creates a new live session for campaign play
	// and requires manager/owner permission.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if h.sessionClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Session action unavailable", "session service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Session action unavailable", "failed to parse session start form")
		return
	}
	sessionName := strings.TrimSpace(r.FormValue("name"))
	if sessionName == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Session action unavailable", "session name is required")
		return
	}

	if !canManageCampaignSessions(actor.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for session action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actor.GetId()))
	_, err := h.sessionClient.StartSession(ctx, &statev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       sessionName,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Session action unavailable", "failed to start session")
		return
	}

	http.Redirect(w, r, "/campaigns/"+campaignID+"/sessions", http.StatusFound)
}

func (h *handler) handleAppCampaignSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignSessionEnd transitions a session to ended state once
	// authorization checks are satisfied.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if h.sessionClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Session action unavailable", "session service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Session action unavailable", "failed to parse session end form")
		return
	}
	sessionID := strings.TrimSpace(r.FormValue("session_id"))
	if sessionID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Session action unavailable", "session id is required")
		return
	}

	if !canManageCampaignSessions(actor.GetCampaignAccess()) {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for session action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actor.GetId()))
	_, err := h.sessionClient.EndSession(ctx, &statev1.EndSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Session action unavailable", "failed to end session")
		return
	}

	http.Redirect(w, r, "/campaigns/"+campaignID+"/sessions", http.StatusFound)
}

func canManageCampaignSessions(access statev1.CampaignAccess) bool {
	return access == statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER || access == statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
}

func renderAppCampaignSessionsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, sessions []*statev1.Session, canManageSessions bool) {
	renderAppCampaignSessionsPageWithContext(w, r, page, campaignID, sessions, canManageSessions)
}

func renderAppCampaignSessionsPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, sessions []*statev1.Session, canManageSessions bool) {
	// renderAppCampaignSessionsPage maps session models to a navigable list and
	// conditionally exposes end actions only for managers/owners.
	campaignID = strings.TrimSpace(campaignID)
	sessionItems := make([]webtemplates.SessionListItem, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		sessionID := strings.TrimSpace(session.GetId())
		name := strings.TrimSpace(session.GetName())
		if name == "" {
			name = sessionID
		}
		sessionItems = append(sessionItems, webtemplates.SessionListItem{
			ID:       sessionID,
			Name:     name,
			IsActive: session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE,
		})
	}
	writeGameContentType(w)
	if err := webtemplates.SessionsListPage(page, campaignID, canManageSessions, sessionItems).Render(r.Context(), w); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_sessions_page")
	}
}

func renderAppCampaignSessionDetailPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, session *statev1.Session) {
	renderAppCampaignSessionDetailPageWithContext(w, r, page, campaignID, session)
}

func renderAppCampaignSessionDetailPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, session *statev1.Session) {
	// renderAppCampaignSessionDetailPage renders the canonical read surface for one
	// game session and links back into the session list.
	if session == nil {
		session = &statev1.Session{}
	}
	campaignID = strings.TrimSpace(campaignID)
	sessionID := strings.TrimSpace(session.GetId())
	sessionName := strings.TrimSpace(session.GetName())
	if sessionName == "" {
		sessionName = sessionID
	}
	detail := webtemplates.SessionDetail{
		CampaignID: campaignID,
		ID:         sessionID,
		Name:       sessionName,
		Status:     sessionStatusLabel(page.Loc, session.GetStatus()),
	}
	writeGameContentType(w)
	if err := webtemplates.SessionDetailPage(page, detail).Render(r.Context(), w); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_session_detail_page")
	}
}

func sessionStatusLabel(loc webtemplates.Localizer, status statev1.SessionStatus) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return webtemplates.T(loc, "game.session_status.active")
	case statev1.SessionStatus_SESSION_ENDED:
		return webtemplates.T(loc, "game.session_status.ended")
	default:
		return webtemplates.T(loc, "game.session_status.unspecified")
	}
}
