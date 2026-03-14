package notifications

import (
	"testing"
	"time"

	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
)

func TestNotificationPrimaryActionURLForInviteMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		item notificationsapp.NotificationSummary
		want string
	}{
		{
			name: "invite created",
			item: notificationsapp.NotificationSummary{
				MessageType: "campaign.invite.created.v1",
				PayloadJSON: `{"invite_id":"inv-1"}`,
			},
			want: "/invite/inv-1",
		},
		{
			name: "invite accepted",
			item: notificationsapp.NotificationSummary{
				MessageType: "campaign.invite.accepted.v1",
				PayloadJSON: `{"campaign_id":"camp-1"}`,
			},
			want: "/app/campaigns/camp-1",
		},
		{
			name: "unknown message",
			item: notificationsapp.NotificationSummary{
				MessageType: "auth.onboarding.welcome",
				PayloadJSON: `{}`,
			},
			want: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := notificationPrimaryActionURL(tc.item); got != tc.want {
				t.Fatalf("notificationPrimaryActionURL(%+v) = %q, want %q", tc.item, got, tc.want)
			}
		})
	}
}

func TestParseInviteNotificationPayloadAndDetailView(t *testing.T) {
	t.Parallel()

	payload, ok := parseInviteNotificationPayload(`{"invite_id":" inv-1 ","campaign_id":" camp-1 "}`)
	if !ok || payload.InviteID != "inv-1" || payload.CampaignID != "camp-1" {
		t.Fatalf("payload = %+v, ok = %v, want trimmed ids", payload, ok)
	}
	if _, ok := parseInviteNotificationPayload(`{"invite_id":`); ok {
		t.Fatal("expected malformed payload to fail")
	}

	h := handlers{nowFunc: func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) }}
	view := h.notificationDetailView(notificationsapp.NotificationSummary{
		ID:          "note-1",
		MessageType: "campaign.invite.created.v1",
		PayloadJSON: `{"invite_id":"inv-1"}`,
		CreatedAt:   time.Date(2026, 3, 1, 11, 59, 0, 0, time.UTC),
	}, nil)
	if view == nil || view.ID != "note-1" {
		t.Fatalf("view = %+v, want detail view for note-1", view)
	}
}
