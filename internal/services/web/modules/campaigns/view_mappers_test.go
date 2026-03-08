package campaigns

import (
	"fmt"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"golang.org/x/text/language"
)

func TestCampaignListItemUpdatedAtReturnsExpectedRelativeLabels(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		updatedAt time.Time
		want      string
	}{
		{name: "zero time", updatedAt: time.Time{}, want: "Updated just now"},
		{name: "30 seconds ago", updatedAt: now.Add(-30 * time.Second), want: "Updated just now"},
		{name: "1 minute ago", updatedAt: now.Add(-1 * time.Minute), want: "Updated 1 minute ago"},
		{name: "5 minutes ago", updatedAt: now.Add(-5 * time.Minute), want: "Updated 5 minutes ago"},
		{name: "1 hour ago", updatedAt: now.Add(-1 * time.Hour), want: "Updated 1 hour ago"},
		{name: "3 hours ago", updatedAt: now.Add(-3 * time.Hour), want: "Updated 3 hours ago"},
		{name: "1 day ago", updatedAt: now.Add(-24 * time.Hour), want: "Updated 1 day ago"},
		{name: "7 days ago", updatedAt: now.Add(-7 * 24 * time.Hour), want: "Updated 7 days ago"},
		{name: "future time clamped", updatedAt: now.Add(10 * time.Minute), want: "Updated just now"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := campaignListItemUpdatedAt(tc.updatedAt.UnixNano(), now, loc)
			if got != tc.want {
				t.Fatalf("campaignListItemUpdatedAt(%v, %v) = %q, want %q", tc.updatedAt, now, got, tc.want)
			}
		})
	}
}

func TestMapCampaignListItemsIncludesUpdatedAtLabel(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	items := []CampaignSummary{
		{
			ID:                "c-1",
			Name:              "Campaign 1",
			UpdatedAtUnixNano: now.Add(-2 * time.Minute).UnixNano(),
		},
	}
	mapped := mapCampaignListItems(items, now, loc)
	if len(mapped) != 1 {
		t.Fatalf("len(mapped) = %d, want 1", len(mapped))
	}
	if mapped[0].UpdatedAt != "Updated 2 minutes ago" {
		t.Fatalf("mapped[0].UpdatedAt = %q, want %q", mapped[0].UpdatedAt, "Updated 2 minutes ago")
	}
}

func TestCampaignWorkspaceMenuIncludesSessionSubItemsOldestToNewest(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	workspace := CampaignWorkspace{
		ID:               "c1",
		ParticipantCount: "4",
		CharacterCount:   "3",
	}
	sessions := []CampaignSession{
		{
			ID:        "s3",
			Name:      "Third Light",
			Status:    "Ended",
			StartedAt: "2026-02-03 20:00 UTC",
			EndedAt:   "2026-02-03 22:00 UTC",
		},
		{
			ID:        "s1",
			Name:      "",
			Status:    "Active",
			StartedAt: "2026-02-01 20:00 UTC",
			EndedAt:   "",
		},
		{
			ID:        "s2",
			Name:      "Second Light",
			Status:    "Ended",
			StartedAt: "2026-02-02 20:00 UTC",
			EndedAt:   "2026-02-02 22:00 UTC",
		},
	}

	menu := campaignWorkspaceMenu(workspace, routepath.AppCampaignParticipants("c1"), sessions, loc)
	if menu == nil {
		t.Fatalf("campaignWorkspaceMenu(...) = nil, want non-nil")
	}
	if len(menu.Items) != 4 {
		t.Fatalf("len(menu.Items) = %d, want 4", len(menu.Items))
	}
	if menu.Items[1].Label != "Participants" {
		t.Fatalf("menu.Items[1].Label = %q, want %q", menu.Items[1].Label, "Participants")
	}
	if menu.Items[2].Label != "Characters" {
		t.Fatalf("menu.Items[2].Label = %q, want %q", menu.Items[2].Label, "Characters")
	}
	if menu.Items[3].Label != "Sessions" {
		t.Fatalf("menu.Items[3].Label = %q, want %q", menu.Items[3].Label, "Sessions")
	}
	if menu.Items[3].URL != routepath.AppCampaignSessions("c1") {
		t.Fatalf("menu.Items[3].URL = %q, want %q", menu.Items[3].URL, routepath.AppCampaignSessions("c1"))
	}
	if menu.Items[3].Badge != "3" {
		t.Fatalf("menu.Items[3].Badge = %q, want %q", menu.Items[3].Badge, "3")
	}
	if menu.Items[3].IconID != commonv1.IconId_ICON_ID_SESSION {
		t.Fatalf("menu.Items[3].IconID = %v, want %v", menu.Items[3].IconID, commonv1.IconId_ICON_ID_SESSION)
	}
	if len(menu.Items[3].SubItems) != 3 {
		t.Fatalf("len(menu.Items[3].SubItems) = %d, want 3", len(menu.Items[3].SubItems))
	}

	gotOrder := []string{
		menu.Items[3].SubItems[0].URL,
		menu.Items[3].SubItems[1].URL,
		menu.Items[3].SubItems[2].URL,
	}
	wantOrder := []string{
		routepath.AppCampaignSession("c1", "s1"),
		routepath.AppCampaignSession("c1", "s2"),
		routepath.AppCampaignSession("c1", "s3"),
	}
	for idx := range wantOrder {
		if gotOrder[idx] != wantOrder[idx] {
			t.Fatalf("menu session order[%d] = %q, want %q", idx, gotOrder[idx], wantOrder[idx])
		}
	}

	first := menu.Items[3].SubItems[0]
	if first.Label != "Unnamed session" {
		t.Fatalf("first session label = %q, want %q", first.Label, "Unnamed session")
	}
	if !first.ActiveSession {
		t.Fatalf("first.ActiveSession = %v, want true", first.ActiveSession)
	}
	if first.StartDetail != "Start: 2026-02-01 20:00 UTC" {
		t.Fatalf("first.StartDetail = %q, want %q", first.StartDetail, "Start: 2026-02-01 20:00 UTC")
	}
	if first.EndDetail != "End: In progress" {
		t.Fatalf("first.EndDetail = %q, want %q", first.EndDetail, "End: In progress")
	}
}

func TestCampaignWorkspaceMenuLimitsSessionSubItemsToMostRecentTen(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	workspace := CampaignWorkspace{ID: "c1"}

	sessions := make([]CampaignSession, 0, 12)
	for day := 1; day <= 12; day++ {
		sessionID := fmt.Sprintf("s%02d", day)
		start := fmt.Sprintf("2026-02-%02d 20:00 UTC", day)
		end := fmt.Sprintf("2026-02-%02d 22:00 UTC", day)
		sessions = append(sessions, CampaignSession{
			ID:        sessionID,
			Name:      "Session " + sessionID,
			Status:    "Ended",
			StartedAt: start,
			EndedAt:   end,
		})
	}

	menu := campaignWorkspaceMenu(workspace, routepath.AppCampaignParticipants("c1"), sessions, loc)
	if menu == nil {
		t.Fatalf("campaignWorkspaceMenu(...) = nil, want non-nil")
	}

	sessionItem := menu.Items[3]
	if sessionItem.Badge != "12" {
		t.Fatalf("session badge = %q, want %q", sessionItem.Badge, "12")
	}
	if len(sessionItem.SubItems) != 10 {
		t.Fatalf("len(session subitems) = %d, want 10", len(sessionItem.SubItems))
	}

	if got := sessionItem.SubItems[0].URL; got != routepath.AppCampaignSession("c1", "s03") {
		t.Fatalf("first subitem URL = %q, want %q", got, routepath.AppCampaignSession("c1", "s03"))
	}
	if got := sessionItem.SubItems[9].URL; got != routepath.AppCampaignSession("c1", "s12") {
		t.Fatalf("last subitem URL = %q, want %q", got, routepath.AppCampaignSession("c1", "s12"))
	}
}
