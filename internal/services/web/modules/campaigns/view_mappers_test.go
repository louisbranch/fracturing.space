package campaigns

import (
	"testing"
	"time"

	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
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
