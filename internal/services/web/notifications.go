package web

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	notificationsrender "github.com/louisbranch/fracturing.space/internal/services/notifications/render"
	notificationsmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/notifications"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const webNotificationsPageSize = 50

const (
	notificationsFilterUnread = "unread"
	notificationsFilterAll    = "all"
)

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

	sess := sessionFromRequest(r, h.sessions)
	if sess != nil {
		sess.setCachedUnreadNotifications(hasUnreadNotifications(resp.GetNotifications()))
	}

	page := h.pageContext(w, r)
	query := notificationPageQueryFromRequest(r)
	items := toNotificationListItems(page.Loc, resp.GetNotifications(), notificationsNow())
	state := toNotificationsPageState(page.Loc, items, query)
	if renderedUnreadID := renderedUnreadNotificationID(state.Items); renderedUnreadID != "" {
		if h.markNotificationRead(readCtx, renderedUnreadID) && sess != nil {
			sess.clearCachedUnreadNotifications()
		}
	}
	renderAppNotificationsPage(w, r, page, state)
}

func (h *handler) handleAppNotificationsRoutes(w http.ResponseWriter, r *http.Request) {
	notificationsmodule.HandleNotificationSubpath(w, r, notificationsmodule.NewService(notificationsmodule.Handlers{
		Notifications:    h.handleAppNotifications,
		NotificationOpen: h.handleAppNotificationOpen,
	}))
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

	redirectLocation := notificationsListURL(notificationPageQuery{
		filter:     normalizeNotificationsFilter(r.URL.Query().Get("filter")),
		selectedID: notificationID,
	})
	http.Redirect(w, r, redirectLocation, http.StatusFound)
}

type notificationPageQuery struct {
	filter     string
	selectedID string
}

func notificationPageQueryFromRequest(r *http.Request) notificationPageQuery {
	if r == nil || r.URL == nil {
		return notificationPageQuery{filter: notificationsFilterUnread}
	}
	query := r.URL.Query()
	return notificationPageQuery{
		filter:     normalizeNotificationsFilter(query.Get("filter")),
		selectedID: normalizedNotificationPathID(query.Get("selected")),
	}
}

func renderAppNotificationsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, state webtemplates.NotificationsPageState) {
	if err := writePage(w, r, webtemplates.NotificationsPage(page, state), composeHTMXTitleForPage(page, "game.notifications.title")); err != nil {
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
		rendered := notificationsrender.Render(loc, notificationsrender.Input{
			Topic:       notification.GetTopic(),
			PayloadJSON: strings.TrimSpace(notification.GetPayloadJson()),
			Channel:     notificationsrender.ChannelInApp,
		})

		topic := strings.TrimSpace(rendered.Title)
		if topic == "" {
			topic = webtemplates.T(loc, "game.notifications.topic_unknown")
		}
		source := notificationSourceLabel(loc, notification.GetSource())

		items = append(items, sortableItem{
			NotificationListItem: webtemplates.NotificationListItem{
				ID:                id,
				CanOpen:           isSafeNotificationPathID(id),
				Topic:             topic,
				Source:            source,
				CreatedAtISO:      createdAt.Format(time.RFC3339),
				CreatedAtText:     formatNotificationRelativeTime(loc, now, createdAt),
				CreatedAtAbsolute: formatNotificationAbsoluteTime(createdAt),
				BodyText:          strings.TrimSpace(rendered.BodyText),
				Unread:            notification.GetReadAt() == nil,
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

func toNotificationsPageState(loc webtemplates.Localizer, items []webtemplates.NotificationListItem, query notificationPageQuery) webtemplates.NotificationsPageState {
	filter := normalizeNotificationsFilter(query.filter)
	filteredItems := filterNotificationItems(items, filter)

	unreadCount := 0
	for _, item := range items {
		if item.Unread {
			unreadCount++
		}
	}

	state := webtemplates.NotificationsPageState{
		Filter:      filter,
		UnreadCount: unreadCount,
		AllCount:    len(items),
		Items:       make([]webtemplates.NotificationListItem, 0, len(filteredItems)),
	}

	selectedID := normalizedNotificationPathID(query.selectedID)
	selectedIndex := -1
	if selectedID != "" {
		for i, item := range filteredItems {
			if item.ID == selectedID {
				selectedIndex = i
				break
			}
		}
	}
	if selectedIndex < 0 && len(filteredItems) > 0 {
		selectedIndex = 0
	}

	for i, item := range filteredItems {
		item.Selected = i == selectedIndex
		item.ActionURL = notificationActionURL(item.ID, filter)
		state.Items = append(state.Items, item)
	}

	if selectedIndex >= 0 && selectedIndex < len(filteredItems) {
		selected := filteredItems[selectedIndex]
		detailBody := notificationDetailText(loc, selected.BodyText)
		state.Detail = webtemplates.NotificationDetail{
			HasSelection:      true,
			Topic:             selected.Topic,
			Source:            selected.Source,
			CreatedAtISO:      selected.CreatedAtISO,
			CreatedAtText:     selected.CreatedAtText,
			CreatedAtAbsolute: selected.CreatedAtAbsolute,
			Body:              detailBody,
			HasBody:           strings.TrimSpace(detailBody) != "",
			Unread:            selected.Unread,
		}
	}

	return state
}

func renderedUnreadNotificationID(items []webtemplates.NotificationListItem) string {
	for _, item := range items {
		if !item.Selected {
			continue
		}
		if !item.Unread {
			return ""
		}
		return normalizedNotificationPathID(item.ID)
	}
	return ""
}

func normalizeNotificationsFilter(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case notificationsFilterAll:
		return notificationsFilterAll
	case notificationsFilterUnread:
		return notificationsFilterUnread
	default:
		return notificationsFilterUnread
	}
}

func filterNotificationItems(items []webtemplates.NotificationListItem, filter string) []webtemplates.NotificationListItem {
	if filter == notificationsFilterAll {
		filtered := make([]webtemplates.NotificationListItem, 0, len(items))
		filtered = append(filtered, items...)
		return filtered
	}
	filtered := make([]webtemplates.NotificationListItem, 0, len(items))
	for _, item := range items {
		if item.Unread {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func notificationActionURL(notificationID string, filter string) string {
	notificationID = normalizedNotificationPathID(notificationID)
	if notificationID == "" {
		return routepath.AppNotifications
	}
	values := url.Values{}
	values.Set("filter", normalizeNotificationsFilter(filter))
	values.Set("selected", notificationID)
	return routepath.AppNotifications + "?" + values.Encode()
}

func notificationsListURL(query notificationPageQuery) string {
	values := url.Values{}
	values.Set("filter", normalizeNotificationsFilter(query.filter))
	selectedID := normalizedNotificationPathID(query.selectedID)
	if selectedID != "" {
		values.Set("selected", selectedID)
	}
	return routepath.AppNotifications + "?" + values.Encode()
}

func notificationDetailText(loc webtemplates.Localizer, bodyText string) string {
	bodyText = strings.TrimSpace(bodyText)
	if bodyText == "" {
		return webtemplates.T(loc, "game.notifications.detail_empty")
	}
	return bodyText
}

func formatNotificationAbsoluteTime(createdAt time.Time) string {
	if createdAt.IsZero() {
		return "-"
	}
	return createdAt.UTC().Format("2006-01-02 15:04 UTC")
}

func (h *handler) markNotificationRead(ctx context.Context, notificationID string) bool {
	if h == nil || h.notificationClient == nil {
		return false
	}
	notificationID = normalizedNotificationPathID(notificationID)
	if notificationID == "" {
		return false
	}

	_, err := h.notificationClient.MarkNotificationRead(ctx, &notificationsv1.MarkNotificationReadRequest{
		NotificationId: notificationID,
	})
	if err != nil {
		log.Printf("web: mark notification read failed: notification_id=%s err=%v", notificationID, err)
		return false
	}
	return true
}

func notificationSourceLabel(loc webtemplates.Localizer, source notificationsv1.NotificationSource) string {
	switch source {
	case notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM:
		return webtemplates.T(loc, "game.notifications.source_system")
	default:
		return webtemplates.T(loc, "game.notifications.source_unknown")
	}
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
	return normalizedNotificationPathID(value) != ""
}

func normalizedNotificationPathID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "/") || strings.Contains(value, "\\") {
		return ""
	}
	return value
}
