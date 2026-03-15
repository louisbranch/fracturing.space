package notifications

import (
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
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
func (h handlers) notificationListView(items []notificationsapp.NotificationSummary, loc Localizer) []NotificationListItemView {
	if len(items) == 0 {
		return nil
	}
	rows := make([]NotificationListItemView, 0, len(items))
	for _, item := range items {
		itemID := strings.TrimSpace(item.ID)
		if itemID == "" {
			continue
		}
		rendered := h.copyRenderer().RenderInApp(loc, item)
		rows = append(rows, NotificationListItemView{
			ID:           itemID,
			IconID:       notificationMessageIconID(item.MessageType),
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
func (h handlers) notificationDetailView(item notificationsapp.NotificationSummary, loc Localizer) *NotificationDetailView {
	itemID := strings.TrimSpace(item.ID)
	if itemID == "" {
		return nil
	}
	rendered := h.copyRenderer().RenderInApp(loc, item)
	return &NotificationDetailView{
		ID:           itemID,
		IconID:       notificationMessageIconID(item.MessageType),
		Title:        rendered.Title,
		Body:         rendered.Body,
		Facts:        rendered.Facts,
		Actions:      rendered.Actions,
		SourceLabel:  notificationSourceLabel(item.Source, loc),
		CreatedLabel: notificationCreatedLabel(item.CreatedAt, h.now(), loc),
		Read:         item.Read,
		OpenURL:      routepath.AppNotificationOpen(itemID),
		DetailURL:    routepath.AppNotification(itemID),
	}
}

// notificationTitle centralizes this web behavior in one helper seam.
func notificationTitle(value platformi18n.CopyRef, loc Localizer) string {
	resolved := strings.TrimSpace(platformi18n.ResolveCopy(loc, value))
	if resolved == "" {
		return T(loc, "game.notifications.topic_unknown")
	}
	return resolved
}

// notificationBody centralizes this web behavior in one helper seam.
func notificationBody(value platformi18n.CopyRef, loc Localizer) string {
	resolved := strings.TrimSpace(platformi18n.ResolveCopy(loc, value))
	if resolved == "" {
		return T(loc, "game.notifications.detail_empty")
	}
	return resolved
}

// notificationSourceLabel centralizes this web behavior in one helper seam.
func notificationSourceLabel(source string, loc Localizer) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == notificationsgateway.NotificationSourceSystem {
		return T(loc, "game.notifications.source_system")
	}
	return T(loc, "game.notifications.source_unknown")
}

// notificationCreatedLabel centralizes this web behavior in one helper seam.
func notificationCreatedLabel(createdAt time.Time, now time.Time, loc Localizer) string {
	if createdAt.IsZero() {
		return T(loc, "game.notifications.time.just_now")
	}
	delta := now.Sub(createdAt.UTC())
	if delta < 0 {
		delta = 0
	}
	if delta < time.Minute {
		return T(loc, "game.notifications.time.just_now")
	}
	if delta < time.Hour {
		minutes := int(delta / time.Minute)
		if minutes <= 1 {
			return T(loc, "game.notifications.time.minute_ago")
		}
		return T(loc, "game.notifications.time.minutes_ago", minutes)
	}
	if delta < 24*time.Hour {
		hours := int(delta / time.Hour)
		if hours <= 1 {
			return T(loc, "game.notifications.time.hour_ago")
		}
		return T(loc, "game.notifications.time.hours_ago", hours)
	}
	days := int(delta / (24 * time.Hour))
	if days <= 1 {
		return T(loc, "game.notifications.time.day_ago")
	}
	return T(loc, "game.notifications.time.days_ago", days)
}

// notificationMessageIconID maps notification message types to shared content icons.
func notificationMessageIconID(messageType string) commonv1.IconId {
	messageType = strings.ToLower(strings.TrimSpace(messageType))
	switch {
	case strings.HasPrefix(messageType, "campaign.invite."):
		return commonv1.IconId_ICON_ID_INVITES
	case strings.HasPrefix(messageType, "campaign."):
		return commonv1.IconId_ICON_ID_CAMPAIGN
	case strings.HasPrefix(messageType, "auth.onboarding."):
		return commonv1.IconId_ICON_ID_PROFILE
	case strings.HasPrefix(messageType, "system.message."):
		return commonv1.IconId_ICON_ID_MESSAGE
	default:
		return commonv1.IconId_ICON_ID_MESSAGE
	}
}

// notificationsSideMenu builds the shared app side menu from notification rows.
func notificationsSideMenu(currentPath string, items []NotificationListItemView, loc webtemplates.Localizer) *webtemplates.AppSideMenu {
	if len(items) == 0 {
		return nil
	}
	menuItems := make([]webtemplates.AppSideMenuItem, 0, len(items))
	for _, item := range items {
		detailURL := strings.TrimSpace(item.DetailURL)
		if detailURL == "" {
			continue
		}
		menuItems = append(menuItems, webtemplates.AppSideMenuItem{
			Label:      notificationItemTitle(item, loc),
			URL:        detailURL,
			MatchExact: true,
			IconID:     item.IconID,
		})
	}
	if len(menuItems) == 0 {
		return nil
	}
	return &webtemplates.AppSideMenu{
		CurrentPath: strings.TrimSpace(currentPath),
		Items:       menuItems,
	}
}
