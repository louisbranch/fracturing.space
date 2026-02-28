package notifications

import (
	"testing"
	"time"

	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	"golang.org/x/text/language"
)

func TestNotificationCreatedLabelReturnsExpectedRelativeLabels(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		createdAt time.Time
		want      string
	}{
		{name: "zero time", createdAt: time.Time{}, want: "just now"},
		{name: "30 seconds ago", createdAt: now.Add(-30 * time.Second), want: "just now"},
		{name: "1 minute ago", createdAt: now.Add(-1 * time.Minute), want: "1 minute ago"},
		{name: "5 minutes ago", createdAt: now.Add(-5 * time.Minute), want: "5 minutes ago"},
		{name: "1 hour ago", createdAt: now.Add(-1 * time.Hour), want: "1 hour ago"},
		{name: "3 hours ago", createdAt: now.Add(-3 * time.Hour), want: "3 hours ago"},
		{name: "1 day ago", createdAt: now.Add(-24 * time.Hour), want: "1 day ago"},
		{name: "7 days ago", createdAt: now.Add(-7 * 24 * time.Hour), want: "7 days ago"},
		{name: "future time clamped", createdAt: now.Add(10 * time.Minute), want: "just now"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := notificationCreatedLabel(tc.createdAt, now, loc)
			if got != tc.want {
				t.Fatalf("notificationCreatedLabel(%v, %v) = %q, want %q", tc.createdAt, now, got, tc.want)
			}
		})
	}
}
