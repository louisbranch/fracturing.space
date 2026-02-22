package campaigns

import (
	"net/http"
	"strings"

	sharedpath "github.com/louisbranch/fracturing.space/internal/services/admin/module/sharedpath"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
)

// Service defines campaign route handlers consumed by this route module.
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

// RegisterRoutes wires campaign routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Campaigns, service.HandleCampaignsPage)
	mux.HandleFunc(routepath.CampaignsTable, service.HandleCampaignsTable)
	mux.HandleFunc(routepath.CampaignsPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleCampaignPath(w, r, service)
	})
}

// HandleCampaignPath parses campaign subroutes and dispatches to service handlers.
func HandleCampaignPath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, routepath.CampaignsPrefix)
	parts := sharedpath.SplitPathParts(path)
	if len(parts) == 1 && strings.EqualFold(parts[0], "create") {
		http.NotFound(w, r)
		return
	}

	// /campaigns/{id}/characters
	if len(parts) == 2 && parts[1] == "characters" {
		service.HandleCharactersList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/characters/table
	if len(parts) == 3 && parts[1] == "characters" && parts[2] == "table" {
		service.HandleCharactersTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/characters/{characterId}
	if len(parts) == 3 && parts[1] == "characters" {
		service.HandleCharacterSheet(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/characters/{characterId}/activity
	if len(parts) == 4 && parts[1] == "characters" && parts[3] == "activity" {
		service.HandleCharacterActivity(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/participants
	if len(parts) == 2 && parts[1] == "participants" {
		service.HandleParticipantsList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/participants/table
	if len(parts) == 3 && parts[1] == "participants" && parts[2] == "table" {
		service.HandleParticipantsTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/invites
	if len(parts) == 2 && parts[1] == "invites" {
		service.HandleInvitesList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/invites/table
	if len(parts) == 3 && parts[1] == "invites" && parts[2] == "table" {
		service.HandleInvitesTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/sessions
	if len(parts) == 2 && parts[1] == "sessions" {
		service.HandleSessionsList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/sessions/table
	if len(parts) == 3 && parts[1] == "sessions" && parts[2] == "table" {
		service.HandleSessionsTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/sessions/{sessionId}
	if len(parts) == 3 && parts[1] == "sessions" {
		service.HandleSessionDetail(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/sessions/{sessionId}/events
	if len(parts) == 4 && parts[1] == "sessions" && parts[3] == "events" {
		service.HandleSessionEvents(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/events
	if len(parts) == 2 && parts[1] == "events" {
		service.HandleEventLog(w, r, parts[0])
		return
	}
	// /campaigns/{id}/events/table
	if len(parts) == 3 && parts[1] == "events" && parts[2] == "table" {
		service.HandleEventLogTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}
	if len(parts) == 1 && strings.TrimSpace(parts[0]) != "" {
		service.HandleCampaignDetail(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}
