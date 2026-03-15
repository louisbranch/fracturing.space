package campaigns

import (
	"fmt"
	"testing"
	"time"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"golang.org/x/text/message"
)

type menuTestLocalizer map[string]string

func (l menuTestLocalizer) Sprintf(key message.Reference, args ...any) string {
	keyString := fmt.Sprint(key)
	if value, ok := l[keyString]; ok {
		return value
	}
	if len(args) == 0 {
		return keyString
	}
	return keyString
}

func TestCampaignSessionMenuStartTimeParsesUTCLayout(t *testing.T) {
	t.Parallel()

	got, ok := campaignSessionMenuStartTime(campaignapp.CampaignSession{StartedAt: "2026-03-12 20:45 UTC"})
	if !ok {
		t.Fatalf("campaignSessionMenuStartTime() ok = false, want true")
	}
	if got.Location() != time.UTC {
		t.Fatalf("location = %v, want UTC", got.Location())
	}
	if got.Year() != 2026 || got.Month() != time.March || got.Day() != 12 || got.Hour() != 20 || got.Minute() != 45 {
		t.Fatalf("parsed time = %v", got)
	}
}

func TestCampaignSessionMenuStartTimeRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []campaignapp.CampaignSession{
		{},
		{StartedAt: "bad timestamp"},
	}
	for _, session := range tests {
		if _, ok := campaignSessionMenuStartTime(session); ok {
			t.Fatalf("campaignSessionMenuStartTime(%+v) ok = true, want false", session)
		}
	}
}

func TestSortedActiveSessionsOrdersNewestFirstAndFallsBackToID(t *testing.T) {
	t.Parallel()

	sessions := []campaignapp.CampaignSession{
		{ID: "ended", Status: "ended", StartedAt: "2026-03-12 19:00 UTC"},
		{ID: "b", Status: "active", StartedAt: "2026-03-12 18:00 UTC"},
		{ID: "a", Status: "active", StartedAt: "bad timestamp"},
		{ID: "c", Status: "active", StartedAt: "2026-03-12 21:00 UTC"},
		{ID: "d", Status: "active", StartedAt: "bad timestamp"},
	}

	got := sortedActiveSessions(sessions)
	if len(got) != 4 {
		t.Fatalf("len(sortedActiveSessions) = %d, want 4", len(got))
	}
	wantIDs := []string{"c", "b", "a", "d"}
	for i, want := range wantIDs {
		if got[i].ID != want {
			t.Fatalf("sortedActiveSessions()[%d] = %q, want %q", i, got[i].ID, want)
		}
	}
}

func TestCampaignSessionMenuSubItemsIncludesOnlyActiveSessions(t *testing.T) {
	t.Parallel()

	loc := menuTestLocalizer{
		"game.sessions.menu.start":       "Start",
		"game.sessions.action_join_game": "Join",
		"game.sessions.menu.unnamed":     "Unnamed",
	}
	items := campaignSessionMenuSubItems("camp-1", []campaignapp.CampaignSession{
		{ID: "sess-active", Name: "Table Night", Status: "active", StartedAt: "2026-03-12 18:00 UTC"},
		{ID: "sess-unnamed", Name: "sess-unnamed", Status: "active"},
		{ID: "sess-ended", Name: "Finished", Status: "ended", StartedAt: "2026-03-11 18:00 UTC"},
	}, loc)

	if len(items) != 2 {
		t.Fatalf("len(campaignSessionMenuSubItems) = %d, want 2", len(items))
	}
	if items[0].Label != "Table Night" {
		t.Fatalf("items[0].Label = %q, want Table Night", items[0].Label)
	}
	if items[0].StartDetail != "Start: 2026-03-12 18:00 UTC" {
		t.Fatalf("items[0].StartDetail = %q", items[0].StartDetail)
	}
	if items[1].Label != "Unnamed" {
		t.Fatalf("items[1].Label = %q, want Unnamed", items[1].Label)
	}
	if items[1].JoinURL == "" || items[1].JoinLabel != "Join" {
		t.Fatalf("items[1] join data = %+v", items[1])
	}
}

func TestCampaignWorkspaceLocaleFormValueNormalizesAliases(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"pt":                  "pt-BR",
		"PT-BR":               "pt-BR",
		"Portuguese (Brazil)": "pt-BR",
		"en-US":               "en-US",
		"":                    "en-US",
	}
	for input, want := range tests {
		if got := campaignWorkspaceLocaleFormValue(input); got != want {
			t.Fatalf("campaignWorkspaceLocaleFormValue(%q) = %q, want %q", input, got, want)
		}
	}
}
