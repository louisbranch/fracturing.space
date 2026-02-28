package notifications

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// notificationService defines the service operations used by notification handlers.
type notificationService interface {
	listNotifications(ctx context.Context, userID string) ([]NotificationSummary, error)
	getNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error)
	openNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error)
}

type handlers struct {
	modulehandler.Base
	service notificationService
	nowFunc func() time.Time
}

func newHandlers(s service, base modulehandler.Base) handlers {
	return handlers{Base: base, service: s, nowFunc: time.Now}
}

func (h handlers) handleDetailRoute(w http.ResponseWriter, r *http.Request) {
	notificationID := strings.TrimSpace(r.PathValue("notificationID"))
	if notificationID == "" {
		h.WriteNotFound(w, r)
		return
	}
	h.handleDetail(w, r, notificationID)
}

func (h handlers) handleOpenRoute(w http.ResponseWriter, r *http.Request) {
	notificationID := strings.TrimSpace(r.PathValue("notificationID"))
	if notificationID == "" {
		h.WriteNotFound(w, r)
		return
	}
	h.handleOpen(w, r, notificationID)
}

func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, userID := h.RequestContextAndUserID(r)
	items, err := h.service.listNotifications(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.WritePage(w, r, webtemplates.T(loc, "game.notifications.title"), http.StatusOK, notificationsMainHeader(loc), webtemplates.AppMainLayoutOptions{}, webtemplates.NotificationsFragment(webtemplates.NotificationsPageView{Items: h.notificationListView(items, loc)}, loc))
}

func (h handlers) handleDetail(w http.ResponseWriter, r *http.Request, notificationID string) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, userID := h.RequestContextAndUserID(r)
	items, err := h.service.listNotifications(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	item, err := h.service.getNotification(ctx, userID, notificationID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.WritePage(w, r, webtemplates.T(loc, "game.notifications.title"), http.StatusOK, notificationsMainHeader(loc), webtemplates.AppMainLayoutOptions{}, webtemplates.NotificationsFragment(webtemplates.NotificationsPageView{Items: h.notificationListView(items, loc), Selected: h.notificationDetailView(item, loc)}, loc))
}

func (h handlers) handleOpen(w http.ResponseWriter, r *http.Request, notificationID string) {
	ctx, userID := h.RequestContextAndUserID(r)
	item, err := h.service.openNotification(ctx, userID, notificationID)
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

func notificationsMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "game.notifications.title")}
}
