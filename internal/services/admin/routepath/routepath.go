package routepath

import (
	"net/url"
	"strings"
)

const (
	Root = "/"
)

const (
	StaticPrefix = "/static/"
)

const (
	Dashboard        = "/dashboard"
	DashboardAlt     = "/dashboard/"
	DashboardStats   = "/dashboard/_stats"
	DashboardContent = DashboardStats
)

const (
	Campaigns       = "/campaigns"
	CampaignsCreate = "/campaigns/create"
	CampaignsRows   = "/campaigns/_rows"
	CampaignsTable  = CampaignsRows
	CampaignsPrefix = "/campaigns/"
)

const (
	Systems       = "/systems"
	SystemsRows   = "/systems/_rows"
	SystemsTable  = SystemsRows
	SystemsPrefix = "/systems/"
)

const (
	Catalog       = "/catalog"
	CatalogPrefix = "/catalog/"
)

const (
	Icons       = "/icons"
	IconsPrefix = "/icons/"
	IconsRows   = "/icons/_rows"
	IconsTable  = IconsRows
)

const (
	Users       = "/users"
	UsersRows   = "/users/_rows"
	UsersTable  = UsersRows
	UsersLookup = "/users/lookup"
	UsersCreate = "/users/create"
	UsersPrefix = "/users/"
)

const (
	Scenarios       = "/scenarios"
	ScenariosRun    = "/scenarios/run"
	ScenariosPrefix = "/scenarios/"
)

func Campaign(campaignID string) string {
	return Campaigns + "/" + escapeSegment(campaignID)
}

func CampaignCharacters(campaignID string) string {
	return Campaign(campaignID) + "/characters"
}

func CampaignCharactersRows(campaignID string) string {
	return CampaignCharacters(campaignID) + "/_rows"
}

func CampaignCharactersTable(campaignID string) string {
	return CampaignCharactersRows(campaignID)
}

func CampaignCharacter(campaignID string, characterID string) string {
	return CampaignCharacters(campaignID) + "/" + escapeSegment(characterID)
}

func CampaignCharacterActivity(campaignID string, characterID string) string {
	return CampaignCharacter(campaignID, characterID) + "/activity"
}

func CampaignParticipants(campaignID string) string {
	return Campaign(campaignID) + "/participants"
}

func CampaignParticipantsRows(campaignID string) string {
	return CampaignParticipants(campaignID) + "/_rows"
}

func CampaignParticipantsTable(campaignID string) string {
	return CampaignParticipantsRows(campaignID)
}

func CampaignInvites(campaignID string) string {
	return Campaign(campaignID) + "/invites"
}

func CampaignInvitesRows(campaignID string) string {
	return CampaignInvites(campaignID) + "/_rows"
}

func CampaignInvitesTable(campaignID string) string {
	return CampaignInvitesRows(campaignID)
}

func CampaignSessions(campaignID string) string {
	return Campaign(campaignID) + "/sessions"
}

func CampaignSessionsRows(campaignID string) string {
	return CampaignSessions(campaignID) + "/_rows"
}

func CampaignSessionsTable(campaignID string) string {
	return CampaignSessionsRows(campaignID)
}

func CampaignSession(campaignID string, sessionID string) string {
	return CampaignSessions(campaignID) + "/" + escapeSegment(sessionID)
}

func CampaignSessionEvents(campaignID string, sessionID string) string {
	return CampaignSession(campaignID, sessionID) + "/events"
}

func CampaignEvents(campaignID string) string {
	return Campaign(campaignID) + "/events"
}

func CampaignEventsRows(campaignID string) string {
	return CampaignEvents(campaignID) + "/_rows"
}

func CampaignEventsTable(campaignID string) string {
	return CampaignEventsRows(campaignID)
}

func System(systemID string) string {
	return Systems + "/" + escapeSegment(systemID)
}

func CatalogSection(systemID string, sectionID string) string {
	return Catalog + "/" + escapeSegment(systemID) + "/" + escapeSegment(sectionID)
}

func CatalogSectionRows(systemID string, sectionID string) string {
	return CatalogSection(systemID, sectionID) + "/_rows"
}

func CatalogSectionTable(systemID string, sectionID string) string {
	return CatalogSectionRows(systemID, sectionID)
}

func CatalogEntry(systemID string, sectionID string, entryID string) string {
	return CatalogSection(systemID, sectionID) + "/" + escapeSegment(entryID)
}

func UserDetail(userID string) string {
	return Users + "/" + escapeSegment(userID)
}

func UserInvites(userID string) string {
	return UserDetail(userID) + "/invites"
}

func ScenarioEvents(campaignID string) string {
	return Scenarios + "/" + escapeSegment(campaignID) + "/events"
}

func ScenarioEventsRows(campaignID string) string {
	return ScenarioEvents(campaignID) + "/_rows"
}

func ScenarioEventsTable(campaignID string) string {
	return ScenarioEventsRows(campaignID)
}

func ScenarioTimelineRows(campaignID string) string {
	return Scenarios + "/" + escapeSegment(campaignID) + "/timeline/_rows"
}

func ScenarioTimelineTable(campaignID string) string {
	return ScenarioTimelineRows(campaignID)
}

func escapeSegment(raw string) string {
	return url.PathEscape(strings.TrimSpace(raw))
}
