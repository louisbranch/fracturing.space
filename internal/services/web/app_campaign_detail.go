package web

import (
	"net/http"
	"strings"
)

// handleAppCampaignDetail parses campaign workspace routes and dispatches each
// subpath to the ownership/authorization-aware leaf handler.
func (h *handler) handleAppCampaignDetail(w http.ResponseWriter, r *http.Request) {
	// Route parser for nested campaign routes.
	// This keeps sub-resources (sessions/participants/characters/invites) in one
	// place while maintaining explicit authorization checks per branch.
	path := strings.TrimPrefix(r.URL.Path, "/app/campaigns/")
	rawParts := strings.Split(path, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	if len(parts) == 2 && parts[1] == "sessions" {
		h.handleAppCampaignSessions(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "sessions" && parts[2] == "start" {
		h.handleAppCampaignSessionStart(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "sessions" && parts[2] == "end" {
		h.handleAppCampaignSessionEnd(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "sessions" {
		h.handleAppCampaignSessionDetail(w, r, parts[0], parts[2])
		return
	}
	if len(parts) == 2 && parts[1] == "participants" {
		h.handleAppCampaignParticipants(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "participants" && parts[2] == "update" {
		h.handleAppCampaignParticipantUpdate(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "characters" {
		h.handleAppCampaignCharacters(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "characters" && parts[2] == "create" {
		h.handleAppCampaignCharacterCreate(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "characters" && parts[2] == "update" {
		h.handleAppCampaignCharacterUpdate(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "characters" && parts[2] == "control" {
		h.handleAppCampaignCharacterControl(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "characters" {
		h.handleAppCampaignCharacterDetail(w, r, parts[0], parts[2])
		return
	}
	if len(parts) == 2 && parts[1] == "invites" {
		h.handleAppCampaignInvites(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "invites" && parts[2] == "create" {
		h.handleAppCampaignInviteCreate(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "invites" && parts[2] == "revoke" {
		h.handleAppCampaignInviteRevoke(w, r, parts[0])
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	campaignID := parts[0]
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.requireCampaignParticipant(w, r, campaignID) {
		return
	}

	h.renderCampaignPage(w, r, campaignID)
}
