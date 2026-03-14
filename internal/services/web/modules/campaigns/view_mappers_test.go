package campaigns

import (
	"fmt"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
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
	items := []campaignapp.CampaignSummary{
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

func TestCampaignWorkspaceMenuShowsOnlyActiveSessionSubItems(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	workspace := campaignapp.CampaignWorkspace{
		ID:               "c1",
		ParticipantCount: "4",
		CharacterCount:   "3",
	}
	sessions := []campaignapp.CampaignSession{
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

	menu := campaignWorkspaceMenu(workspace, routepath.AppCampaignParticipants("c1"), sessions, true, loc)
	if menu == nil {
		t.Fatalf("campaignWorkspaceMenu(...) = nil, want non-nil")
	}
	if len(menu.Items) != 5 {
		t.Fatalf("len(menu.Items) = %d, want 5", len(menu.Items))
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
	if menu.Items[4].Label != "Invites" {
		t.Fatalf("menu.Items[4].Label = %q, want %q", menu.Items[4].Label, "Invites")
	}
	if menu.Items[4].URL != routepath.AppCampaignInvites("c1") {
		t.Fatalf("menu.Items[4].URL = %q, want %q", menu.Items[4].URL, routepath.AppCampaignInvites("c1"))
	}
	if menu.Items[4].IconID != commonv1.IconId_ICON_ID_INVITES {
		t.Fatalf("menu.Items[4].IconID = %v, want %v", menu.Items[4].IconID, commonv1.IconId_ICON_ID_INVITES)
	}
	if menu.Items[4].Badge != "" {
		t.Fatalf("menu.Items[4].Badge = %q, want empty badge", menu.Items[4].Badge)
	}
	// Badge still shows total count of all sessions.
	if menu.Items[3].Badge != "3" {
		t.Fatalf("menu.Items[3].Badge = %q, want %q", menu.Items[3].Badge, "3")
	}
	if menu.Items[3].IconID != commonv1.IconId_ICON_ID_SESSION {
		t.Fatalf("menu.Items[3].IconID = %v, want %v", menu.Items[3].IconID, commonv1.IconId_ICON_ID_SESSION)
	}
	// Only the active session should appear as a sub-item.
	if len(menu.Items[3].SubItems) != 1 {
		t.Fatalf("len(menu.Items[3].SubItems) = %d, want 1", len(menu.Items[3].SubItems))
	}

	active := menu.Items[3].SubItems[0]
	if active.URL != routepath.AppCampaignSession("c1", "s1") {
		t.Fatalf("active subitem URL = %q, want %q", active.URL, routepath.AppCampaignSession("c1", "s1"))
	}
	if active.Label != "Unnamed session" {
		t.Fatalf("active session label = %q, want %q", active.Label, "Unnamed session")
	}
	if !active.ActiveSession {
		t.Fatalf("active.ActiveSession = %v, want true", active.ActiveSession)
	}
	if active.StartDetail != "Start: 2026-02-01 20:00 UTC" {
		t.Fatalf("active.StartDetail = %q, want %q", active.StartDetail, "Start: 2026-02-01 20:00 UTC")
	}
	if active.JoinURL != routepath.AppCampaignGame("c1") {
		t.Fatalf("active.JoinURL = %q, want %q", active.JoinURL, routepath.AppCampaignGame("c1"))
	}
	if active.JoinLabel != "Join Game" {
		t.Fatalf("active.JoinLabel = %q, want %q", active.JoinLabel, "Join Game")
	}
}

func TestCampaignWorkspaceMenuBadgeCountsAllSessionsButSubItemsOnlyActive(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	workspace := campaignapp.CampaignWorkspace{ID: "c1"}

	sessions := make([]campaignapp.CampaignSession, 0, 12)
	for day := 1; day <= 12; day++ {
		sessionID := fmt.Sprintf("s%02d", day)
		start := fmt.Sprintf("2026-02-%02d 20:00 UTC", day)
		end := fmt.Sprintf("2026-02-%02d 22:00 UTC", day)
		sessions = append(sessions, campaignapp.CampaignSession{
			ID:        sessionID,
			Name:      "Session " + sessionID,
			Status:    "Ended",
			StartedAt: start,
			EndedAt:   end,
		})
	}

	menu := campaignWorkspaceMenu(workspace, routepath.AppCampaignParticipants("c1"), sessions, true, loc)
	if menu == nil {
		t.Fatalf("campaignWorkspaceMenu(...) = nil, want non-nil")
	}

	sessionItem := menu.Items[3]
	// Badge still shows total count.
	if sessionItem.Badge != "12" {
		t.Fatalf("session badge = %q, want %q", sessionItem.Badge, "12")
	}
	// No active sessions, so no sub-items.
	if len(sessionItem.SubItems) != 0 {
		t.Fatalf("len(session subitems) = %d, want 0", len(sessionItem.SubItems))
	}
}

func TestCampaignWorkspaceMenuHidesInvitesWithoutPermission(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	workspace := campaignapp.CampaignWorkspace{ID: "c1"}

	menu := campaignWorkspaceMenu(workspace, routepath.AppCampaignParticipants("c1"), nil, false, loc)
	if menu == nil {
		t.Fatalf("campaignWorkspaceMenu(...) = nil, want non-nil")
	}
	if len(menu.Items) != 4 {
		t.Fatalf("len(menu.Items) = %d, want 4", len(menu.Items))
	}
	for _, item := range menu.Items {
		// Invariant: invite-manage navigation must not be exposed without permission.
		if item.URL == routepath.AppCampaignInvites("c1") {
			t.Fatalf("menu should not include invites item: %#v", menu.Items)
		}
	}
}

func TestMapInviteSeatOptionsShowsOnlyAvailableHumanSeats(t *testing.T) {
	t.Parallel()

	options := mapInviteSeatOptions(
		[]campaignapp.CampaignParticipant{
			{ID: "p-open-b", Name: "Bryn", Controller: "Human"},
			{ID: "p-pending", Name: "Ari", Controller: "Human"},
			{ID: "p-bound", Name: "Cato", Controller: "Human", UserID: "user-1"},
			{ID: "p-ai", Name: "Oracle", Controller: "AI"},
			{ID: "p-open-a", Name: "Ada", Controller: "controller_human"},
		},
		[]campaignapp.CampaignInvite{
			{ID: "inv-1", ParticipantID: "p-pending", Status: "Pending"},
			{ID: "inv-2", ParticipantID: "p-open-b", Status: "Claimed"},
		},
	)

	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(options))
	}
	if options[0].ParticipantID != "p-open-a" || options[0].Label != "Ada" {
		t.Fatalf("options[0] = %#v, want participant p-open-a / Ada", options[0])
	}
	if options[1].ParticipantID != "p-open-b" || options[1].Label != "Bryn" {
		t.Fatalf("options[1] = %#v, want participant p-open-b / Bryn", options[1])
	}
}
