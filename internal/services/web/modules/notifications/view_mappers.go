package notifications

import (
	"strings"
	"time"

	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// now returns the current time using the injected nowFunc, defaulting to time.Now
// if the handler was not constructed with newHandlers.
func (h handlers) now() time.Time {
	if h.nowFunc != nil {
		return h.nowFunc()
	}
	return time.Now()
}

// notificationListView centralizes this web behavior in one helper seam.
func (h handlers) notificationListView(items []notificationsapp.NotificationSummary, loc webtemplates.Localizer) []webtemplates.NotificationListItemView {
	if len(items) == 0 {
		return nil
	}
	rows := make([]webtemplates.NotificationListItemView, 0, len(items))
	for _, item := range items {
		itemID := strings.TrimSpace(item.ID)
		if itemID == "" {
			continue
		}
		rendered := h.copyRenderer().RenderInApp(loc, item)
		rows = append(rows, webtemplates.NotificationListItemView{
			ID:           itemID,
			Title:        rendered.Title,
			Body:         rendered.Body,
			SourceLabel:  notificationSourceLabel(item.Source, loc),
			CreatedLabel: notificationCreatedLabel(item.CreatedAt, h.now(), loc),
			Read:         item.Read,
			OpenURL:      routepath.AppNotificationOpen(itemID),
			DetailURL:    routepath.AppNotification(itemID),
		})
	}
	return rows
}

// notificationDetailView centralizes this web behavior in one helper seam.
func (h handlers) notificationDetailView(item notificationsapp.NotificationSummary, loc webtemplates.Localizer) *webtemplates.NotificationDetailView {
	itemID := strings.TrimSpace(item.ID)
	if itemID == "" {
		return nil
	}
	rendered := h.copyRenderer().RenderInApp(loc, item)
	return &webtemplates.NotificationDetailView{
		ID:           itemID,
		Title:        rendered.Title,
		Body:         rendered.Body,
		SourceLabel:  notificationSourceLabel(item.Source, loc),
		CreatedLabel: notificationCreatedLabel(item.CreatedAt, h.now(), loc),
		Read:         item.Read,
		OpenURL:      routepath.AppNotificationOpen(itemID),
		DetailURL:    routepath.AppNotification(itemID),
	}
}

// notificationTitle centralizes this web behavior in one helper seam.
func notificationTitle(value string, loc webtemplates.Localizer) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return webtemplates.T(loc, "game.notifications.topic_unknown")
	}
	return value
}

// notificationBody centralizes this web behavior in one helper seam.
func notificationBody(value string, loc webtemplates.Localizer) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return webtemplates.T(loc, "game.notifications.detail_empty")
	}
	return value
}

// notificationSourceLabel centralizes this web behavior in one helper seam.
func notificationSourceLabel(source string, loc webtemplates.Localizer) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == notificationsgateway.NotificationSourceSystem {
		return webtemplates.T(loc, "game.notifications.source_system")
	}
	return webtemplates.T(loc, "game.notifications.source_unknown")
}

// notificationCreatedLabel centralizes this web behavior in one helper seam.
func notificationCreatedLabel(createdAt time.Time, now time.Time, loc webtemplates.Localizer) string {
	if createdAt.IsZero() {
		return webtemplates.T(loc, "game.notifications.time.just_now")
	}
	delta := now.Sub(createdAt.UTC())
	if delta < 0 {
		delta = 0
	}
	if delta < time.Minute {
		return webtemplates.T(loc, "game.notifications.time.just_now")
	}
	if delta < time.Hour {
		minutes := int(delta / time.Minute)
		if minutes <= 1 {
			return webtemplates.T(loc, "game.notifications.time.minute_ago")
		}
		return webtemplates.T(loc, "game.notifications.time.minutes_ago", minutes)
	}
	if delta < 24*time.Hour {
		hours := int(delta / time.Hour)
		if hours <= 1 {
			return webtemplates.T(loc, "game.notifications.time.hour_ago")
		}
		return webtemplates.T(loc, "game.notifications.time.hours_ago", hours)
	}
	days := int(delta / (24 * time.Hour))
	if days <= 1 {
		return webtemplates.T(loc, "game.notifications.time.day_ago")
	}
	return webtemplates.T(loc, "game.notifications.time.days_ago", days)
}
