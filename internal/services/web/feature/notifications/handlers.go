package notifications

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/render"
	websupport "github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const webNotificationsPageSize = 50

const (
	notificationsFilterUnread = "unread"
	notificationsFilterAll    = "all"
)

type AppNotificationsHandlers struct {
	Authenticate          func(*http.Request) bool
	RedirectToLogin       func(http.ResponseWriter, *http.Request)
	HasNotificationClient func() bool
	ReadContext           func(http.ResponseWriter, *http.Request, string) (context.Context, string, bool)
	ListNotifications     func(context.Context, *notificationsv1.ListNotificationsRequest) (*notificationsv1.ListNotificationsResponse, error)
	MarkNotificationRead  func(context.Context, string) error
	SetUnreadCache        func(bool)
	ClearUnreadCache      func()
	PageContext           func(*http.Request) webtemplates.PageContext
	RenderErrorPage       func(http.ResponseWriter, *http.Request, int, string, string)
	Now                   func() time.Time
}

func HandleAppNotifications(h AppNotificationsHandlers, w http.ResponseWriter, r *http.Request) {
	if h.Authenticate == nil ||
		h.RedirectToLogin == nil ||
		h.ReadContext == nil ||
		h.PageContext == nil ||
		h.RenderErrorPage == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		websupport.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	readCtx, _, ok := h.ReadContext(w, r, "Notifications unavailable")
	if !ok {
		return
	}
	hasNotificationClient := false
	if h.HasNotificationClient != nil {
		hasNotificationClient = h.HasNotificationClient()
	}
	if !hasNotificationClient {
		h.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Notifications unavailable", "notification service client is not configured")
		return
	}
	if h.ListNotifications == nil {
		http.NotFound(w, r)
		return
	}
	resp, err := h.ListNotifications(readCtx, &notificationsv1.ListNotificationsRequest{
		PageSize: webNotificationsPageSize,
	})
	if err != nil {
		h.RenderErrorPage(w, r, websupport.GRPCErrorHTTPStatus(err, http.StatusBadGateway), "Notifications unavailable", "failed to list notifications")
		return
	}
	if h.SetUnreadCache != nil {
		h.SetUnreadCache(hasUnreadNotifications(resp.GetNotifications()))
	}

	page := h.PageContext(r)
	query := notificationPageQueryFromRequest(r)
	now := notificationsNow(h.Now)
	items := ToNotificationListItems(page.Loc, resp.GetNotifications(), now)
	state := toNotificationsPageState(page.Loc, items, query)
	if renderedUnreadID := renderedUnreadNotificationID(state.Items); renderedUnreadID != "" {
		if h.MarkNotificationRead != nil && h.MarkNotificationRead(readCtx, renderedUnreadID) == nil {
			if h.ClearUnreadCache != nil {
				h.ClearUnreadCache()
			}
		}
	}
	RenderAppNotificationsPage(w, r, page, state)
}

func HandleAppNotificationOpen(h AppNotificationsHandlers, w http.ResponseWriter, r *http.Request, notificationID string) {
	if h.Authenticate == nil ||
		h.RedirectToLogin == nil ||
		h.ReadContext == nil ||
		h.RenderErrorPage == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		websupport.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if !isSafeNotificationPathID(notificationID) {
		http.NotFound(w, r)
		return
	}
	readCtx, _, ok := h.ReadContext(w, r, "Notification unavailable")
	if !ok {
		return
	}
	hasNotificationClient := false
	if h.HasNotificationClient != nil {
		hasNotificationClient = h.HasNotificationClient()
	}
	if !hasNotificationClient {
		h.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Notification unavailable", "notification service client is not configured")
		return
	}
	if h.MarkNotificationRead == nil {
		http.NotFound(w, r)
		return
	}
	if err := h.MarkNotificationRead(readCtx, notificationID); err != nil {
		h.RenderErrorPage(w, r, websupport.GRPCErrorHTTPStatus(err, http.StatusBadGateway), "Notification unavailable", "failed to mark notification read")
		return
	}
	if h.ClearUnreadCache != nil {
		h.ClearUnreadCache()
	}
	redirectLocation := notificationsListURL(notificationPageQuery{
		filter:     normalizeNotificationsFilter(r.URL.Query().Get("filter")),
		selectedID: normalizedNotificationPathID(notificationID),
	})
	http.Redirect(w, r, redirectLocation, http.StatusFound)
}

func RenderAppNotificationsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, state webtemplates.NotificationsPageState) {
	if err := websupport.WritePage(
		w,
		r,
		webtemplates.NotificationsPage(page, state),
		websupport.ComposeHTMXTitleForPage(page, "game.notifications.title"),
	); err != nil {
		websupport.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
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

func ToNotificationListItems(loc webtemplates.Localizer, notifications []*notificationsv1.Notification, now time.Time) []webtemplates.NotificationListItem {
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
		rendered := render.Render(loc, render.Input{
			Topic:       notification.GetTopic(),
			PayloadJSON: strings.TrimSpace(notification.GetPayloadJson()),
			Channel:     render.ChannelInApp,
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

func hasUnreadNotifications(notifications []*notificationsv1.Notification) bool {
	for _, notification := range notifications {
		if notification == nil {
			continue
		}
		if notification.GetReadAt() == nil {
			return true
		}
	}
	return false
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
		return "/app/notifications"
	}
	values := url.Values{}
	values.Set("filter", normalizeNotificationsFilter(filter))
	values.Set("selected", notificationID)
	return "/app/notifications?" + values.Encode()
}

func notificationsListURL(query notificationPageQuery) string {
	values := url.Values{}
	values.Set("filter", normalizeNotificationsFilter(query.filter))
	selectedID := normalizedNotificationPathID(query.selectedID)
	if selectedID != "" {
		values.Set("selected", selectedID)
	}
	return "/app/notifications?" + values.Encode()
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

func notificationsNow(now func() time.Time) time.Time {
	if now == nil {
		return time.Now().UTC()
	}
	return now().UTC()
}
