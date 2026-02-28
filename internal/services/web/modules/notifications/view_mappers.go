package notifications

import (
	"strings"
	"time"

	notificationsrender "github.com/louisbranch/fracturing.space/internal/services/notifications/render"
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

func (h handlers) notificationListView(items []NotificationSummary, loc webtemplates.Localizer) []webtemplates.NotificationListItemView {
	if len(items) == 0 {
		return nil
	}
	rows := make([]webtemplates.NotificationListItemView, 0, len(items))
	for _, item := range items {
		itemID := strings.TrimSpace(item.ID)
		if itemID == "" {
			continue
		}
		rendered := notificationsrender.Render(loc, notificationsrender.Input{
			MessageType: item.MessageType,
			PayloadJSON: item.PayloadJSON,
			Channel:     notificationsrender.ChannelInApp,
		})
		rows = append(rows, webtemplates.NotificationListItemView{
			ID:           itemID,
			Title:        notificationTitle(rendered.Title, loc),
			Body:         notificationBody(rendered.BodyText, loc),
			SourceLabel:  notificationSourceLabel(item.Source, loc),
			CreatedLabel: notificationCreatedLabel(item.CreatedAt, h.now(), loc),
			Read:         item.Read,
			OpenURL:      routepath.AppNotificationOpen(itemID),
			DetailURL:    routepath.AppNotification(itemID),
		})
	}
	return rows
}

func (h handlers) notificationDetailView(item NotificationSummary, loc webtemplates.Localizer) *webtemplates.NotificationDetailView {
	itemID := strings.TrimSpace(item.ID)
	if itemID == "" {
		return nil
	}
	rendered := notificationsrender.Render(loc, notificationsrender.Input{
		MessageType: item.MessageType,
		PayloadJSON: item.PayloadJSON,
		Channel:     notificationsrender.ChannelInApp,
	})
	return &webtemplates.NotificationDetailView{
		ID:           itemID,
		Title:        notificationTitle(rendered.Title, loc),
		Body:         notificationBody(rendered.BodyText, loc),
		SourceLabel:  notificationSourceLabel(item.Source, loc),
		CreatedLabel: notificationCreatedLabel(item.CreatedAt, h.now(), loc),
		Read:         item.Read,
		OpenURL:      routepath.AppNotificationOpen(itemID),
		DetailURL:    routepath.AppNotification(itemID),
	}
}

func notificationTitle(value string, loc webtemplates.Localizer) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return webtemplates.T(loc, "game.notifications.topic_unknown")
	}
	return value
}

func notificationBody(value string, loc webtemplates.Localizer) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return webtemplates.T(loc, "game.notifications.detail_empty")
	}
	return value
}

func notificationSourceLabel(source string, loc webtemplates.Localizer) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == notificationSourceSystem {
		return webtemplates.T(loc, "game.notifications.source_system")
	}
	return webtemplates.T(loc, "game.notifications.source_unknown")
}

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
