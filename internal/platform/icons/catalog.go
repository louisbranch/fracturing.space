package icons

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

// Definition describes a core icon entry.
type Definition struct {
	ID          commonv1.IconId
	Name        string
	Description string
}

var catalog = []Definition{
	{
		ID:          commonv1.IconId_ICON_ID_GENERIC,
		Name:        "Generic",
		Description: "Default icon for uncategorized entries.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_CAMPAIGN,
		Name:        "Campaign",
		Description: "Campaign lifecycle and metadata events.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_SESSION,
		Name:        "Session",
		Description: "Session lifecycle and control events.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_PARTICIPANT,
		Name:        "Participant",
		Description: "Participant and seat management events.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_CHARACTER,
		Name:        "Character",
		Description: "Character creation and updates.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_GM,
		Name:        "GM",
		Description: "Game master actions and GM-facing events.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_CHAT,
		Name:        "Chat",
		Description: "Table chat and communication.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_DECISION,
		Name:        "Decision",
		Description: "Decision gates and table prompts.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_NOTE,
		Name:        "Note",
		Description: "Notes, canon, and story annotations.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_ROLL,
		Name:        "Roll",
		Description: "Dice rolls and resolution events.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_COMBAT,
		Name:        "Combat",
		Description: "Combat actions and conflicts.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_DAMAGE,
		Name:        "Damage",
		Description: "Damage application and outcomes.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_ARMOR,
		Name:        "Armor",
		Description: "Armor and mitigation effects.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_CONDITION,
		Name:        "Condition",
		Description: "Status conditions and constraints.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_DEATH,
		Name:        "Death",
		Description: "Death moves and character loss.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_REST,
		Name:        "Rest",
		Description: "Rests, downtime, and recovery.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_COUNTDOWN,
		Name:        "Countdown",
		Description: "Countdown clocks and timers.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_HOPE,
		Name:        "Hope",
		Description: "Daggerheart Hope resource.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_STRESS,
		Name:        "Stress",
		Description: "Daggerheart Stress resource.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_GM_FEAR,
		Name:        "GM Fear",
		Description: "Daggerheart GM Fear resource.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_DOMAIN,
		Name:        "Domain",
		Description: "Daggerheart domains.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_DOMAIN_CARD,
		Name:        "Domain Card",
		Description: "Domain cards and loadout entries.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_CLASS,
		Name:        "Class",
		Description: "Character classes.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_SUBCLASS,
		Name:        "Subclass",
		Description: "Subclass features and progression.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_HERITAGE,
		Name:        "Heritage",
		Description: "Heritages (ancestry/community).",
	},
	{
		ID:          commonv1.IconId_ICON_ID_EXPERIENCE,
		Name:        "Experience",
		Description: "Experiences and narrative tags.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_WEAPON,
		Name:        "Weapon",
		Description: "Weapons and attack gear.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_ITEM,
		Name:        "Item",
		Description: "Items and equipment.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_LOOT,
		Name:        "Loot",
		Description: "Loot and rewards.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_ADVERSARY,
		Name:        "Adversary",
		Description: "Adversaries and enemies.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_ENVIRONMENT,
		Name:        "Environment",
		Description: "Environments and scenes.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_ANIMAL,
		Name:        "Animal",
		Description: "Animal encounters.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_KEY,
		Name:        "Key",
		Description: "Secret or API key actions.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_LOG_OUT,
		Name:        "Log Out",
		Description: "User logout actions.",
	},
	{
		ID:          commonv1.IconId_ICON_ID_SETTINGS,
		Name:        "Settings",
		Description: "Application settings and configuration.",
	},
}

// Catalog returns a copy of the icon catalog definitions.
func Catalog() []Definition {
	result := make([]Definition, len(catalog))
	copy(result, catalog)
	return result
}

// CatalogMarkdown renders the icon catalog as markdown.
func CatalogMarkdown() string {
	var builder strings.Builder
	builder.WriteString("# Icon Catalog\n\n")
	builder.WriteString("Generated by `go generate ./internal/platform/icons`.\n\n")
	builder.WriteString("| Icon ID | Name | Description |\n")
	builder.WriteString("| --- | --- | --- |\n")
	for _, def := range catalog {
		builder.WriteString("| ")
		builder.WriteString(def.ID.String())
		builder.WriteString(" | ")
		builder.WriteString(def.Name)
		builder.WriteString(" | ")
		builder.WriteString(def.Description)
		builder.WriteString(" |\n")
	}
	return builder.String()
}
