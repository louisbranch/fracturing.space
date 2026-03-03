package catalog

import (
	"log"
	"strconv"
	"strings"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	// catalogDescriptionLimit caps the number of characters shown in catalog tables.
	catalogDescriptionLimit = 80
)

// catalogSectionColumns defines which columns to show per catalog section.
var catalogSectionColumnKeys = map[string][]string{
	templates.CatalogSectionClasses:              {"catalog.table.starting_hp", "catalog.table.starting_evasion"},
	templates.CatalogSectionSubclasses:           {"catalog.table.spellcast_trait", "catalog.table.feature_count"},
	templates.CatalogSectionHeritages:            {"catalog.table.kind", "catalog.table.feature_count"},
	templates.CatalogSectionExperiences:          {"catalog.table.description"},
	templates.CatalogSectionDomains:              {"catalog.table.description"},
	templates.CatalogSectionDomainCards:          {"catalog.table.domain", "catalog.table.level", "catalog.table.type"},
	templates.CatalogSectionItems:                {"catalog.table.rarity", "catalog.table.kind", "catalog.table.stack_max"},
	templates.CatalogSectionWeapons:              {"catalog.table.category", "catalog.table.tier", "catalog.table.damage_type"},
	templates.CatalogSectionArmor:                {"catalog.table.tier", "catalog.table.armor_score"},
	templates.CatalogSectionLoot:                 {"catalog.table.roll", "catalog.table.description"},
	templates.CatalogSectionDamageTypes:          {"catalog.table.description"},
	templates.CatalogSectionAdversaries:          {"catalog.table.tier", "catalog.table.role"},
	templates.CatalogSectionBeastforms:           {"catalog.table.tier", "catalog.table.trait"},
	templates.CatalogSectionCompanionExperiences: {"catalog.table.description"},
	templates.CatalogSectionEnvironments:         {"catalog.table.type", "catalog.table.difficulty"},
}

func catalogSectionColumns(sectionID string, loc *message.Printer) []string {
	keys, ok := catalogSectionColumnKeys[sectionID]
	if !ok {
		return nil
	}
	columns := make([]string, 0, len(keys))
	for _, key := range keys {
		columns = append(columns, loc.Sprintf(key))
	}
	return columns
}

// buildCatalogClassRows formats class entries for tables.
func buildCatalogClassRows(classes []*daggerheartv1.DaggerheartClass) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(classes))
	for _, entry := range classes {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionClasses, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetStartingHp())),
			strconv.Itoa(int(entry.GetStartingEvasion())),
		}))
	}
	return rows
}

// buildCatalogSubclassRows formats subclass entries for tables.
func buildCatalogSubclassRows(subclasses []*daggerheartv1.DaggerheartSubclass) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(subclasses))
	for _, entry := range subclasses {
		if entry == nil {
			continue
		}
		featureCount := len(entry.GetFoundationFeatures()) + len(entry.GetSpecializationFeatures()) + len(entry.GetMasteryFeatures())
		rows = append(rows, catalogRow(templates.CatalogSectionSubclasses, entry.GetId(), entry.GetName(), []string{
			entry.GetSpellcastTrait(),
			strconv.Itoa(featureCount),
		}))
	}
	return rows
}

// buildCatalogHeritageRows formats heritage entries for tables.
func buildCatalogHeritageRows(heritages []*daggerheartv1.DaggerheartHeritage) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(heritages))
	for _, entry := range heritages {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionHeritages, entry.GetId(), entry.GetName(), []string{
			formatEnumValue(entry.GetKind().String()),
			strconv.Itoa(len(entry.GetFeatures())),
		}))
	}
	return rows
}

// buildCatalogExperienceRows formats experience entries for tables.
func buildCatalogExperienceRows(entries []*daggerheartv1.DaggerheartExperienceEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionExperiences, entry.GetId(), entry.GetName(), []string{
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogDomainRows formats domain entries for tables.
func buildCatalogDomainRows(domains []*daggerheartv1.DaggerheartDomain) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(domains))
	for _, entry := range domains {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionDomains, entry.GetId(), entry.GetName(), []string{
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogDomainCardRows formats domain card entries for tables.
func buildCatalogDomainCardRows(cards []*daggerheartv1.DaggerheartDomainCard) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(cards))
	for _, entry := range cards {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionDomainCards, entry.GetId(), entry.GetName(), []string{
			entry.GetDomainId(),
			strconv.Itoa(int(entry.GetLevel())),
			formatEnumValue(entry.GetType().String()),
		}))
	}
	return rows
}

// buildCatalogItemRows formats item entries for tables.
func buildCatalogItemRows(items []*daggerheartv1.DaggerheartItem) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(items))
	for _, entry := range items {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionItems, entry.GetId(), entry.GetName(), []string{
			formatEnumValue(entry.GetRarity().String()),
			formatEnumValue(entry.GetKind().String()),
			strconv.Itoa(int(entry.GetStackMax())),
		}))
	}
	return rows
}

// buildCatalogWeaponRows formats weapon entries for tables.
func buildCatalogWeaponRows(weapons []*daggerheartv1.DaggerheartWeapon) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(weapons))
	for _, entry := range weapons {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionWeapons, entry.GetId(), entry.GetName(), []string{
			formatEnumValue(entry.GetCategory().String()),
			strconv.Itoa(int(entry.GetTier())),
			formatEnumValue(entry.GetDamageType().String()),
		}))
	}
	return rows
}

// buildCatalogArmorRows formats armor entries for tables.
func buildCatalogArmorRows(armor []*daggerheartv1.DaggerheartArmor) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(armor))
	for _, entry := range armor {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionArmor, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetTier())),
			strconv.Itoa(int(entry.GetArmorScore())),
		}))
	}
	return rows
}

// buildCatalogLootRows formats loot entries for tables.
func buildCatalogLootRows(entries []*daggerheartv1.DaggerheartLootEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionLoot, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetRoll())),
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogDamageTypeRows formats damage type entries for tables.
func buildCatalogDamageTypeRows(entries []*daggerheartv1.DaggerheartDamageTypeEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionDamageTypes, entry.GetId(), entry.GetName(), []string{
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogAdversaryRows formats adversary entries for tables.
func buildCatalogAdversaryRows(entries []*daggerheartv1.DaggerheartAdversaryEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionAdversaries, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetTier())),
			entry.GetRole(),
		}))
	}
	return rows
}

// buildCatalogBeastformRows formats beastform entries for tables.
func buildCatalogBeastformRows(entries []*daggerheartv1.DaggerheartBeastformEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionBeastforms, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetTier())),
			entry.GetTrait(),
		}))
	}
	return rows
}

// buildCatalogCompanionExperienceRows formats companion experience entries for tables.
func buildCatalogCompanionExperienceRows(entries []*daggerheartv1.DaggerheartCompanionExperienceEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionCompanionExperiences, entry.GetId(), entry.GetName(), []string{
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogEnvironmentRows formats environment entries for tables.
func buildCatalogEnvironmentRows(entries []*daggerheartv1.DaggerheartEnvironment) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionEnvironments, entry.GetId(), entry.GetName(), []string{
			formatEnumValue(entry.GetType().String()),
			strconv.Itoa(int(entry.GetDifficulty())),
		}))
	}
	return rows
}

// buildCatalogClassDetail formats class details for the catalog panel.
func buildCatalogClassDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartClass, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get class: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.starting_hp"), Value: strconv.Itoa(int(entry.GetStartingHp()))},
		{Label: loc.Sprintf("catalog.table.starting_evasion"), Value: strconv.Itoa(int(entry.GetStartingEvasion()))},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogSubclassDetail formats subclass details for the catalog panel.
func buildCatalogSubclassDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartSubclass, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get subclass: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	featureCount := len(entry.GetFoundationFeatures()) + len(entry.GetSpecializationFeatures()) + len(entry.GetMasteryFeatures())
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.spellcast_trait"), Value: entry.GetSpellcastTrait()},
		{Label: loc.Sprintf("catalog.table.feature_count"), Value: strconv.Itoa(featureCount)},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogHeritageDetail formats heritage details for the catalog panel.
func buildCatalogHeritageDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartHeritage, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get heritage: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.kind"), Value: formatEnumValue(entry.GetKind().String())},
		{Label: loc.Sprintf("catalog.table.feature_count"), Value: strconv.Itoa(len(entry.GetFeatures()))},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogExperienceDetail formats experience details for the catalog panel.
func buildCatalogExperienceDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartExperienceEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get experience: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogDomainDetail formats domain details for the catalog panel.
func buildCatalogDomainDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartDomain, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get domain: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogDomainCardDetail formats domain card details for the catalog panel.
func buildCatalogDomainCardDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartDomainCard, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get domain card: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.domain"), Value: entry.GetDomainId()},
		{Label: loc.Sprintf("catalog.table.level"), Value: strconv.Itoa(int(entry.GetLevel()))},
		{Label: loc.Sprintf("catalog.table.type"), Value: formatEnumValue(entry.GetType().String())},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogItemDetail formats item details for the catalog panel.
func buildCatalogItemDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartItem, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get item: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.rarity"), Value: formatEnumValue(entry.GetRarity().String())},
		{Label: loc.Sprintf("catalog.table.kind"), Value: formatEnumValue(entry.GetKind().String())},
		{Label: loc.Sprintf("catalog.table.stack_max"), Value: strconv.Itoa(int(entry.GetStackMax()))},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogWeaponDetail formats weapon details for the catalog panel.
func buildCatalogWeaponDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartWeapon, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get weapon: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.category"), Value: formatEnumValue(entry.GetCategory().String())},
		{Label: loc.Sprintf("catalog.table.tier"), Value: strconv.Itoa(int(entry.GetTier()))},
		{Label: loc.Sprintf("catalog.table.damage_type"), Value: formatEnumValue(entry.GetDamageType().String())},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogArmorDetail formats armor details for the catalog panel.
func buildCatalogArmorDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartArmor, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get armor: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.tier"), Value: strconv.Itoa(int(entry.GetTier()))},
		{Label: loc.Sprintf("catalog.table.armor_score"), Value: strconv.Itoa(int(entry.GetArmorScore()))},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogLootDetail formats loot details for the catalog panel.
func buildCatalogLootDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartLootEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get loot entry: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.roll"), Value: strconv.Itoa(int(entry.GetRoll()))},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogDamageTypeDetail formats damage type details for the catalog panel.
func buildCatalogDamageTypeDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartDamageTypeEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get damage type: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogAdversaryDetail formats adversary details for the catalog panel.
func buildCatalogAdversaryDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartAdversaryEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get adversary: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.tier"), Value: strconv.Itoa(int(entry.GetTier()))},
		{Label: loc.Sprintf("catalog.table.role"), Value: entry.GetRole()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogBeastformDetail formats beastform details for the catalog panel.
func buildCatalogBeastformDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartBeastformEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get beastform: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.tier"), Value: strconv.Itoa(int(entry.GetTier()))},
		{Label: loc.Sprintf("catalog.table.trait"), Value: entry.GetTrait()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogCompanionExperienceDetail formats companion experience details for the catalog panel.
func buildCatalogCompanionExperienceDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartCompanionExperienceEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get companion experience: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogEnvironmentDetail formats environment details for the catalog panel.
func buildCatalogEnvironmentDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartEnvironment, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get environment: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.type"), Value: formatEnumValue(entry.GetType().String())},
		{Label: loc.Sprintf("catalog.table.difficulty"), Value: strconv.Itoa(int(entry.GetDifficulty()))},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// catalogRow builds a table row with a detail link for a section.
func catalogRow(sectionID, entryID, name string, cells []string) templates.CatalogTableRow {
	return templates.CatalogTableRow{
		Primary:   catalogPrimaryLabel(name, entryID),
		DetailURL: routepath.CatalogEntry(DaggerheartSystemID, sectionID, entryID),
		Cells:     cells,
	}
}

// catalogPrimaryLabel picks a user-facing label for a row.
func catalogPrimaryLabel(name, entryID string) string {
	if strings.TrimSpace(name) == "" {
		return entryID
	}
	return name
}

// catalogDetailErrorMessage normalizes not-found vs. unavailable states.
func catalogDetailErrorMessage(err error, loc *message.Printer, missing bool) string {
	if !missing {
		return ""
	}
	if err == nil {
		return loc.Sprintf("catalog.error.not_found")
	}
	if status.Code(err) == codes.NotFound {
		return loc.Sprintf("catalog.error.not_found")
	}
	return loc.Sprintf("catalog.error.entry_unavailable")
}

// catalogDetailView builds the shared detail view model with raw JSON.
func catalogDetailView(sectionID, entryID, title string, fields []templates.CatalogDetailField, raw proto.Message, msg string, loc *message.Printer) templates.CatalogDetailView {
	if title == "" {
		title = templates.DaggerheartCatalogSectionLabel(loc, sectionID)
	}
	return templates.CatalogDetailView{
		SectionID: sectionID,
		Title:     title,
		ID:        entryID,
		Fields:    fields,
		Message:   msg,
		RawJSON:   formatProtoJSON(raw),
		BackURL:   routepath.CatalogSection(DaggerheartSystemID, sectionID),
	}
}

// formatProtoJSON renders raw proto data for the detail panel.
func formatProtoJSON(message proto.Message) string {
	if message == nil {
		return ""
	}
	data, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(message)
	if err != nil {
		return ""
	}
	return string(data)
}

// formatEnumValue normalizes enum strings to title case labels.
func formatEnumValue(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "_")
	label := parts[len(parts)-1]
	if label == "UNSPECIFIED" {
		return ""
	}
	label = strings.ToLower(label)
	return strings.ToUpper(label[:1]) + label[1:]
}

// truncateText shortens text to a maximum length with an ellipsis.
func truncateText(text string, limit int) string {
	if limit <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}
