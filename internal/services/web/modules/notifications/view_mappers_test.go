package notifications

import (
	"testing"
	"time"

	webi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
)

func TestNotificationListViewUsesRendererAndSkipsBlankIDs(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	h := handlers{
		renderer: stubNotificationCopyRenderer{
			copy: notificationCopy{Title: "Rendered title", Body: "Rendered body"},
		},
		nowFunc: func() time.Time { return now },
	}
	loc := webi18n.Printer(language.English)

	items := h.notificationListView([]notificationsapp.NotificationSummary{
		{ID: " note-1 ", MessageType: "ignored.topic", PayloadJSON: `{}`, Source: "system", CreatedAt: now.Add(-time.Minute)},
		{ID: " ", MessageType: "ignored.topic", PayloadJSON: `{}`, Source: "system", CreatedAt: now},
	}, loc)

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "note-1" {
		t.Fatalf("ID = %q, want %q", items[0].ID, "note-1")
	}
	if items[0].Title != "Rendered title" {
		t.Fatalf("Title = %q, want %q", items[0].Title, "Rendered title")
	}
	if items[0].Body != "Rendered body" {
		t.Fatalf("Body = %q, want %q", items[0].Body, "Rendered body")
	}
	if items[0].OpenURL != routepath.AppNotificationOpen("note-1") {
		t.Fatalf("OpenURL = %q, want %q", items[0].OpenURL, routepath.AppNotificationOpen("note-1"))
	}
	if items[0].DetailURL != routepath.AppNotification("note-1") {
		t.Fatalf("DetailURL = %q, want %q", items[0].DetailURL, routepath.AppNotification("note-1"))
	}
}

func TestNotificationDetailViewUsesRendererAndReturnsNilForBlankID(t *testing.T) {
	t.Parallel()

	h := handlers{
		renderer: stubNotificationCopyRenderer{
			copy: notificationCopy{Title: "Rendered title", Body: "Rendered body"},
		},
		nowFunc: func() time.Time { return time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC) },
	}
	loc := webi18n.Printer(language.English)

	if got := h.notificationDetailView(notificationsapp.NotificationSummary{}, loc); got != nil {
		t.Fatalf("notificationDetailView(blank) = %+v, want nil", got)
	}

	view := h.notificationDetailView(notificationsapp.NotificationSummary{ID: " note-1 ", Source: "system"}, loc)
	if view == nil {
		t.Fatal("notificationDetailView() = nil, want view")
	}
	if view.Title != "Rendered title" {
		t.Fatalf("Title = %q, want %q", view.Title, "Rendered title")
	}
	if view.Body != "Rendered body" {
		t.Fatalf("Body = %q, want %q", view.Body, "Rendered body")
	}
}

type stubNotificationCopyRenderer struct {
	copy notificationCopy
}

func (s stubNotificationCopyRenderer) RenderInApp(webtemplates.Localizer, notificationsapp.NotificationSummary) notificationCopy {
	return s.copy
}
