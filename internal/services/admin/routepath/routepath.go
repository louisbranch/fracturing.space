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
	DashboardContent = "/dashboard/content"
)

const (
	Campaigns       = "/campaigns"
	CampaignsCreate = "/campaigns/create"
	CampaignsTable  = "/campaigns/table"
	CampaignsPrefix = "/campaigns/"
)

const (
	Systems       = "/systems"
	SystemsTable  = "/systems/table"
	SystemsPrefix = "/systems/"
)

const (
	Catalog       = "/catalog"
	CatalogPrefix = "/catalog/"
)

const (
	Icons      = "/icons"
	IconsTable = "/icons/table"
)

const (
	Users          = "/users"
	UsersTable     = "/users/table"
	UsersLookup    = "/users/lookup"
	UsersCreate    = "/users/create"
	UsersMagicLink = "/users/magic-link"
	UsersPrefix    = "/users/"
)

const (
	Scenarios       = "/scenarios"
	ScenariosPrefix = "/scenarios/"
)

func Campaign(campaignID string) string {
	return Campaigns + "/" + escapeSegment(campaignID)
}

func CampaignCharacters(campaignID string) string {
	return Campaign(campaignID) + "/characters"
}

func CampaignCharactersTable(campaignID string) string {
	return CampaignCharacters(campaignID) + "/table"
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

func CampaignParticipantsTable(campaignID string) string {
	return CampaignParticipants(campaignID) + "/table"
}

func CampaignInvites(campaignID string) string {
	return Campaign(campaignID) + "/invites"
}

func CampaignInvitesTable(campaignID string) string {
	return CampaignInvites(campaignID) + "/table"
}

func CampaignSessions(campaignID string) string {
	return Campaign(campaignID) + "/sessions"
}

func CampaignSessionsTable(campaignID string) string {
	return CampaignSessions(campaignID) + "/table"
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

func CampaignEventsTable(campaignID string) string {
	return CampaignEvents(campaignID) + "/table"
}

func System(systemID string) string {
	return Systems + "/" + escapeSegment(systemID)
}

func CatalogSection(systemID string, sectionID string) string {
	return Catalog + "/" + escapeSegment(systemID) + "/" + escapeSegment(sectionID)
}

func CatalogSectionTable(systemID string, sectionID string) string {
	return CatalogSection(systemID, sectionID) + "/table"
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

func ScenarioEventsTable(campaignID string) string {
	return ScenarioEvents(campaignID) + "/table"
}

func ScenarioTimelineTable(campaignID string) string {
	return Scenarios + "/" + escapeSegment(campaignID) + "/timeline/table"
}

func escapeSegment(raw string) string {
	return url.PathEscape(strings.TrimSpace(raw))
}
