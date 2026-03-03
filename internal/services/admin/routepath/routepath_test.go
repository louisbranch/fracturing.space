package routepath

import "testing"

func TestTopLevelRoutes(t *testing.T) {
	t.Parallel()

	if Root != "/" {
		t.Fatalf("Root = %q", Root)
	}
	if StaticPrefix != "/static/" {
		t.Fatalf("StaticPrefix = %q", StaticPrefix)
	}
	if DashboardContent != "/app/dashboard?fragment=rows" {
		t.Fatalf("DashboardContent = %q", DashboardContent)
	}
	if Campaigns != "/app/campaigns" {
		t.Fatalf("Campaigns = %q", Campaigns)
	}
	if CampaignsTable != "/app/campaigns?fragment=rows" {
		t.Fatalf("CampaignsTable = %q", CampaignsTable)
	}
	if Systems != "/app/systems" {
		t.Fatalf("Systems = %q", Systems)
	}
	if SystemsTable != "/app/systems?fragment=rows" {
		t.Fatalf("SystemsTable = %q", SystemsTable)
	}
	if Catalog != "/app/catalog" {
		t.Fatalf("Catalog = %q", Catalog)
	}
	if Icons != "/app/icons" {
		t.Fatalf("Icons = %q", Icons)
	}
	if Users != "/app/users" {
		t.Fatalf("Users = %q", Users)
	}
	if Scenarios != "/app/scenarios" {
		t.Fatalf("Scenarios = %q", Scenarios)
	}
}

func TestCampaignBuilders(t *testing.T) {
	t.Parallel()

	if got := Campaign("camp-1"); got != "/app/campaigns/camp-1" {
		t.Fatalf("Campaign = %q", got)
	}
	if got := CampaignCharacters("camp-1"); got != "/app/campaigns/camp-1/characters" {
		t.Fatalf("CampaignCharacters = %q", got)
	}
	if got := CampaignCharacter("camp-1", "char-1"); got != "/app/campaigns/camp-1/characters/char-1" {
		t.Fatalf("CampaignCharacter = %q", got)
	}
	if got := CampaignCharacterActivity("camp-1", "char-1"); got != "/app/campaigns/camp-1/characters/char-1/activity" {
		t.Fatalf("CampaignCharacterActivity = %q", got)
	}
	if got := CampaignParticipants("camp-1"); got != "/app/campaigns/camp-1/participants" {
		t.Fatalf("CampaignParticipants = %q", got)
	}
	if got := CampaignInvites("camp-1"); got != "/app/campaigns/camp-1/invites" {
		t.Fatalf("CampaignInvites = %q", got)
	}
	if got := CampaignSessions("camp-1"); got != "/app/campaigns/camp-1/sessions" {
		t.Fatalf("CampaignSessions = %q", got)
	}
	if got := CampaignSession("camp-1", "sess-1"); got != "/app/campaigns/camp-1/sessions/sess-1" {
		t.Fatalf("CampaignSession = %q", got)
	}
	if got := CampaignSessionEvents("camp-1", "sess-1"); got != "/app/campaigns/camp-1/sessions/sess-1/events" {
		t.Fatalf("CampaignSessionEvents = %q", got)
	}
	if got := CampaignEvents("camp-1"); got != "/app/campaigns/camp-1/events" {
		t.Fatalf("CampaignEvents = %q", got)
	}
	if got := CampaignEventsTable("camp-1"); got != "/app/campaigns/camp-1/events?fragment=rows" {
		t.Fatalf("CampaignEventsTable = %q", got)
	}
}

func TestCatalogAndUserBuilders(t *testing.T) {
	t.Parallel()

	if got := System("daggerheart"); got != "/app/systems/daggerheart" {
		t.Fatalf("System = %q", got)
	}
	if got := CatalogSection("daggerheart", "classes"); got != "/app/catalog/daggerheart/classes" {
		t.Fatalf("CatalogSection = %q", got)
	}
	if got := CatalogSectionTable("daggerheart", "classes"); got != "/app/catalog/daggerheart/classes?fragment=rows" {
		t.Fatalf("CatalogSectionTable = %q", got)
	}
	if got := CatalogEntry("daggerheart", "classes", "class-1"); got != "/app/catalog/daggerheart/classes/class-1" {
		t.Fatalf("CatalogEntry = %q", got)
	}
	if got := UserDetail("u-1"); got != "/app/users/u-1" {
		t.Fatalf("UserDetail = %q", got)
	}
	if got := UserInvites("u-1"); got != "/app/users/u-1/invites" {
		t.Fatalf("UserInvites = %q", got)
	}
}

func TestScenarioBuilders(t *testing.T) {
	t.Parallel()

	if got := ScenarioEvents("camp-1"); got != "/app/scenarios/camp-1/events" {
		t.Fatalf("ScenarioEvents = %q", got)
	}
	if got := ScenarioEventsTable("camp-1"); got != "/app/scenarios/camp-1/events?fragment=rows" {
		t.Fatalf("ScenarioEventsTable = %q", got)
	}
	if got := ScenarioTimelineTable("camp-1"); got != "/app/scenarios/camp-1/timeline?fragment=rows" {
		t.Fatalf("ScenarioTimelineTable = %q", got)
	}
}

func TestBuildersEscapeSegments(t *testing.T) {
	t.Parallel()

	if got := Campaign("camp/1"); got != "/app/campaigns/camp%2F1" {
		t.Fatalf("Campaign escaped = %q", got)
	}
	if got := UserDetail("user/1"); got != "/app/users/user%2F1" {
		t.Fatalf("UserDetail escaped = %q", got)
	}
	if got := CatalogEntry("dagger heart", "class/cards", "entry 1"); got != "/app/catalog/dagger%20heart/class%2Fcards/entry%201" {
		t.Fatalf("CatalogEntry escaped = %q", got)
	}
}
