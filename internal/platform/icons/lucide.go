package icons

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

const lucideSymbolPrefix = "lucide-"

var lucideIconNames = map[commonv1.IconId]string{
	commonv1.IconId_ICON_ID_GENERIC:             "sparkle",
	commonv1.IconId_ICON_ID_CAMPAIGN:            "book-open",
	commonv1.IconId_ICON_ID_SESSION:             "calendar",
	commonv1.IconId_ICON_ID_PARTICIPANT:         "users",
	commonv1.IconId_ICON_ID_CHARACTER:           "square-user",
	commonv1.IconId_ICON_ID_GM:                  "crown",
	commonv1.IconId_ICON_ID_CHAT:                "message-circle",
	commonv1.IconId_ICON_ID_DECISION:            "message-circle-question-mark",
	commonv1.IconId_ICON_ID_NOTE:                "scroll",
	commonv1.IconId_ICON_ID_ROLL:                "dices",
	commonv1.IconId_ICON_ID_COMBAT:              "swords",
	commonv1.IconId_ICON_ID_DAMAGE:              "flame",
	commonv1.IconId_ICON_ID_ARMOR:               "shield",
	commonv1.IconId_ICON_ID_CONDITION:           "badge-alert",
	commonv1.IconId_ICON_ID_DEATH:               "skull",
	commonv1.IconId_ICON_ID_REST:                "tent",
	commonv1.IconId_ICON_ID_COUNTDOWN:           "alarm-clock",
	commonv1.IconId_ICON_ID_HOPE:                "sparkles",
	commonv1.IconId_ICON_ID_STRESS:              "heart-crack",
	commonv1.IconId_ICON_ID_GM_FEAR:             "ghost",
	commonv1.IconId_ICON_ID_DOMAIN:              "library",
	commonv1.IconId_ICON_ID_DOMAIN_CARD:         "wallet-cards",
	commonv1.IconId_ICON_ID_CLASS:               "book-marked",
	commonv1.IconId_ICON_ID_SUBCLASS:            "book-plus",
	commonv1.IconId_ICON_ID_HERITAGE:            "book-user",
	commonv1.IconId_ICON_ID_EXPERIENCE:          "book-heart",
	commonv1.IconId_ICON_ID_WEAPON:              "sword",
	commonv1.IconId_ICON_ID_ITEM:                "backpack",
	commonv1.IconId_ICON_ID_LOOT:                "coins",
	commonv1.IconId_ICON_ID_ADVERSARY:           "venetian-mask",
	commonv1.IconId_ICON_ID_ENVIRONMENT:         "map",
	commonv1.IconId_ICON_ID_ANIMAL:              "paw-print",
	commonv1.IconId_ICON_ID_PROFILE:             "circle-user",
	commonv1.IconId_ICON_ID_INVITES:             "mail",
	commonv1.IconId_ICON_ID_NOTIFICATION:        "bell",
	commonv1.IconId_ICON_ID_NOTIFICATION_UNREAD: "bell-dot",
	commonv1.IconId_ICON_ID_AI:                  "bot",
	commonv1.IconId_ICON_ID_KEY:                 "key",
	commonv1.IconId_ICON_ID_LOG_OUT:             "log-out",
	commonv1.IconId_ICON_ID_SETTINGS:            "settings",
}

// LucideName returns the Lucide icon name for a core icon identifier.
func LucideName(id commonv1.IconId) (string, bool) {
	name, ok := lucideIconNames[id]
	return name, ok
}

// LucideNameOrDefault provides a stable Lucide name even when the icon ID is unknown.
func LucideNameOrDefault(id commonv1.IconId) string {
	if name, ok := lucideIconNames[id]; ok {
		return name
	}
	return "sparkle"
}

// LucideSymbolID returns the sprite symbol ID for a Lucide icon name.
func LucideSymbolID(name string) string {
	return lucideSymbolPrefix + name
}

// LucideSprite returns the SVG sprite markup for core Lucide icons.
func LucideSprite() string {
	return lucideSprite
}
