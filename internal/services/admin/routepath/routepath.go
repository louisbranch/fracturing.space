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
	AppPrefix = "/app/"
)

const (
	FragmentQueryKey = "fragment"
	FragmentRows     = "rows"
)

const (
	AppDashboard     = "/app/dashboard"
	DashboardPrefix  = "/app/dashboard/"
	Dashboard        = AppDashboard
	DashboardAlt     = DashboardPrefix
	DashboardStats   = "/app/dashboard?fragment=rows"
	DashboardContent = DashboardStats
)

const (
	AppCampaigns    = "/app/campaigns"
	Campaigns       = AppCampaigns
	CampaignsRows   = "/app/campaigns?fragment=rows"
	CampaignsTable  = CampaignsRows
	CampaignsPrefix = "/app/campaigns/"
)

const (
	AppSystems    = "/app/systems"
	Systems       = AppSystems
	SystemsRows   = "/app/systems?fragment=rows"
	SystemsTable  = SystemsRows
	SystemsPrefix = "/app/systems/"
)

const (
	AppCatalog    = "/app/catalog"
	Catalog       = AppCatalog
	CatalogPrefix = "/app/catalog/"
)

const (
	AppIcons    = "/app/icons"
	Icons       = AppIcons
	IconsPrefix = "/app/icons/"
	IconsRows   = "/app/icons?fragment=rows"
	IconsTable  = IconsRows
)

const (
	AppUsers    = "/app/users"
	Users       = AppUsers
	UsersRows   = "/app/users?fragment=rows"
	UsersTable  = UsersRows
	UsersLookup = "/app/users/lookup"
	UsersCreate = "/app/users/create"
	UsersPrefix = "/app/users/"
)

const (
	AppScenarios    = "/app/scenarios"
	Scenarios       = AppScenarios
	ScenariosRun    = "/app/scenarios/run"
	ScenariosPrefix = "/app/scenarios/"
)

const (
	AppCampaignPattern                  = CampaignsPrefix + "{campaignID}"
	AppCampaignCharactersPattern        = CampaignsPrefix + "{campaignID}/characters"
	AppCampaignCharacterPattern         = CampaignsPrefix + "{campaignID}/characters/{characterID}"
	AppCampaignCharacterActivityPattern = CampaignsPrefix + "{campaignID}/characters/{characterID}/activity"
	AppCampaignParticipantsPattern      = CampaignsPrefix + "{campaignID}/participants"
	AppCampaignInvitesPattern           = CampaignsPrefix + "{campaignID}/invites"
	AppCampaignSessionsPattern          = CampaignsPrefix + "{campaignID}/sessions"
	AppCampaignSessionPattern           = CampaignsPrefix + "{campaignID}/sessions/{sessionID}"
	AppCampaignSessionEventsPattern     = CampaignsPrefix + "{campaignID}/sessions/{sessionID}/events"
	AppCampaignEventsPattern            = CampaignsPrefix + "{campaignID}/events"
)

const (
	AppSystemPattern = SystemsPrefix + "{systemID}"
)

const (
	AppCatalogSectionPattern = CatalogPrefix + "{systemID}/{sectionID}"
	AppCatalogEntryPattern   = CatalogPrefix + "{systemID}/{sectionID}/{entryID}"
)

const (
	AppUserPattern        = UsersPrefix + "{userID}"
	AppUserInvitesPattern = UsersPrefix + "{userID}/invites"
)

const (
	AppScenarioEventsPattern   = ScenariosPrefix + "{campaignID}/events"
	AppScenarioTimelinePattern = ScenariosPrefix + "{campaignID}/timeline"
)

func Campaign(campaignID string) string {
	return AppCampaigns + "/" + escapeSegment(campaignID)
}

func CampaignCharacters(campaignID string) string {
	return Campaign(campaignID) + "/characters"
}

func CampaignCharactersRows(campaignID string) string {
	return withRowsFragment(CampaignCharacters(campaignID))
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
	return withRowsFragment(CampaignParticipants(campaignID))
}

func CampaignParticipantsTable(campaignID string) string {
	return CampaignParticipantsRows(campaignID)
}

func CampaignInvites(campaignID string) string {
	return Campaign(campaignID) + "/invites"
}

func CampaignInvitesRows(campaignID string) string {
	return withRowsFragment(CampaignInvites(campaignID))
}

func CampaignInvitesTable(campaignID string) string {
	return CampaignInvitesRows(campaignID)
}

func CampaignSessions(campaignID string) string {
	return Campaign(campaignID) + "/sessions"
}

func CampaignSessionsRows(campaignID string) string {
	return withRowsFragment(CampaignSessions(campaignID))
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
	return withRowsFragment(CampaignEvents(campaignID))
}

func CampaignEventsTable(campaignID string) string {
	return CampaignEventsRows(campaignID)
}

func System(systemID string) string {
	return AppSystems + "/" + escapeSegment(systemID)
}

func CatalogSection(systemID string, sectionID string) string {
	return AppCatalog + "/" + escapeSegment(systemID) + "/" + escapeSegment(sectionID)
}

func CatalogSectionRows(systemID string, sectionID string) string {
	return withRowsFragment(CatalogSection(systemID, sectionID))
}

func CatalogSectionTable(systemID string, sectionID string) string {
	return CatalogSectionRows(systemID, sectionID)
}

func CatalogEntry(systemID string, sectionID string, entryID string) string {
	return CatalogSection(systemID, sectionID) + "/" + escapeSegment(entryID)
}

func UserDetail(userID string) string {
	return AppUsers + "/" + escapeSegment(userID)
}

func UserInvites(userID string) string {
	return UserDetail(userID) + "/invites"
}

func ScenarioEvents(campaignID string) string {
	return AppScenarios + "/" + escapeSegment(campaignID) + "/events"
}

func ScenarioEventsRows(campaignID string) string {
	return withRowsFragment(ScenarioEvents(campaignID))
}

func ScenarioEventsTable(campaignID string) string {
	return ScenarioEventsRows(campaignID)
}

func ScenarioTimeline(campaignID string) string {
	return AppScenarios + "/" + escapeSegment(campaignID) + "/timeline"
}

func ScenarioTimelineRows(campaignID string) string {
	return withRowsFragment(ScenarioTimeline(campaignID))
}

func ScenarioTimelineTable(campaignID string) string {
	return ScenarioTimelineRows(campaignID)
}

func withRowsFragment(path string) string {
	return withQueryParam(path, FragmentQueryKey, FragmentRows)
}

func withQueryParam(path string, key string, value string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	parsed, err := url.Parse(path)
	if err != nil {
		return path
	}
	q := parsed.Query()
	q.Set(strings.TrimSpace(key), strings.TrimSpace(value))
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func escapeSegment(raw string) string {
	return url.PathEscape(strings.TrimSpace(raw))
}
