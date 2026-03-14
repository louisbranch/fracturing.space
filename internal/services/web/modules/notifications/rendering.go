package notifications

import (
	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	notificationsrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/render"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// notificationCopy is the transport-local title/body pair used by view mapping.
type notificationCopy struct {
	Title string
	Body  string
}

// notificationCopyRenderer isolates message copy rendering from URL/time/view
// mapping so transport tests can stub copy behavior without importing catalog
// logic.
type notificationCopyRenderer interface {
	RenderInApp(webtemplates.Localizer, notificationsapp.NotificationSummary) notificationCopy
}

// defaultNotificationCopyRenderer adapts the module-owned render package to the
// transport-facing notification copy contract.
type defaultNotificationCopyRenderer struct{}

// copyRenderer returns the configured renderer or the production default when
// handlers are constructed directly in focused tests.
func (h handlers) copyRenderer() notificationCopyRenderer {
	if h.renderer != nil {
		return h.renderer
	}
	return defaultNotificationCopyRenderer{}
}

// RenderInApp maps notification summaries into already-fallback-normalized copy
// that the surrounding transport layer can place into views directly.
func (defaultNotificationCopyRenderer) RenderInApp(loc webtemplates.Localizer, item notificationsapp.NotificationSummary) notificationCopy {
	rendered := notificationsrender.RenderInApp(loc, notificationsrender.Input{
		MessageType: item.MessageType,
		PayloadJSON: item.PayloadJSON,
	})
	return notificationCopy{
		Title: notificationTitle(rendered.Title, loc),
		Body:  notificationBody(rendered.BodyText, loc),
	}
}
