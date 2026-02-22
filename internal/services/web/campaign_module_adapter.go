package web

import (
	"net/http"

	campaignsmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/campaigns"
)

type campaignModuleService struct {
	handler *handler
}

func newCampaignModuleService(h *handler) campaignsmodule.Service {
	if h == nil {
		return nil
	}
	return campaignModuleService{handler: h}
}

func (s campaignModuleService) HandleCampaigns(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppCampaigns(w, r)
}

func (s campaignModuleService) HandleCampaignCreate(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppCampaignCreate(w, r)
}

func (s campaignModuleService) HandleCampaignOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignOverview(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignSessions(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignSessionStart(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignSessionStart(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignSessionEnd(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	s.handler.handleAppCampaignSessionDetail(w, r, campaignID, sessionID)
}

func (s campaignModuleService) HandleCampaignParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignParticipants(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignParticipantUpdate(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignCharacters(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignCharacters(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignCharacterCreate(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignCharacterUpdate(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignCharacterControl(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignCharacterControl(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	s.handler.handleAppCampaignCharacterDetail(w, r, campaignID, characterID)
}

func (s campaignModuleService) HandleCampaignInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignInvites(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignInviteCreate(w, r, campaignID)
}

func (s campaignModuleService) HandleCampaignInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleAppCampaignInviteRevoke(w, r, campaignID)
}
