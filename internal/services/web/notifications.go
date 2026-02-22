package web

import (
	"net/http"
	"sort"
	"strings"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const webNotificationsPageSize = 50

var notificationsNow = func() time.Time {
	return time.Now().UTC()
}

func (h *handler) handleAppNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	readCtx, _, ok := h.campaignReadContext(w, r, "Notifications unavailable")
	if !ok {
		return
	}
	if h.notificationClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Notifications unavailable", "notification service client is not configured")
		return
	}

	resp, err := h.notificationClient.ListNotifications(readCtx, &notificationsv1.ListNotificationsRequest{
		PageSize: webNotificationsPageSize,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Notifications unavailable", "failed to list notifications")
		return
	}

	if sess := sessionFromRequest(r, h.sessions); sess != nil {
		sess.setCachedUnreadNotifications(hasUnreadNotifications(resp.GetNotifications()))
	}

	renderAppNotificationsPage(w, r, h.pageContext(w, r), resp.GetNotifications(), notificationsNow())
}

func (h *handler) handleAppNotificationsRoutes(w http.ResponseWriter, r *http.Request) {
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	notificationID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/notifications/"))
	if notificationID == "" || strings.Contains(notificationID, "/") {
		http.NotFound(w, r)
		return
	}

	h.handleAppNotificationOpen(w, r, notificationID)
}

func (h *handler) handleAppNotificationOpen(w http.ResponseWriter, r *http.Request, notificationID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if !isSafeNotificationPathID(notificationID) {
		http.NotFound(w, r)
		return
	}
	readCtx, _, ok := h.campaignReadContext(w, r, "Notification unavailable")
	if !ok {
		return
	}
	if h.notificationClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Notification unavailable", "notification service client is not configured")
		return
	}

	_, err := h.notificationClient.MarkNotificationRead(readCtx, &notificationsv1.MarkNotificationReadRequest{
		NotificationId: notificationID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Notification unavailable", "failed to mark notification read")
		return
	}
	if sess := sessionFromRequest(r, h.sessions); sess != nil {
		sess.clearCachedUnreadNotifications()
	}

	http.Redirect(w, r, "/notifications", http.StatusFound)
}

func renderAppNotificationsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, notifications []*notificationsv1.Notification, now time.Time) {
	items := toNotificationListItems(page.Loc, notifications, now)
	if err := writePage(w, r, webtemplates.NotificationsPage(page, items), composeHTMXTitleForPage(page, "game.notifications.title")); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func toNotificationListItems(loc webtemplates.Localizer, notifications []*notificationsv1.Notification, now time.Time) []webtemplates.NotificationListItem {
	now = now.UTC()
	type sortableItem struct {
		webtemplates.NotificationListItem
		createdAt time.Time
	}

	items := make([]sortableItem, 0, len(notifications))
	for _, notification := range notifications {
		if notification == nil {
			continue
		}

		createdAt := toSafeProtoTime(notification.GetCreatedAt())
		if createdAt.IsZero() {
			createdAt = now
		}
		id := strings.TrimSpace(notification.GetId())
		topic := strings.TrimSpace(notification.GetTopic())
		if topic == "" {
			topic = webtemplates.T(loc, "game.notifications.topic_unknown")
		}
		source := strings.TrimSpace(notification.GetSource())
		if source == "" {
			source = webtemplates.T(loc, "game.notifications.source_unknown")
		}

		items = append(items, sortableItem{
			NotificationListItem: webtemplates.NotificationListItem{
				ID:            id,
				CanOpen:       isSafeNotificationPathID(id),
				Topic:         topic,
				Source:        source,
				CreatedAtISO:  createdAt.Format(time.RFC3339),
				CreatedAtText: formatNotificationRelativeTime(loc, now, createdAt),
				Unread:        notification.GetReadAt() == nil,
			},
			createdAt: createdAt,
		})
	}

	sort.SliceStable(items, func(i int, j int) bool {
		if items[i].createdAt.Equal(items[j].createdAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].createdAt.After(items[j].createdAt)
	})

	normalized := make([]webtemplates.NotificationListItem, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, item.NotificationListItem)
	}
	return normalized
}

func toSafeProtoTime(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	if err := value.CheckValid(); err != nil {
		return time.Time{}
	}
	return value.AsTime().UTC()
}

func formatNotificationRelativeTime(loc webtemplates.Localizer, now time.Time, createdAt time.Time) string {
	if createdAt.IsZero() {
		return webtemplates.T(loc, "game.notifications.time.just_now")
	}
	delta := now.Sub(createdAt)
	if delta < 0 {
		return webtemplates.T(loc, "game.notifications.time.just_now")
	}
	if delta < time.Minute {
		return webtemplates.T(loc, "game.notifications.time.just_now")
	}
	if delta < time.Hour {
		minutes := int(delta / time.Minute)
		if minutes == 1 {
			return webtemplates.T(loc, "game.notifications.time.minute_ago")
		}
		return webtemplates.T(loc, "game.notifications.time.minutes_ago", minutes)
	}
	if delta < 24*time.Hour {
		hours := int(delta / time.Hour)
		if hours == 1 {
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

func isSafeNotificationPathID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return !strings.Contains(value, "/") && !strings.Contains(value, "\\")
}
