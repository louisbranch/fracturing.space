package notifications

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	notificationsrender "github.com/louisbranch/fracturing.space/internal/services/notifications/render"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/webctx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

type handlers struct {
	service service
	deps    runtimeDependencies
}

type runtimeDependencies struct {
	resolveLanguage module.ResolveLanguage
	resolveViewer   module.ResolveViewer
	resolveUserID   module.ResolveUserID
}

func newRuntimeDependencies(deps module.Dependencies) runtimeDependencies {
	return runtimeDependencies{
		resolveLanguage: deps.ResolveLanguage,
		resolveViewer:   deps.ResolveViewer,
		resolveUserID:   deps.ResolveUserID,
	}
}

func (d runtimeDependencies) moduleDependencies() module.Dependencies {
	return module.Dependencies{
		ResolveViewer:   d.resolveViewer,
		ResolveLanguage: d.resolveLanguage,
		ResolveUserID:   d.resolveUserID,
	}
}

func newHandlers(s service, deps module.Dependencies) handlers {
	return handlers{service: s, deps: newRuntimeDependencies(deps)}
}

func (h handlers) handleDetailRoute(w http.ResponseWriter, r *http.Request) {
	notificationID := strings.TrimSpace(r.PathValue("notificationID"))
	if notificationID == "" {
		h.handleNotFound(w, r)
		return
	}
	h.handleDetail(w, r, notificationID)
}

func (h handlers) handleOpenRoute(w http.ResponseWriter, r *http.Request) {
	notificationID := strings.TrimSpace(r.PathValue("notificationID"))
	if notificationID == "" {
		h.handleNotFound(w, r)
		return
	}
	h.handleOpen(w, r, notificationID)
}

func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	weberror.WriteAppError(w, r, http.StatusNotFound, h.deps.moduleDependencies())
}

func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.pageLocalizer(w, r)
	ctx, userID := h.requestContextAndUserID(r)
	items, err := h.service.listNotifications(ctx, userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writePage(w, r, webtemplates.T(loc, "game.notifications.title"), notificationsMainHeader(loc), webtemplates.NotificationsFragment(webtemplates.NotificationsPageView{Items: h.notificationListView(items, loc)}, loc))
}

func (h handlers) handleDetail(w http.ResponseWriter, r *http.Request, notificationID string) {
	loc, _ := h.pageLocalizer(w, r)
	ctx, userID := h.requestContextAndUserID(r)
	items, err := h.service.listNotifications(ctx, userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	item, err := h.service.getNotification(ctx, userID, notificationID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writePage(w, r, webtemplates.T(loc, "game.notifications.title"), notificationsMainHeader(loc), webtemplates.NotificationsFragment(webtemplates.NotificationsPageView{Items: h.notificationListView(items, loc), Selected: h.notificationDetailView(item, loc)}, loc))
}

func (h handlers) handleOpen(w http.ResponseWriter, r *http.Request, notificationID string) {
	ctx, userID := h.requestContextAndUserID(r)
	item, err := h.service.openNotification(ctx, userID, notificationID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	openID := strings.TrimSpace(item.ID)
	if openID == "" {
		openID = notificationID
	}
	h.writeRedirect(w, r, routepath.AppNotification(openID))
}

func notificationsMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "game.notifications.title")}
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
			Topic:       item.Topic,
			PayloadJSON: item.PayloadJSON,
			Channel:     notificationsrender.ChannelInApp,
		})
		rows = append(rows, webtemplates.NotificationListItemView{
			ID:           itemID,
			Title:        notificationTitle(rendered.Title, loc),
			Body:         notificationBody(rendered.BodyText, loc),
			SourceLabel:  notificationSourceLabel(item.Source, loc),
			CreatedLabel: notificationCreatedLabel(item.CreatedAt, loc),
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
		Topic:       item.Topic,
		PayloadJSON: item.PayloadJSON,
		Channel:     notificationsrender.ChannelInApp,
	})
	return &webtemplates.NotificationDetailView{
		ID:           itemID,
		Title:        notificationTitle(rendered.Title, loc),
		Body:         notificationBody(rendered.BodyText, loc),
		SourceLabel:  notificationSourceLabel(item.Source, loc),
		CreatedLabel: notificationCreatedLabel(item.CreatedAt, loc),
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

func notificationCreatedLabel(createdAt time.Time, loc webtemplates.Localizer) string {
	if createdAt.IsZero() {
		return webtemplates.T(loc, "game.notifications.time.just_now")
	}
	delta := time.Since(createdAt.UTC())
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

func (h handlers) writePage(w http.ResponseWriter, r *http.Request, title string, header *webtemplates.AppMainHeader, body templ.Component) {
	if err := pagerender.WriteModulePage(w, r, h.deps.moduleDependencies(), pagerender.ModulePage{
		Title:    title,
		Header:   header,
		Fragment: body,
	}); err != nil {
		h.writeError(w, r, err)
	}
}

func (h handlers) writeRedirect(w http.ResponseWriter, r *http.Request, location string) {
	if w == nil {
		return
	}
	httpx.WriteRedirect(w, r, location)
}

func (h handlers) pageLocalizer(w http.ResponseWriter, r *http.Request) (webtemplates.Localizer, string) {
	loc, lang := webi18n.ResolveLocalizer(w, r, h.deps.resolveLanguage)
	return loc, lang
}

func (h handlers) requestUserID(r *http.Request) string {
	if r == nil || h.deps.resolveUserID == nil {
		return ""
	}
	return strings.TrimSpace(h.deps.resolveUserID(r))
}

func (h handlers) requestContextAndUserID(r *http.Request) (context.Context, string) {
	ctx := webctx.WithResolvedUserID(r, h.deps.resolveUserID)
	return ctx, h.requestUserID(r)
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	weberror.WriteModuleError(w, r, err, h.deps.moduleDependencies())
}
