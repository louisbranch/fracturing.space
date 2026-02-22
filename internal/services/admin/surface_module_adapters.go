package admin

import (
	"net/http"

	campaignsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/campaigns"
	catalogmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/catalog"
	dashboardmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/dashboard"
	iconsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/icons"
	scenariosmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/scenarios"
	systemsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/systems"
	usersmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/users"
)

type dashboardModuleService struct {
	handler *Handler
}

func newDashboardModuleService(h *Handler) dashboardmodule.Service {
	if h == nil {
		return nil
	}
	return dashboardModuleService{handler: h}
}

func (s dashboardModuleService) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	s.handler.handleDashboard(w, r)
}

func (s dashboardModuleService) HandleDashboardContent(w http.ResponseWriter, r *http.Request) {
	s.handler.handleDashboardContent(w, r)
}

type campaignsModuleService struct {
	handler *Handler
}

func newCampaignsModuleService(h *Handler) campaignsmodule.Service {
	if h == nil {
		return nil
	}
	return campaignsModuleService{handler: h}
}

func (s campaignsModuleService) HandleCampaignsPage(w http.ResponseWriter, r *http.Request) {
	s.handler.handleCampaignsPage(w, r)
}

func (s campaignsModuleService) HandleCampaignsTable(w http.ResponseWriter, r *http.Request) {
	s.handler.handleCampaignsTable(w, r)
}

func (s campaignsModuleService) HandleCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleCampaignDetail(w, r, campaignID)
}

func (s campaignsModuleService) HandleCharactersList(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleCharactersList(w, r, campaignID)
}

func (s campaignsModuleService) HandleCharactersTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleCharactersTable(w, r, campaignID)
}

func (s campaignsModuleService) HandleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	s.handler.handleCharacterSheet(w, r, campaignID, characterID)
}

func (s campaignsModuleService) HandleCharacterActivity(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	s.handler.handleCharacterActivity(w, r, campaignID, characterID)
}

func (s campaignsModuleService) HandleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleParticipantsList(w, r, campaignID)
}

func (s campaignsModuleService) HandleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleParticipantsTable(w, r, campaignID)
}

func (s campaignsModuleService) HandleInvitesList(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleInvitesList(w, r, campaignID)
}

func (s campaignsModuleService) HandleInvitesTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleInvitesTable(w, r, campaignID)
}

func (s campaignsModuleService) HandleSessionsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleSessionsList(w, r, campaignID)
}

func (s campaignsModuleService) HandleSessionsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleSessionsTable(w, r, campaignID)
}

func (s campaignsModuleService) HandleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	s.handler.handleSessionDetail(w, r, campaignID, sessionID)
}

func (s campaignsModuleService) HandleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	s.handler.handleSessionEvents(w, r, campaignID, sessionID)
}

func (s campaignsModuleService) HandleEventLog(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleEventLog(w, r, campaignID)
}

func (s campaignsModuleService) HandleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleEventLogTable(w, r, campaignID)
}

type systemsModuleService struct {
	handler *Handler
}

func newSystemsModuleService(h *Handler) systemsmodule.Service {
	if h == nil {
		return nil
	}
	return systemsModuleService{handler: h}
}

func (s systemsModuleService) HandleSystemsPage(w http.ResponseWriter, r *http.Request) {
	s.handler.handleSystemsPage(w, r)
}

func (s systemsModuleService) HandleSystemsTable(w http.ResponseWriter, r *http.Request) {
	s.handler.handleSystemsTable(w, r)
}

func (s systemsModuleService) HandleSystemDetail(w http.ResponseWriter, r *http.Request, systemID string) {
	s.handler.handleSystemDetail(w, r, systemID)
}

type catalogModuleService struct {
	handler *Handler
}

func newCatalogModuleService(h *Handler) catalogmodule.Service {
	if h == nil {
		return nil
	}
	return catalogModuleService{handler: h}
}

func (s catalogModuleService) HandleCatalogPage(w http.ResponseWriter, r *http.Request) {
	s.handler.handleCatalogPage(w, r)
}

func (s catalogModuleService) HandleCatalogSection(w http.ResponseWriter, r *http.Request, sectionID string) {
	s.handler.handleCatalogSection(w, r, sectionID)
}

func (s catalogModuleService) HandleCatalogSectionTable(w http.ResponseWriter, r *http.Request, sectionID string) {
	s.handler.handleCatalogSectionTable(w, r, sectionID)
}

func (s catalogModuleService) HandleCatalogSectionDetail(w http.ResponseWriter, r *http.Request, sectionID string, entryID string) {
	s.handler.handleCatalogSectionDetail(w, r, sectionID, entryID)
}

type iconsModuleService struct {
	handler *Handler
}

func newIconsModuleService(h *Handler) iconsmodule.Service {
	if h == nil {
		return nil
	}
	return iconsModuleService{handler: h}
}

func (s iconsModuleService) HandleIconsPage(w http.ResponseWriter, r *http.Request) {
	s.handler.handleIconsPage(w, r)
}

func (s iconsModuleService) HandleIconsTable(w http.ResponseWriter, r *http.Request) {
	s.handler.handleIconsTable(w, r)
}

type usersModuleService struct {
	handler *Handler
}

func newUsersModuleService(h *Handler) usersmodule.Service {
	if h == nil {
		return nil
	}
	return usersModuleService{handler: h}
}

func (s usersModuleService) HandleUsersPage(w http.ResponseWriter, r *http.Request) {
	s.handler.handleUsersPage(w, r)
}

func (s usersModuleService) HandleUsersTable(w http.ResponseWriter, r *http.Request) {
	s.handler.handleUsersTable(w, r)
}

func (s usersModuleService) HandleUserLookup(w http.ResponseWriter, r *http.Request) {
	s.handler.handleUserLookup(w, r)
}

func (s usersModuleService) HandleMagicLink(w http.ResponseWriter, r *http.Request) {
	s.handler.handleMagicLink(w, r)
}

func (s usersModuleService) HandleUserDetail(w http.ResponseWriter, r *http.Request, userID string) {
	s.handler.handleUserDetail(w, r, userID)
}

func (s usersModuleService) HandleUserInvites(w http.ResponseWriter, r *http.Request, userID string) {
	s.handler.handleUserInvites(w, r, userID)
}

type scenariosModuleService struct {
	handler *Handler
}

func newScenariosModuleService(h *Handler) scenariosmodule.Service {
	if h == nil {
		return nil
	}
	return scenariosModuleService{handler: h}
}

func (s scenariosModuleService) HandleScenarios(w http.ResponseWriter, r *http.Request) {
	s.handler.handleScenarios(w, r)
}

func (s scenariosModuleService) HandleScenarioEvents(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleScenarioEvents(w, r, campaignID)
}

func (s scenariosModuleService) HandleScenarioEventsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleScenarioEventsTable(w, r, campaignID)
}

func (s scenariosModuleService) HandleScenarioTimelineTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	s.handler.handleScenarioTimelineTable(w, r, campaignID)
}
