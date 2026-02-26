package web

import (
	"context"
	"net/http"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	featurenotifications "github.com/louisbranch/fracturing.space/internal/services/web/feature/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) appNotificationsRouteDependencies(w http.ResponseWriter, r *http.Request) featurenotifications.AppNotificationsHandlers {
	sess := sessionFromRequest(r, h.sessions)
	setUnreadCache := func(bool) {}
	if sess != nil {
		setUnreadCache = sess.setCachedUnreadNotifications
	}
	var markNotificationRead func(context.Context, string) error
	if h.notificationClient != nil {
		markNotificationRead = func(ctx context.Context, notificationID string) error {
			_, err := h.notificationClient.MarkNotificationRead(ctx, &notificationsv1.MarkNotificationReadRequest{
				NotificationId: notificationID,
			})
			return err
		}
	}
	var clearUnreadCache func()
	if sess != nil {
		clearUnreadCache = func() {
			sess.clearCachedUnreadNotifications()
		}
	}
	var listNotifications func(context.Context, *notificationsv1.ListNotificationsRequest) (*notificationsv1.ListNotificationsResponse, error)
	if h.notificationClient != nil {
		listNotifications = func(ctx context.Context, req *notificationsv1.ListNotificationsRequest) (*notificationsv1.ListNotificationsResponse, error) {
			return h.notificationClient.ListNotifications(ctx, req)
		}
	}
	return featurenotifications.AppNotificationsHandlers{
		Authenticate: func(req *http.Request) bool {
			return sessionFromRequest(req, h.sessions) != nil
		},
		RedirectToLogin: func(writer http.ResponseWriter, req *http.Request) {
			http.Redirect(writer, req, routepath.AuthLogin, http.StatusFound)
		},
		HasNotificationClient: func() bool {
			return h.notificationClient != nil
		},
		ReadContext: func(writer http.ResponseWriter, req *http.Request, title string) (context.Context, string, bool) {
			return campaignfeature.ReadCampaignContext(
				writer,
				req,
				title,
				func(response http.ResponseWriter, request *http.Request) bool {
					return sessionFromRequest(request, h.sessions) != nil
				},
				func(ctx context.Context, request *http.Request) (string, error) {
					if request == nil {
						return "", nil
					}
					sess := sessionFromRequest(request, h.sessions)
					return h.sessionUserIDForSession(ctx, sess)
				},
				h.renderErrorPage,
			)
		},
		ListNotifications:    listNotifications,
		MarkNotificationRead: markNotificationRead,
		ClearUnreadCache:     clearUnreadCache,
		SetUnreadCache:       setUnreadCache,
		PageContext: func(req *http.Request) webtemplates.PageContext {
			return h.pageContext(w, req)
		},
		RenderErrorPage: h.renderErrorPage,
		Now:             notificationsNow,
	}
}

func (h *handler) appNotificationOpenRouteDependencies(w http.ResponseWriter, r *http.Request) featurenotifications.AppNotificationsHandlers {
	var markNotificationRead func(context.Context, string) error
	if h.notificationClient != nil {
		markNotificationRead = func(ctx context.Context, notificationID string) error {
			_, err := h.notificationClient.MarkNotificationRead(ctx, &notificationsv1.MarkNotificationReadRequest{
				NotificationId: notificationID,
			})
			return err
		}
	}
	var clearUnreadCache func()
	if sess := sessionFromRequest(r, h.sessions); sess != nil {
		clearUnreadCache = func() {
			sess.clearCachedUnreadNotifications()
		}
	}
	return featurenotifications.AppNotificationsHandlers{
		Authenticate: func(req *http.Request) bool {
			return sessionFromRequest(req, h.sessions) != nil
		},
		RedirectToLogin: func(writer http.ResponseWriter, req *http.Request) {
			http.Redirect(writer, req, routepath.AuthLogin, http.StatusFound)
		},
		HasNotificationClient: func() bool {
			return h.notificationClient != nil
		},
		ReadContext: func(writer http.ResponseWriter, req *http.Request, title string) (context.Context, string, bool) {
			return campaignfeature.ReadCampaignContext(
				writer,
				req,
				title,
				func(response http.ResponseWriter, request *http.Request) bool {
					return sessionFromRequest(request, h.sessions) != nil
				},
				func(ctx context.Context, request *http.Request) (string, error) {
					if request == nil {
						return "", nil
					}
					sess := sessionFromRequest(request, h.sessions)
					return h.sessionUserIDForSession(ctx, sess)
				},
				h.renderErrorPage,
			)
		},
		MarkNotificationRead: markNotificationRead,
		ClearUnreadCache:     clearUnreadCache,
		RenderErrorPage:      h.renderErrorPage,
	}
}
