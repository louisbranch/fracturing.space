package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

type routeSurface uint8

const (
	routeSurfaceFull routeSurface = iota
	routeSurfaceStableWorkflow
)

func registerRoutes(mux *http.ServeMux, h handlers) {
	registerRoutesForSurface(mux, h, routeSurfaceFull)
}

func registerStableRoutes(mux *http.ServeMux, h handlers) {
	registerRoutesForSurface(mux, h, routeSurfaceStableWorkflow)
}

func registerRoutesForSurface(mux *http.ServeMux, h handlers, surface routeSurface) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaigns, h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.CampaignsPrefix+"{$}", h.handleIndex)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignsNew, h.handleStartNewCampaign)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignsCreate, h.handleCreateCampaign)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignsCreate, h.handleCreateCampaignSubmit)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignPattern, h.handleOverviewRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignPattern, h.handleOverviewMethodNotAllowed)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantsPattern, h.handleParticipantsRoute)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharactersPattern, h.handleCharactersRoute)

	if surface == routeSurfaceFull || surface == routeSurfaceStableWorkflow {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterPattern, h.handleCharacterDetailRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreatePattern, h.handleCharacterCreateRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationStepPattern, h.handleCharacterCreationStepRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationResetPattern, h.handleCharacterCreationResetRoute)
	}

	if surface == routeSurfaceFull {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionsPattern, h.handleSessionsRoute)
		mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionPattern, h.handleSessionDetailRoute)
		mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignGamePattern, h.handleGameRoute)
		mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignInvitesPattern, h.handleInvitesRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionStartPattern, h.handleSessionStartRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionEndPattern, h.handleSessionEndRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignParticipantUpdatePattern, h.handleParticipantUpdateRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterUpdatePattern, h.handleCharacterUpdateRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlPattern, h.handleCharacterControlRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteCreatePattern, h.handleInviteCreateRoute)
		mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteRevokePattern, h.handleInviteRevokeRoute)
	}

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignRestPattern, h.handleNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignRestPattern, h.handleNotFound)
}

type detailRouteKind string

const (
	detailOverview          detailRouteKind = "overview"
	detailSessions          detailRouteKind = "sessions"
	detailSessionStart      detailRouteKind = "session-start"
	detailSessionEnd        detailRouteKind = "session-end"
	detailSession           detailRouteKind = "session-detail"
	detailParticipants      detailRouteKind = "participants"
	detailParticipantUpdate detailRouteKind = "participant-update"
	detailCharacters        detailRouteKind = "characters"
	detailGame              detailRouteKind = "game"
	detailCharacterCreate   detailRouteKind = "character-create"
	detailCharacterUpdate   detailRouteKind = "character-update"
	detailCharacterControl  detailRouteKind = "character-control"
	detailCharacter         detailRouteKind = "character-detail"
	detailInvites           detailRouteKind = "invites"
	detailInviteCreate      detailRouteKind = "invite-create"
	detailInviteRevoke      detailRouteKind = "invite-revoke"
)

type detailRoute struct {
	campaignID  string
	kind        detailRouteKind
	sessionID   string
	characterID string
}

func (k detailRouteKind) marker() string {
	switch k {
	case detailOverview:
		return "campaign-overview"
	case detailSessions:
		return "campaign-sessions"
	case detailSessionStart:
		return "campaign-session-start"
	case detailSessionEnd:
		return "campaign-session-end"
	case detailSession:
		return "campaign-session-detail"
	case detailParticipants:
		return "campaign-participants"
	case detailParticipantUpdate:
		return "campaign-participant-update"
	case detailCharacters:
		return "campaign-characters"
	case detailGame:
		return "campaign-game"
	case detailCharacterCreate:
		return "campaign-character-create"
	case detailCharacterUpdate:
		return "campaign-character-update"
	case detailCharacterControl:
		return "campaign-character-control"
	case detailCharacter:
		return "campaign-character-detail"
	case detailInvites:
		return "campaign-invites"
	case detailInviteCreate:
		return "campaign-invite-create"
	case detailInviteRevoke:
		return "campaign-invite-revoke"
	default:
		return "campaign-overview"
	}
}
