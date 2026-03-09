package templates

import (
	"testing"
	"time"

	webi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"golang.org/x/text/language"
)

func TestRelativeTimeLabelReturnsExpectedLabels(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)

	tests := []struct {
		name  string
		delta time.Duration
		want  string
	}{
		{name: "zero", delta: 0, want: "just now"},
		{name: "negative clamped", delta: -5 * time.Minute, want: "just now"},
		{name: "30 seconds", delta: 30 * time.Second, want: "just now"},
		{name: "1 minute", delta: 1 * time.Minute, want: "1 minute ago"},
		{name: "5 minutes", delta: 5 * time.Minute, want: "5 minutes ago"},
		{name: "1 hour", delta: 1 * time.Hour, want: "1 hour ago"},
		{name: "3 hours", delta: 3 * time.Hour, want: "3 hours ago"},
		{name: "1 day", delta: 24 * time.Hour, want: "1 day ago"},
		{name: "7 days", delta: 7 * 24 * time.Hour, want: "7 days ago"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := RelativeTimeLabel(tc.delta, loc)
			if got != tc.want {
				t.Fatalf("RelativeTimeLabel(%v) = %q, want %q", tc.delta, got, tc.want)
			}
		})
	}
}

func TestFormatDateTimeDisplayReturnsRelativeForRecentTimestamps(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	display := FormatDateTimeDisplay("2026-03-08 09:00 UTC", now, loc)
	if display.DisplayText != "3 hours ago" {
		t.Fatalf("DisplayText = %q, want %q", display.DisplayText, "3 hours ago")
	}
	if display.TooltipText != "2026-03-08 09:00 UTC" {
		t.Fatalf("TooltipText = %q, want %q", display.TooltipText, "2026-03-08 09:00 UTC")
	}
	if display.ISOValue != "2026-03-08T09:00:00Z" {
		t.Fatalf("ISOValue = %q, want %q", display.ISOValue, "2026-03-08T09:00:00Z")
	}
}

func TestFormatDateTimeDisplayReturnsAbsoluteForOldTimestamps(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	display := FormatDateTimeDisplay("2026-02-01 20:00 UTC", now, loc)
	if display.DisplayText != "Feb 1, 2026 20:00 UTC" {
		t.Fatalf("DisplayText = %q, want %q", display.DisplayText, "Feb 1, 2026 20:00 UTC")
	}
	if display.ISOValue != "2026-02-01T20:00:00Z" {
		t.Fatalf("ISOValue = %q, want %q", display.ISOValue, "2026-02-01T20:00:00Z")
	}
}

func TestFormatDateTimeDisplayReturnsEmptyForInvalidInput(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "bad format", input: "not-a-date"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			display := FormatDateTimeDisplay(tc.input, now, loc)
			if display.DisplayText != "" {
				t.Fatalf("DisplayText = %q, want empty", display.DisplayText)
			}
		})
	}
}

func TestFormatDateTimeDisplayClampsNegativeDelta(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	// Timestamp in the future relative to now.
	display := FormatDateTimeDisplay("2026-03-08 13:00 UTC", now, loc)
	if display.DisplayText != "just now" {
		t.Fatalf("DisplayText = %q, want %q", display.DisplayText, "just now")
	}
}

func TestFormatDateTimeDisplayUsesCurrentTimeWhenNowIsZero(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)

	// Pass zero time — function should use time.Now() internally.
	display := FormatDateTimeDisplay("2020-01-01 00:00 UTC", time.Time{}, loc)
	// Timestamp is years in the past, so it should use absolute format.
	if display.DisplayText != "Jan 1, 2020 00:00 UTC" {
		t.Fatalf("DisplayText = %q, want %q", display.DisplayText, "Jan 1, 2020 00:00 UTC")
	}
}

func TestFormatDateTimeNowReturnsNonEmptyForValidTimestamp(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)

	// Use a timestamp far in the past so the result is deterministic (absolute format).
	display := FormatDateTimeNow("2020-06-15 10:30 UTC", loc)
	if display.DisplayText == "" {
		t.Fatal("FormatDateTimeNow returned empty DisplayText for valid timestamp")
	}
	if display.ISOValue != "2020-06-15T10:30:00Z" {
		t.Fatalf("ISOValue = %q, want %q", display.ISOValue, "2020-06-15T10:30:00Z")
	}
}
