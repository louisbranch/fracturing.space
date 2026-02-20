// File catalog.go defines view data for catalog templates.
package templates

import commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"

// CatalogSection represents a catalog section entry.
type CatalogSection struct {
	ID     string
	Label  string
	URL    string
	IconID commonv1.IconId
}

// CatalogTableRow represents a single row in a catalog table.
type CatalogTableRow struct {
	Primary   string
	DetailURL string
	Cells     []string
}

// CatalogTableView provides data for a catalog list table.
type CatalogTableView struct {
	SectionID   string
	Columns     []string
	Rows        []CatalogTableRow
	Message     string
	NextToken   string
	PrevToken   string
	HrefBaseURL string
	HTMXBaseURL string
}

// CatalogDetailField represents a label/value pair in a detail view.
type CatalogDetailField struct {
	Label string
	Value string
}

// CatalogDetailView provides data for a catalog detail panel.
type CatalogDetailView struct {
	SectionID string
	Title     string
	ID        string
	Fields    []CatalogDetailField
	Message   string
	RawJSON   string
	BackURL   string
}

const (
	CatalogSectionClasses              = "classes"
	CatalogSectionSubclasses           = "subclasses"
	CatalogSectionHeritages            = "heritages"
	CatalogSectionExperiences          = "experiences"
	CatalogSectionDomains              = "domains"
	CatalogSectionDomainCards          = "domain-cards"
	CatalogSectionItems                = "items"
	CatalogSectionWeapons              = "weapons"
	CatalogSectionArmor                = "armor"
	CatalogSectionLoot                 = "loot"
	CatalogSectionDamageTypes          = "damage-types"
	CatalogSectionAdversaries          = "adversaries"
	CatalogSectionBeastforms           = "beastforms"
	CatalogSectionCompanionExperiences = "companion-experiences"
	CatalogSectionEnvironments         = "environments"
)

var daggerheartCatalogSectionIDs = []string{
	CatalogSectionClasses,
	CatalogSectionSubclasses,
	CatalogSectionHeritages,
	CatalogSectionExperiences,
	CatalogSectionDomains,
	CatalogSectionDomainCards,
	CatalogSectionItems,
	CatalogSectionWeapons,
	CatalogSectionArmor,
	CatalogSectionLoot,
	CatalogSectionDamageTypes,
	CatalogSectionAdversaries,
	CatalogSectionBeastforms,
	CatalogSectionCompanionExperiences,
	CatalogSectionEnvironments,
}

// DefaultDaggerheartCatalogSection returns the default catalog section ID.
func DefaultDaggerheartCatalogSection() string {
	return CatalogSectionClasses
}

// IsDaggerheartCatalogSection reports whether the section ID is known.
func IsDaggerheartCatalogSection(sectionID string) bool {
	for _, entry := range daggerheartCatalogSectionIDs {
		if entry == sectionID {
			return true
		}
	}
	return false
}

// DaggerheartCatalogSections returns the catalog navigation entries.
func DaggerheartCatalogSections(loc Localizer) []CatalogSection {
	return []CatalogSection{
		{ID: CatalogSectionClasses, Label: T(loc, "catalog.daggerheart.classes"), URL: "/catalog/daggerheart/classes", IconID: commonv1.IconId_ICON_ID_CLASS},
		{ID: CatalogSectionSubclasses, Label: T(loc, "catalog.daggerheart.subclasses"), URL: "/catalog/daggerheart/subclasses", IconID: commonv1.IconId_ICON_ID_SUBCLASS},
		{ID: CatalogSectionHeritages, Label: T(loc, "catalog.daggerheart.heritages"), URL: "/catalog/daggerheart/heritages", IconID: commonv1.IconId_ICON_ID_HERITAGE},
		{ID: CatalogSectionExperiences, Label: T(loc, "catalog.daggerheart.experiences"), URL: "/catalog/daggerheart/experiences", IconID: commonv1.IconId_ICON_ID_EXPERIENCE},
		{ID: CatalogSectionDomains, Label: T(loc, "catalog.daggerheart.domains"), URL: "/catalog/daggerheart/domains", IconID: commonv1.IconId_ICON_ID_DOMAIN},
		{ID: CatalogSectionDomainCards, Label: T(loc, "catalog.daggerheart.domain_cards"), URL: "/catalog/daggerheart/domain-cards", IconID: commonv1.IconId_ICON_ID_DOMAIN_CARD},
		{ID: CatalogSectionItems, Label: T(loc, "catalog.daggerheart.items"), URL: "/catalog/daggerheart/items", IconID: commonv1.IconId_ICON_ID_ITEM},
		{ID: CatalogSectionWeapons, Label: T(loc, "catalog.daggerheart.weapons"), URL: "/catalog/daggerheart/weapons", IconID: commonv1.IconId_ICON_ID_WEAPON},
		{ID: CatalogSectionArmor, Label: T(loc, "catalog.daggerheart.armor"), URL: "/catalog/daggerheart/armor", IconID: commonv1.IconId_ICON_ID_ARMOR},
		{ID: CatalogSectionLoot, Label: T(loc, "catalog.daggerheart.loot"), URL: "/catalog/daggerheart/loot", IconID: commonv1.IconId_ICON_ID_LOOT},
		{ID: CatalogSectionDamageTypes, Label: T(loc, "catalog.daggerheart.damage_types"), URL: "/catalog/daggerheart/damage-types", IconID: commonv1.IconId_ICON_ID_DAMAGE},
		{ID: CatalogSectionAdversaries, Label: T(loc, "catalog.daggerheart.adversaries"), URL: "/catalog/daggerheart/adversaries", IconID: commonv1.IconId_ICON_ID_ADVERSARY},
		{ID: CatalogSectionBeastforms, Label: T(loc, "catalog.daggerheart.beastforms"), URL: "/catalog/daggerheart/beastforms", IconID: commonv1.IconId_ICON_ID_ADVERSARY},
		{ID: CatalogSectionCompanionExperiences, Label: T(loc, "catalog.daggerheart.companion_experiences"), URL: "/catalog/daggerheart/companion-experiences", IconID: commonv1.IconId_ICON_ID_EXPERIENCE},
		{ID: CatalogSectionEnvironments, Label: T(loc, "catalog.daggerheart.environments"), URL: "/catalog/daggerheart/environments", IconID: commonv1.IconId_ICON_ID_ENVIRONMENT},
	}
}

// DaggerheartCatalogSectionLabel returns the label for a section.
func DaggerheartCatalogSectionLabel(loc Localizer, sectionID string) string {
	for _, entry := range DaggerheartCatalogSections(loc) {
		if entry.ID == sectionID {
			return entry.Label
		}
	}
	return sectionID
}
