package notifications

import (
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
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
	if items[0].IconID != commonv1.IconId_ICON_ID_MESSAGE {
		t.Fatalf("IconID = %v, want %v", items[0].IconID, commonv1.IconId_ICON_ID_MESSAGE)
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
			copy: notificationCopy{
				Title: "Rendered title",
				Body:  "Rendered body",
				Facts: []NotificationFactView{{Label: "Campaign", Value: "Skyfall"}},
				Actions: []NotificationActionView{{
					Label:   "Accept invitation",
					URL:     "/invite/inv-1/accept",
					Method:  "POST",
					Primary: true,
				}},
			},
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
	if view.IconID != commonv1.IconId_ICON_ID_MESSAGE {
		t.Fatalf("IconID = %v, want %v", view.IconID, commonv1.IconId_ICON_ID_MESSAGE)
	}
	if view.Body != "Rendered body" {
		t.Fatalf("Body = %q, want %q", view.Body, "Rendered body")
	}
	if len(view.Facts) != 1 || view.Facts[0].Value != "Skyfall" {
		t.Fatalf("Facts = %+v, want rendered fact", view.Facts)
	}
	if len(view.Actions) != 1 || view.Actions[0].Label != "Accept invitation" {
		t.Fatalf("Actions = %+v, want rendered action", view.Actions)
	}
}

type stubNotificationCopyRenderer struct {
	copy notificationCopy
}

func (s stubNotificationCopyRenderer) RenderInApp(webtemplates.Localizer, notificationsapp.NotificationSummary) notificationCopy {
	return s.copy
}

func TestNotificationMessageIconIDUsesKnownMappingsAndFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messageType string
		want        commonv1.IconId
	}{
		{name: "invite", messageType: "campaign.invite.created.v1", want: commonv1.IconId_ICON_ID_INVITES},
		{name: "campaign fallback", messageType: "campaign.updated.v1", want: commonv1.IconId_ICON_ID_CAMPAIGN},
		{name: "onboarding", messageType: "auth.onboarding.welcome.v1", want: commonv1.IconId_ICON_ID_PROFILE},
		{name: "system message", messageType: "system.message.v1", want: commonv1.IconId_ICON_ID_MESSAGE},
		{name: "unknown fallback", messageType: "custom.topic", want: commonv1.IconId_ICON_ID_MESSAGE},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := notificationMessageIconID(tc.messageType); got != tc.want {
				t.Fatalf("notificationMessageIconID(%q) = %v, want %v", tc.messageType, got, tc.want)
			}
		})
	}
}

func TestNotificationsSideMenuBuildsSharedMenuItems(t *testing.T) {
	t.Parallel()

	loc := webi18n.Printer(language.English)
	items := []NotificationListItemView{
		{
			ID:        "n1",
			IconID:    commonv1.IconId_ICON_ID_MESSAGE,
			Title:     "Welcome",
			DetailURL: routepath.AppNotification("n1"),
		},
		{
			ID:        "n2",
			IconID:    commonv1.IconId_ICON_ID_INVITES,
			Title:     "Campaign invitation",
			DetailURL: routepath.AppNotification("n2"),
		},
	}

	menu := notificationsSideMenu(routepath.AppNotification("n2"), items, loc)
	if menu == nil {
		t.Fatal("notificationsSideMenu() = nil, want menu")
	}
	if menu.CurrentPath != routepath.AppNotification("n2") {
		t.Fatalf("CurrentPath = %q, want %q", menu.CurrentPath, routepath.AppNotification("n2"))
	}
	if len(menu.Items) != 2 {
		t.Fatalf("len(menu.Items) = %d, want 2", len(menu.Items))
	}
	if menu.Items[0].URL != routepath.AppNotification("n1") {
		t.Fatalf("menu.Items[0].URL = %q, want %q", menu.Items[0].URL, routepath.AppNotification("n1"))
	}
	if menu.Items[1].IconID != commonv1.IconId_ICON_ID_INVITES {
		t.Fatalf("menu.Items[1].IconID = %v, want %v", menu.Items[1].IconID, commonv1.IconId_ICON_ID_INVITES)
	}
	if !menu.Items[1].MatchExact {
		t.Fatal("menu.Items[1].MatchExact = false, want true")
	}
}
