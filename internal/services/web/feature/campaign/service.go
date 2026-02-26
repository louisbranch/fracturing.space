package campaign

import (
	"net/http"

	routing "github.com/louisbranch/fracturing.space/internal/services/web/feature/routing"
)

// PairParamHandler is a callback handler for routes with two path identifiers.
type PairParamHandler func(http.ResponseWriter, *http.Request, string, string)

// Handlers configures callback-backed campaign service construction.
type Handlers struct {
	Campaigns                 http.HandlerFunc
	CampaignCreate            http.HandlerFunc
	CampaignOverview          routing.StringParamHandler
	CampaignSessions          routing.StringParamHandler
	CampaignSessionStart      routing.StringParamHandler
	CampaignSessionEnd        routing.StringParamHandler
	CampaignSessionDetail     PairParamHandler
	CampaignParticipants      routing.StringParamHandler
	CampaignParticipantUpdate routing.StringParamHandler
	CampaignCharacters        routing.StringParamHandler
	CampaignCharacterCreate   routing.StringParamHandler
	CampaignCharacterUpdate   routing.StringParamHandler
	CampaignCharacterControl  routing.StringParamHandler
	CampaignCharacterDetail   PairParamHandler
	CampaignInvites           routing.StringParamHandler
	CampaignInviteCreate      routing.StringParamHandler
	CampaignInviteRevoke      routing.StringParamHandler
}

type callbackService struct {
	handlers Handlers
}

// NewService builds a campaign Service backed by handler callbacks.
func NewService(handlers Handlers) Service {
	return callbackService{handlers: handlers}
}

func (s callbackService) HandleCampaigns(w http.ResponseWriter, r *http.Request) {
	routing.CallOrNotFound(w, r, s.handlers.Campaigns)
}

func (s callbackService) HandleCampaignCreate(w http.ResponseWriter, r *http.Request) {
	routing.CallOrNotFound(w, r, s.handlers.CampaignCreate)
}

func (s callbackService) HandleCampaignOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignOverview, campaignID)
}

func (s callbackService) HandleCampaignSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignSessions, campaignID)
}

func (s callbackService) HandleCampaignSessionStart(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignSessionStart, campaignID)
}

func (s callbackService) HandleCampaignSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignSessionEnd, campaignID)
}

func (s callbackService) HandleCampaignSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	if s.handlers.CampaignSessionDetail == nil {
		http.NotFound(w, r)
		return
	}
	s.handlers.CampaignSessionDetail(w, r, campaignID, sessionID)
}

func (s callbackService) HandleCampaignParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignParticipants, campaignID)
}

func (s callbackService) HandleCampaignParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignParticipantUpdate, campaignID)
}

func (s callbackService) HandleCampaignCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignCharacters, campaignID)
}

func (s callbackService) HandleCampaignCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignCharacterCreate, campaignID)
}

func (s callbackService) HandleCampaignCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignCharacterUpdate, campaignID)
}

func (s callbackService) HandleCampaignCharacterControl(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignCharacterControl, campaignID)
}

func (s callbackService) HandleCampaignCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	if s.handlers.CampaignCharacterDetail == nil {
		http.NotFound(w, r)
		return
	}
	s.handlers.CampaignCharacterDetail(w, r, campaignID, characterID)
}

func (s callbackService) HandleCampaignInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignInvites, campaignID)
}

func (s callbackService) HandleCampaignInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignInviteCreate, campaignID)
}

func (s callbackService) HandleCampaignInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.CampaignInviteRevoke, campaignID)
}
