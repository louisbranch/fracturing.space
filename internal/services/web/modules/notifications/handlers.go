package notifications

import (
	"context"
	"net/http"
	"strings"
	"time"

	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/routeparam"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	service  notificationsapp.Service
	renderer notificationCopyRenderer
	nowFunc  func() time.Time
}

// newHandlers builds package wiring for this web seam.
func newHandlers(s notificationsapp.Service, base modulehandler.Base) handlers {
	return handlers{
		Base:     base,
		service:  s,
		renderer: defaultNotificationCopyRenderer{},
		nowFunc:  time.Now,
	}
}

// routeNotificationID extracts the canonical notification route parameter.
func (h handlers) routeNotificationID(r *http.Request) (string, bool) {
	return routeparam.Read(r, "notificationID")
}

// withNotificationID extracts the notification ID path param and delegates to
// fn, returning 404 when the param is missing.
func (h handlers) withNotificationID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return routeparam.WithRequired("notificationID", h.WriteNotFound, fn)
}

// handleIndex handles this route in the module transport layer.
func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, userID := h.RequestContextAndUserID(r)
	listItems, err := h.loadNotificationListView(ctx, userID, loc)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.writeNotificationsPage(w, r, loc, routepath.AppNotifications, listItems, nil)
}

// handleDetail handles this route in the module transport layer.
func (h handlers) handleDetail(w http.ResponseWriter, r *http.Request, notificationID string) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, userID := h.RequestContextAndUserID(r)
	listItems, err := h.loadNotificationListView(ctx, userID, loc)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	item, err := h.service.GetNotification(ctx, userID, notificationID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	selected := h.notificationDetailView(item, loc)
	currentPath := routepath.AppNotification(notificationID)
	if selected != nil && strings.TrimSpace(selected.DetailURL) != "" {
		currentPath = selected.DetailURL
	}
	h.writeNotificationsPage(w, r, loc, currentPath, listItems, selected)
}

// handleOpen handles this route in the module transport layer.
func (h handlers) handleOpen(w http.ResponseWriter, r *http.Request, notificationID string) {
	ctx, userID := h.RequestContextAndUserID(r)
	item, err := h.service.OpenNotification(ctx, userID, notificationID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	openID := strings.TrimSpace(item.ID)
	if openID == "" {
		openID = notificationID
	}
	httpx.WriteRedirect(w, r, routepath.AppNotification(openID))
}

// loadNotificationListView loads and maps notification list items for rendering.
func (h handlers) loadNotificationListView(ctx context.Context, userID string, loc Localizer) ([]NotificationListItemView, error) {
	items, err := h.service.ListNotifications(ctx, userID)
	if err != nil {
		return nil, err
	}
	return h.notificationListView(items, loc), nil
}

// writeNotificationsPage renders the list/detail notifications page.
func (h handlers) writeNotificationsPage(
	w http.ResponseWriter,
	r *http.Request,
	loc webtemplates.Localizer,
	currentPath string,
	items []NotificationListItemView,
	selected *NotificationDetailView,
) {
	h.WritePage(
		w,
		r,
		webtemplates.T(loc, "game.notifications.title"),
		http.StatusOK,
		notificationsMainHeader(loc),
		webtemplates.AppMainLayoutOptions{SideMenu: notificationsSideMenu(currentPath, items, loc)},
		NotificationsFragment(NotificationsPageView{
			Items:    items,
			Selected: selected,
		}, loc),
	)
}

// notificationsMainHeader centralizes this web behavior in one helper seam.
func notificationsMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "game.notifications.title")}
}
