package web

import (
	"net/http"
)

func (h *handler) HandleCampaigns(w http.ResponseWriter, r *http.Request) {
	h.handleAppCampaigns(w, r)
}

func (h *handler) HandleCampaignCreate(w http.ResponseWriter, r *http.Request) {
	h.handleAppCampaignCreate(w, r)
}

func (h *handler) HandleCampaignOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignOverview(w, r, campaignID)
}

func (h *handler) HandleCampaignSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignSessions(w, r, campaignID)
}

func (h *handler) HandleCampaignSessionStart(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignSessionStart(w, r, campaignID)
}

func (h *handler) HandleCampaignSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignSessionEnd(w, r, campaignID)
}

func (h *handler) HandleCampaignSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	h.handleAppCampaignSessionDetail(w, r, campaignID, sessionID)
}

func (h *handler) HandleCampaignParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignParticipants(w, r, campaignID)
}

func (h *handler) HandleCampaignParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignParticipantUpdate(w, r, campaignID)
}

func (h *handler) HandleCampaignCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignCharacters(w, r, campaignID)
}

func (h *handler) HandleCampaignCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignCharacterCreate(w, r, campaignID)
}

func (h *handler) HandleCampaignCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignCharacterUpdate(w, r, campaignID)
}

func (h *handler) HandleCampaignCharacterControl(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignCharacterControl(w, r, campaignID)
}

func (h *handler) HandleCampaignCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	h.handleAppCampaignCharacterDetail(w, r, campaignID, characterID)
}

func (h *handler) HandleCampaignInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignInvites(w, r, campaignID)
}

func (h *handler) HandleCampaignInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignInviteCreate(w, r, campaignID)
}

func (h *handler) HandleCampaignInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleAppCampaignInviteRevoke(w, r, campaignID)
}
