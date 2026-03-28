package notifications

import (
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// notificationCopy is the transport-local detail payload used by view mapping.
type notificationCopy struct {
	Title   string
	Body    string
	Facts   []NotificationFactView
	Actions []NotificationActionView
}

// notificationCopyRenderer isolates message copy rendering from URL/time/view
// mapping so transport tests can stub copy behavior without importing catalog
// logic.
type notificationCopyRenderer interface {
	RenderInApp(webtemplates.Localizer, notificationsapp.NotificationSummary) notificationCopy
}

// defaultNotificationCopyRenderer adapts the shared notification payload
// contract to the transport-facing notification copy contract.
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
	payload, ok := notificationpayload.ParseInAppPayload(item.PayloadJSON)
	if !ok {
		return notificationCopy{
			Title: webtemplates.T(loc, "notification.generic.title"),
			Body:  webtemplates.T(loc, "notification.generic.body"),
		}
	}
	facts := make([]NotificationFactView, 0, len(payload.Facts))
	for _, fact := range payload.Facts {
		facts = append(facts, NotificationFactView{
			Label: platformi18n.ResolveCopy(loc, fact.Label),
			Value: fact.Value,
		})
	}
	actions := make([]NotificationActionView, 0, len(payload.Actions))
	for _, action := range payload.Actions {
		url, ok := notificationActionURL(action)
		if !ok {
			continue
		}
		actions = append(actions, NotificationActionView{
			Label:   platformi18n.ResolveCopy(loc, action.Label),
			URL:     url,
			Method:  action.Method,
			Primary: action.Style == notificationpayload.ActionStylePrimary,
		})
	}
	return notificationCopy{
		Title:   notificationTitle(payload.Title, loc),
		Body:    notificationBody(payload.Body, loc),
		Facts:   facts,
		Actions: actions,
	}
}

// notificationActionURL resolves typed inbox actions into the small set of
// internal routes the notifications transport is allowed to render.
func notificationActionURL(action notificationpayload.PayloadAction) (string, bool) {
	switch action.Kind {
	case notificationpayload.ActionKindPublicInviteView:
		return routepath.PublicInvite(action.TargetID), true
	case notificationpayload.ActionKindAppCampaignOpen:
		return routepath.AppCampaign(action.TargetID), true
	default:
		return "", false
	}
}
