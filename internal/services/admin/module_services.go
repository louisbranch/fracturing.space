package admin

import (
	"net/http"
)

func (h *Handler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	h.handleDashboard(w, r)
}

func (h *Handler) HandleDashboardContent(w http.ResponseWriter, r *http.Request) {
	h.handleDashboardContent(w, r)
}

func (h *Handler) HandleCampaignsPage(w http.ResponseWriter, r *http.Request) {
	h.handleCampaignsPage(w, r)
}

func (h *Handler) HandleCampaignsTable(w http.ResponseWriter, r *http.Request) {
	h.handleCampaignsTable(w, r)
}

func (h *Handler) HandleCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleCampaignDetail(w, r, campaignID)
}

func (h *Handler) HandleCharactersList(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleCharactersList(w, r, campaignID)
}

func (h *Handler) HandleCharactersTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleCharactersTable(w, r, campaignID)
}

func (h *Handler) HandleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	h.handleCharacterSheet(w, r, campaignID, characterID)
}

func (h *Handler) HandleCharacterActivity(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	h.handleCharacterActivity(w, r, campaignID, characterID)
}

func (h *Handler) HandleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleParticipantsList(w, r, campaignID)
}

func (h *Handler) HandleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleParticipantsTable(w, r, campaignID)
}

func (h *Handler) HandleInvitesList(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleInvitesList(w, r, campaignID)
}

func (h *Handler) HandleInvitesTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleInvitesTable(w, r, campaignID)
}

func (h *Handler) HandleSessionsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleSessionsList(w, r, campaignID)
}

func (h *Handler) HandleSessionsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleSessionsTable(w, r, campaignID)
}

func (h *Handler) HandleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	h.handleSessionDetail(w, r, campaignID, sessionID)
}

func (h *Handler) HandleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	h.handleSessionEvents(w, r, campaignID, sessionID)
}

func (h *Handler) HandleEventLog(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleEventLog(w, r, campaignID)
}

func (h *Handler) HandleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleEventLogTable(w, r, campaignID)
}

func (h *Handler) HandleSystemsPage(w http.ResponseWriter, r *http.Request) {
	h.handleSystemsPage(w, r)
}

func (h *Handler) HandleSystemsTable(w http.ResponseWriter, r *http.Request) {
	h.handleSystemsTable(w, r)
}

func (h *Handler) HandleSystemDetail(w http.ResponseWriter, r *http.Request, systemID string) {
	h.handleSystemDetail(w, r, systemID)
}

func (h *Handler) HandleCatalogPage(w http.ResponseWriter, r *http.Request) {
	h.handleCatalogPage(w, r)
}

func (h *Handler) HandleCatalogSection(w http.ResponseWriter, r *http.Request, sectionID string) {
	h.handleCatalogSection(w, r, sectionID)
}

func (h *Handler) HandleCatalogSectionTable(w http.ResponseWriter, r *http.Request, sectionID string) {
	h.handleCatalogSectionTable(w, r, sectionID)
}

func (h *Handler) HandleCatalogSectionDetail(w http.ResponseWriter, r *http.Request, sectionID string, entryID string) {
	h.handleCatalogSectionDetail(w, r, sectionID, entryID)
}

func (h *Handler) HandleIconsPage(w http.ResponseWriter, r *http.Request) {
	h.handleIconsPage(w, r)
}

func (h *Handler) HandleIconsTable(w http.ResponseWriter, r *http.Request) {
	h.handleIconsTable(w, r)
}

func (h *Handler) HandleUsersPage(w http.ResponseWriter, r *http.Request) {
	h.handleUsersPage(w, r)
}

func (h *Handler) HandleUsersTable(w http.ResponseWriter, r *http.Request) {
	h.handleUsersTable(w, r)
}

func (h *Handler) HandleUserLookup(w http.ResponseWriter, r *http.Request) {
	h.handleUserLookup(w, r)
}

func (h *Handler) HandleMagicLink(w http.ResponseWriter, r *http.Request) {
	h.handleMagicLink(w, r)
}

func (h *Handler) HandleUserDetail(w http.ResponseWriter, r *http.Request, userID string) {
	h.handleUserDetail(w, r, userID)
}

func (h *Handler) HandleUserInvites(w http.ResponseWriter, r *http.Request, userID string) {
	h.handleUserInvites(w, r, userID)
}

func (h *Handler) HandleScenarios(w http.ResponseWriter, r *http.Request) {
	h.handleScenarios(w, r)
}

func (h *Handler) HandleScenarioEvents(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleScenarioEvents(w, r, campaignID)
}

func (h *Handler) HandleScenarioEventsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleScenarioEventsTable(w, r, campaignID)
}

func (h *Handler) HandleScenarioTimelineTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	h.handleScenarioTimelineTable(w, r, campaignID)
}
