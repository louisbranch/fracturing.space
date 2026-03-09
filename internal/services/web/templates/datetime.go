package templates

import (
	"time"
)

// DateTimeDisplay holds pre-computed values for rendering a <time> element with tooltip.
type DateTimeDisplay struct {
	DisplayText string
	TooltipText string
	ISOValue    string
}

const dateTimeTimestampLayout = "2006-01-02 15:04 UTC"
const dateTimeAbsoluteThreshold = 7 * 24 * time.Hour

// RelativeTimeLabel converts a duration into a localized relative label
// (e.g. "just now", "3 hours ago"). Extracted from campaignListItemUpdatedAt.
func RelativeTimeLabel(delta time.Duration, loc Localizer) string {
	if delta < 0 {
		delta = 0
	}
	switch {
	case delta < time.Minute:
		return T(loc, "game.notifications.time.just_now")
	case delta < time.Hour:
		minutes := int(delta / time.Minute)
		if minutes <= 1 {
			return T(loc, "game.notifications.time.minute_ago")
		}
		return T(loc, "game.notifications.time.minutes_ago", minutes)
	case delta < 24*time.Hour:
		hours := int(delta / time.Hour)
		if hours <= 1 {
			return T(loc, "game.notifications.time.hour_ago")
		}
		return T(loc, "game.notifications.time.hours_ago", hours)
	default:
		days := int(delta / (24 * time.Hour))
		if days <= 1 {
			return T(loc, "game.notifications.time.day_ago")
		}
		return T(loc, "game.notifications.time.days_ago", days)
	}
}

// FormatDateTimeDisplay builds a DateTimeDisplay from a timestamp string and reference time.
// For timestamps < 7 days old, it uses a relative label; older timestamps show absolute UTC
// as server-rendered fallback text with a data attribute for client-side local time conversion.
func FormatDateTimeDisplay(timestampStr string, now time.Time, loc Localizer) DateTimeDisplay {
	if timestampStr == "" {
		return DateTimeDisplay{}
	}
	parsed, err := time.Parse(dateTimeTimestampLayout, timestampStr)
	if err != nil {
		return DateTimeDisplay{}
	}
	parsed = parsed.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	delta := now.Sub(parsed)
	if delta < 0 {
		delta = 0
	}

	tooltip := parsed.Format(dateTimeTimestampLayout)
	iso := parsed.Format(time.RFC3339)

	if delta < dateTimeAbsoluteThreshold {
		return DateTimeDisplay{
			DisplayText: RelativeTimeLabel(delta, loc),
			TooltipText: tooltip,
			ISOValue:    iso,
		}
	}
	return DateTimeDisplay{
		DisplayText: parsed.Format("Jan 2, 2006 15:04 UTC"),
		TooltipText: tooltip,
		ISOValue:    iso,
	}
}

// FormatDateTimeNow is a convenience wrapper using time.Now().UTC().
func FormatDateTimeNow(timestampStr string, loc Localizer) DateTimeDisplay {
	return FormatDateTimeDisplay(timestampStr, time.Now().UTC(), loc)
}
