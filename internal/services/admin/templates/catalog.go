// File catalog.go defines view data for catalog templates.
package templates

// CatalogSection represents a catalog section entry.
type CatalogSection struct {
	ID    string
	Label string
	URL   string
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
		{ID: CatalogSectionClasses, Label: T(loc, "catalog.daggerheart.classes"), URL: "/catalog/daggerheart/classes"},
		{ID: CatalogSectionSubclasses, Label: T(loc, "catalog.daggerheart.subclasses"), URL: "/catalog/daggerheart/subclasses"},
		{ID: CatalogSectionHeritages, Label: T(loc, "catalog.daggerheart.heritages"), URL: "/catalog/daggerheart/heritages"},
		{ID: CatalogSectionExperiences, Label: T(loc, "catalog.daggerheart.experiences"), URL: "/catalog/daggerheart/experiences"},
		{ID: CatalogSectionDomains, Label: T(loc, "catalog.daggerheart.domains"), URL: "/catalog/daggerheart/domains"},
		{ID: CatalogSectionDomainCards, Label: T(loc, "catalog.daggerheart.domain_cards"), URL: "/catalog/daggerheart/domain-cards"},
		{ID: CatalogSectionItems, Label: T(loc, "catalog.daggerheart.items"), URL: "/catalog/daggerheart/items"},
		{ID: CatalogSectionWeapons, Label: T(loc, "catalog.daggerheart.weapons"), URL: "/catalog/daggerheart/weapons"},
		{ID: CatalogSectionArmor, Label: T(loc, "catalog.daggerheart.armor"), URL: "/catalog/daggerheart/armor"},
		{ID: CatalogSectionLoot, Label: T(loc, "catalog.daggerheart.loot"), URL: "/catalog/daggerheart/loot"},
		{ID: CatalogSectionDamageTypes, Label: T(loc, "catalog.daggerheart.damage_types"), URL: "/catalog/daggerheart/damage-types"},
		{ID: CatalogSectionAdversaries, Label: T(loc, "catalog.daggerheart.adversaries"), URL: "/catalog/daggerheart/adversaries"},
		{ID: CatalogSectionBeastforms, Label: T(loc, "catalog.daggerheart.beastforms"), URL: "/catalog/daggerheart/beastforms"},
		{ID: CatalogSectionCompanionExperiences, Label: T(loc, "catalog.daggerheart.companion_experiences"), URL: "/catalog/daggerheart/companion-experiences"},
		{ID: CatalogSectionEnvironments, Label: T(loc, "catalog.daggerheart.environments"), URL: "/catalog/daggerheart/environments"},
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
