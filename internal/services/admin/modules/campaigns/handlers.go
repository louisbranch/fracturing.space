package campaigns

import "net/http"

// Service defines campaign handlers consumed by this module's routes.
type Service interface {
	HandleCampaignsPage(w http.ResponseWriter, r *http.Request)
	HandleCampaignsTable(w http.ResponseWriter, r *http.Request)
	HandleCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCharactersList(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCharactersTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string)
	HandleCharacterActivity(w http.ResponseWriter, r *http.Request, campaignID string, characterID string)
	HandleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleInvitesList(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleInvitesTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleSessionsList(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleSessionsTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string)
	HandleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string)
	HandleEventLog(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string)
}
