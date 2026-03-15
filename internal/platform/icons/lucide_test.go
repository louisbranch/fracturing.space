package icons

import (
	"strings"
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
			name: "invites",
			id:   commonv1.IconId_ICON_ID_INVITES,
			want: "user-plus",
		},
		{
			name: "message",
			id:   commonv1.IconId_ICON_ID_MESSAGE,
			want: "mail",
		},
		{
			name: "notification",
			id:   commonv1.IconId_ICON_ID_NOTIFICATION,
			want: "bell",
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

func TestLucideNameNewPlatformIcons(t *testing.T) {
	tests := []struct {
		name string
		id   commonv1.IconId
		want string
	}{
		{
			name: "scene",
			id:   commonv1.IconId_ICON_ID_SCENE,
			want: "drama",
		},
		{
			name: "hp",
			id:   commonv1.IconId_ICON_ID_HP,
			want: "heart",
		},
		{
			name: "evasion",
			id:   commonv1.IconId_ICON_ID_EVASION,
			want: "redo-dot",
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

func TestLucideNameLocaleIcon(t *testing.T) {
	got, ok := LucideName(commonv1.IconId_ICON_ID_LOCALE)
	if !ok {
		t.Fatalf("LucideName(%s) missing mapping", commonv1.IconId_ICON_ID_LOCALE.String())
	}
	if got != "languages" {
		t.Fatalf("LucideName(%s) = %q, want %q", commonv1.IconId_ICON_ID_LOCALE.String(), got, "languages")
	}
}

func TestLucideSpriteIncludesLocaleSymbol(t *testing.T) {
	if !strings.Contains(LucideSprite(), `id="lucide-languages"`) {
		t.Fatalf("LucideSprite() missing locale icon symbol")
	}
}

func TestLucideSpriteIncludesUpdatedInviteAndMessageSymbols(t *testing.T) {
	sprite := LucideSprite()
	for _, symbolID := range []string{
		`id="lucide-drama"`,
		`id="lucide-heart"`,
		`id="lucide-redo-dot"`,
		`id="lucide-user-plus"`,
		`id="lucide-mail"`,
	} {
		if !strings.Contains(sprite, symbolID) {
			t.Fatalf("LucideSprite() missing symbol %s", symbolID)
		}
	}
}
