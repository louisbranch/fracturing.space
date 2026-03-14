package notifications

import (
	"encoding/json"
	"strings"
	"time"

	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
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
func notificationTitle(value string, loc Localizer) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return T(loc, "game.notifications.topic_unknown")
	}
	return value
}

// notificationBody centralizes this web behavior in one helper seam.
func notificationBody(value string, loc Localizer) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return T(loc, "game.notifications.detail_empty")
	}
	return value
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

// inviteNotificationPayload captures the routing fields used for invite deep links.
type inviteNotificationPayload struct {
	InviteID   string `json:"invite_id"`
	CampaignID string `json:"campaign_id"`
}

// notificationPrimaryActionURL derives deep links for notifications that should
// open directly into an invite action surface instead of a generic detail page.
func notificationPrimaryActionURL(item notificationsapp.NotificationSummary) string {
	switch strings.ToLower(strings.TrimSpace(item.MessageType)) {
	case "campaign.invite.created.v1":
		payload, ok := parseInviteNotificationPayload(item.PayloadJSON)
		if !ok || payload.InviteID == "" {
			return ""
		}
		return routepath.PublicInvite(payload.InviteID)
	case "campaign.invite.accepted.v1", "campaign.invite.declined.v1":
		payload, ok := parseInviteNotificationPayload(item.PayloadJSON)
		if !ok || payload.CampaignID == "" {
			return ""
		}
		return routepath.AppCampaign(payload.CampaignID)
	default:
		return ""
	}
}

// parseInviteNotificationPayload normalizes invite notification routing payloads.
func parseInviteNotificationPayload(raw string) (inviteNotificationPayload, bool) {
	var payload inviteNotificationPayload
	if strings.TrimSpace(raw) == "" {
		return inviteNotificationPayload{}, false
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return inviteNotificationPayload{}, false
	}
	payload.InviteID = strings.TrimSpace(payload.InviteID)
	payload.CampaignID = strings.TrimSpace(payload.CampaignID)
	return payload, payload.InviteID != "" || payload.CampaignID != ""
}
