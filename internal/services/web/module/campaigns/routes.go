package campaigns

import (
	"net/http"
	"strings"

	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Service is the campaign workspace transport contract consumed by the route module.
type Service interface {
	HandleCampaigns(w http.ResponseWriter, r *http.Request)
	HandleCampaignCreate(w http.ResponseWriter, r *http.Request)
	HandleCampaignOverview(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignSessions(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignSessionStart(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string)
	HandleCampaignParticipants(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacters(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacterControl(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID string, characterID string)
	HandleCampaignInvites(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string)
}

// RegisterRoutes wires campaign workspace routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.AppCampaigns, service.HandleCampaigns)
	mux.HandleFunc(routepath.AppCampaignsCreate, service.HandleCampaignCreate)
	mux.HandleFunc(routepath.AppCampaignsPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleCampaignDetailPath(w, r, service)
	})
}

// HandleCampaignDetailPath parses campaign workspace subpaths and dispatches to campaign handlers.
func HandleCampaignDetailPath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, routepath.AppCampaignsPrefix)
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "//") {
		http.NotFound(w, r)
		return
	}
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
		service.HandleCampaignSessions(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "sessions" && parts[2] == "start" {
		service.HandleCampaignSessionStart(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "sessions" && parts[2] == "end" {
		service.HandleCampaignSessionEnd(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "sessions" {
		service.HandleCampaignSessionDetail(w, r, parts[0], parts[2])
		return
	}
	if len(parts) == 2 && parts[1] == "participants" {
		service.HandleCampaignParticipants(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "participants" && parts[2] == "update" {
		service.HandleCampaignParticipantUpdate(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "characters" {
		service.HandleCampaignCharacters(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "characters" && parts[2] == "create" {
		service.HandleCampaignCharacterCreate(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "characters" && parts[2] == "update" {
		service.HandleCampaignCharacterUpdate(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "characters" && parts[2] == "control" {
		service.HandleCampaignCharacterControl(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "characters" {
		service.HandleCampaignCharacterDetail(w, r, parts[0], parts[2])
		return
	}
	if len(parts) == 2 && parts[1] == "invites" {
		service.HandleCampaignInvites(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "invites" && parts[2] == "create" {
		service.HandleCampaignInviteCreate(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "invites" && parts[2] == "revoke" {
		service.HandleCampaignInviteRevoke(w, r, parts[0])
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}

	service.HandleCampaignOverview(w, r, parts[0])
}
