package icons

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

const lucideSymbolPrefix = "lucide-"

var lucideIconNames = map[commonv1.IconId]string{
	commonv1.IconId_ICON_ID_GENERIC:     "sparkle",
	commonv1.IconId_ICON_ID_CAMPAIGN:    "book-open",
	commonv1.IconId_ICON_ID_PARTICIPANT: "users",
	commonv1.IconId_ICON_ID_CHARACTER:   "user",
	commonv1.IconId_ICON_ID_SESSION:     "calendar",
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
