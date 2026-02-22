package icons

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestLucideNameCoversIconIds(t *testing.T) {
	for rawID, name := range commonv1.IconId_name {
		id := commonv1.IconId(rawID)
		if id == commonv1.IconId_ICON_ID_UNSPECIFIED {
			continue
		}
		if _, ok := LucideName(id); !ok {
			t.Fatalf("missing Lucide mapping for %s", name)
		}
	}
}

func TestLucideNameNotificationIcons(t *testing.T) {
	tests := []struct {
		name string
		id   commonv1.IconId
		want string
	}{
		{
			name: "notification",
			id:   commonv1.IconId_ICON_ID_NOTIFICATION,
			want: "bell",
		},
		{
			name: "notification unread",
			id:   commonv1.IconId_ICON_ID_NOTIFICATION_UNREAD,
			want: "bell-dot",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, ok := LucideName(tc.id)
			if !ok {
				t.Fatalf("LucideName(%s) missing mapping", tc.id.String())
			}
			if got != tc.want {
				t.Fatalf("LucideName(%s) = %q, want %q", tc.id.String(), got, tc.want)
			}
		})
	}
}
