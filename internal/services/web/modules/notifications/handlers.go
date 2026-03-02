package notifications

import (
	"net/http"
	"strings"
	"time"

	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// notificationService defines the service contract used by notification handlers.
type notificationService = notificationsapp.Service

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	service notificationService
	nowFunc func() time.Time
}

// newHandlers builds package wiring for this web seam.
func newHandlers(s notificationService, base modulehandler.Base) handlers {
	return handlers{Base: base, service: s, nowFunc: time.Now}
}

// handleDetailRoute handles this route in the module transport layer.
func (h handlers) handleDetailRoute(w http.ResponseWriter, r *http.Request) {
	notificationID := strings.TrimSpace(r.PathValue("notificationID"))
	if notificationID == "" {
		h.WriteNotFound(w, r)
		return
	}
	h.handleDetail(w, r, notificationID)
}

// handleOpenRoute handles this route in the module transport layer.
func (h handlers) handleOpenRoute(w http.ResponseWriter, r *http.Request) {
	notificationID := strings.TrimSpace(r.PathValue("notificationID"))
	if notificationID == "" {
		h.WriteNotFound(w, r)
		return
	}
	h.handleOpen(w, r, notificationID)
}

// handleIndex handles this route in the module transport layer.
func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, userID := h.RequestContextAndUserID(r)
	items, err := h.service.ListNotifications(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.WritePage(w, r, webtemplates.T(loc, "game.notifications.title"), http.StatusOK, notificationsMainHeader(loc), webtemplates.AppMainLayoutOptions{}, webtemplates.NotificationsFragment(webtemplates.NotificationsPageView{Items: h.notificationListView(items, loc)}, loc))
}

// handleDetail handles this route in the module transport layer.
func (h handlers) handleDetail(w http.ResponseWriter, r *http.Request, notificationID string) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, userID := h.RequestContextAndUserID(r)
	items, err := h.service.ListNotifications(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	item, err := h.service.GetNotification(ctx, userID, notificationID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.WritePage(w, r, webtemplates.T(loc, "game.notifications.title"), http.StatusOK, notificationsMainHeader(loc), webtemplates.AppMainLayoutOptions{}, webtemplates.NotificationsFragment(webtemplates.NotificationsPageView{Items: h.notificationListView(items, loc), Selected: h.notificationDetailView(item, loc)}, loc))
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

// notificationsMainHeader centralizes this web behavior in one helper seam.
func notificationsMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "game.notifications.title")}
}
