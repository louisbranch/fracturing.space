package web

import (
	"html"
	"io"
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

func (h *handler) handleAppCampaignSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignSessions renders active/archived campaign sessions so
	// participants can reason about current play context.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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

	resp, err := h.sessionClient.ListSessions(r.Context(), &statev1.ListSessionsRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Sessions unavailable", "failed to list sessions")
		return
	}

	renderAppCampaignSessionsPage(w, campaignID, resp.GetSessions(), canManageSessions)
}

func (h *handler) handleAppCampaignSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	// handleAppCampaignSessionDetail returns one session record from the session
	// read model for the campaign workspace.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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

	renderAppCampaignSessionDetailPage(w, campaignID, resp.GetSession())
}

func (h *handler) handleAppCampaignSessionStart(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignSessionStart creates a new live session for campaign play
	// and requires manager/owner permission.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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

	http.Redirect(w, r, "/app/campaigns/"+campaignID+"/sessions", http.StatusFound)
}

func (h *handler) handleAppCampaignSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignSessionEnd transitions a session to ended state once
	// authorization checks are satisfied.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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

	http.Redirect(w, r, "/app/campaigns/"+campaignID+"/sessions", http.StatusFound)
}

func canManageCampaignSessions(access statev1.CampaignAccess) bool {
	return access == statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER || access == statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
}

func renderAppCampaignSessionsPage(w http.ResponseWriter, campaignID string, sessions []*statev1.Session, canManageSessions bool) {
	// renderAppCampaignSessionsPage maps session models to a navigable list and
	// conditionally exposes end actions only for managers/owners.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	escapedCampaignID := html.EscapeString(campaignID)
	_, _ = io.WriteString(w, "<!doctype html><html><head><title>Sessions</title></head><body><h1>Sessions</h1>")
	if canManageSessions {
		_, _ = io.WriteString(w, "<form method=\"post\" action=\"/app/campaigns/"+escapedCampaignID+"/sessions/start\"><input type=\"text\" name=\"name\" placeholder=\"session name\"><button type=\"submit\">Start Session</button></form>")
	}
	_, _ = io.WriteString(w, "<ul>")
	for _, session := range sessions {
		if session == nil {
			continue
		}
		sessionID := strings.TrimSpace(session.GetId())
		name := strings.TrimSpace(session.GetName())
		if name == "" {
			name = sessionID
		}
		_, _ = io.WriteString(w, "<li>")
		if sessionID != "" {
			_, _ = io.WriteString(w, "<a href=\"/app/campaigns/"+escapedCampaignID+"/sessions/"+html.EscapeString(sessionID)+"\">"+html.EscapeString(name)+"</a>")
		} else {
			_, _ = io.WriteString(w, html.EscapeString(name))
		}
		if canManageSessions && session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
			if sessionID != "" {
				_, _ = io.WriteString(w, "<form method=\"post\" action=\"/app/campaigns/"+escapedCampaignID+"/sessions/end\"><input type=\"hidden\" name=\"session_id\" value=\""+html.EscapeString(sessionID)+"\"><button type=\"submit\">End Session</button></form>")
			}
		}
		_, _ = io.WriteString(w, "</li>")
	}
	_, _ = io.WriteString(w, "</ul></body></html>")
}

func renderAppCampaignSessionDetailPage(w http.ResponseWriter, campaignID string, session *statev1.Session) {
	// renderAppCampaignSessionDetailPage renders the canonical read surface for one
	// game session and links back into the session list.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	escapedCampaignID := html.EscapeString(campaignID)
	sessionID := strings.TrimSpace(session.GetId())
	sessionName := strings.TrimSpace(session.GetName())
	if sessionName == "" {
		sessionName = sessionID
	}
	_, _ = io.WriteString(w, "<!doctype html><html><head><title>Session</title></head><body><h1>"+html.EscapeString(sessionName)+"</h1>")
	if sessionID != "" {
		_, _ = io.WriteString(w, "<p>Session ID: "+html.EscapeString(sessionID)+"</p>")
	}
	_, _ = io.WriteString(w, "<p>Status: "+html.EscapeString(sessionStatusLabel(session.GetStatus()))+"</p>")
	_, _ = io.WriteString(w, "<p><a href=\"/app/campaigns/"+escapedCampaignID+"/sessions\">Back to Sessions</a></p>")
	_, _ = io.WriteString(w, "</body></html>")
}

func sessionStatusLabel(status statev1.SessionStatus) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return "active"
	case statev1.SessionStatus_SESSION_ENDED:
		return "ended"
	default:
		return "unspecified"
	}
}
