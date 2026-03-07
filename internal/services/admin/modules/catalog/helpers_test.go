package catalog

import (
	"errors"
	"strings"
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCatalogSectionColumns(t *testing.T) {
	loc := i18nhttp.Printer(i18nhttp.Default())

	if cols := catalogSectionColumns("missing", loc); cols != nil {
		t.Fatalf("catalogSectionColumns(missing) = %#v", cols)
	}
	cols := catalogSectionColumns(templates.CatalogSectionClasses, loc)
	if len(cols) != 2 {
		t.Fatalf("catalogSectionColumns(classes) len = %d", len(cols))
	}
}

func TestCatalogRowsBuilders(t *testing.T) {
	rows := buildCatalogClassRows([]*daggerheartv1.DaggerheartClass{
		nil,
		{Id: "class-1", Name: "Sentinel", StartingHp: 6, StartingEvasion: 12},
	})
	if len(rows) != 1 || rows[0].Primary != "Sentinel" {
		t.Fatalf("buildCatalogClassRows() = %#v", rows)
	}

	rows = buildCatalogSubclassRows([]*daggerheartv1.DaggerheartSubclass{
		{
			Id:                     "subclass-1",
			Name:                   "Warden",
			SpellcastTrait:         "Presence",
			FoundationFeatures:     []*daggerheartv1.DaggerheartFeature{{Id: "f1"}},
			SpecializationFeatures: []*daggerheartv1.DaggerheartFeature{{Id: "f2"}},
			MasteryFeatures:        []*daggerheartv1.DaggerheartFeature{{Id: "f3"}},
		},
	})
	if len(rows) != 1 || len(rows[0].Cells) != 2 || rows[0].Cells[1] != "3" {
		t.Fatalf("buildCatalogSubclassRows() = %#v", rows)
	}

	rows = buildCatalogHeritageRows([]*daggerheartv1.DaggerheartHeritage{
		{Id: "heritage-1", Name: "Elder", Features: []*daggerheartv1.DaggerheartFeature{{Id: "f1"}}},
	})
	if len(rows) != 1 || len(rows[0].Cells) != 2 {
		t.Fatalf("buildCatalogHeritageRows() = %#v", rows)
	}

	rows = buildCatalogExperienceRows([]*daggerheartv1.DaggerheartExperienceEntry{
		{Id: "exp-1", Name: "Scholar", Description: strings.Repeat("a", catalogDescriptionLimit+10)},
	})
	if len(rows) != 1 || !strings.HasSuffix(rows[0].Cells[0], "...") {
		t.Fatalf("buildCatalogExperienceRows() = %#v", rows)
	}

	rows = buildCatalogDomainRows([]*daggerheartv1.DaggerheartDomain{{Id: "domain-1", Name: "Arcana", Description: "description"}})
	if len(rows) != 1 || rows[0].Primary != "Arcana" {
		t.Fatalf("buildCatalogDomainRows() = %#v", rows)
	}

	rows = buildCatalogDomainCardRows([]*daggerheartv1.DaggerheartDomainCard{{Id: "card-1", Name: "Flare", DomainId: "domain-1", Level: 2}})
	if len(rows) != 1 || rows[0].Cells[0] != "domain-1" {
		t.Fatalf("buildCatalogDomainCardRows() = %#v", rows)
	}

	rows = buildCatalogItemRows([]*daggerheartv1.DaggerheartItem{{Id: "item-1", Name: "Draught", StackMax: 2}})
	if len(rows) != 1 || rows[0].Primary != "Draught" {
		t.Fatalf("buildCatalogItemRows() = %#v", rows)
	}

	rows = buildCatalogWeaponRows([]*daggerheartv1.DaggerheartWeapon{{Id: "weapon-1", Name: "Longsword", Tier: 1}})
	if len(rows) != 1 || rows[0].Primary != "Longsword" {
		t.Fatalf("buildCatalogWeaponRows() = %#v", rows)
	}

	rows = buildCatalogArmorRows([]*daggerheartv1.DaggerheartArmor{{Id: "armor-1", Name: "Chainmail", Tier: 1, ArmorScore: 3}})
	if len(rows) != 1 || rows[0].Cells[1] != "3" {
		t.Fatalf("buildCatalogArmorRows() = %#v", rows)
	}

	rows = buildCatalogLootRows([]*daggerheartv1.DaggerheartLootEntry{{Id: "loot-1", Name: "Coin", Roll: 5, Description: "old coin"}})
	if len(rows) != 1 || rows[0].Cells[0] != "5" {
		t.Fatalf("buildCatalogLootRows() = %#v", rows)
	}

	rows = buildCatalogDamageTypeRows([]*daggerheartv1.DaggerheartDamageTypeEntry{{Id: "damage-1", Name: "Fire", Description: "burn"}})
	if len(rows) != 1 || rows[0].Primary != "Fire" {
		t.Fatalf("buildCatalogDamageTypeRows() = %#v", rows)
	}

	rows = buildCatalogAdversaryRows([]*daggerheartv1.DaggerheartAdversaryEntry{{Id: "adv-1", Name: "Razorclaw", Tier: 2, Role: "Brute"}})
	if len(rows) != 1 || rows[0].Cells[1] != "Brute" {
		t.Fatalf("buildCatalogAdversaryRows() = %#v", rows)
	}

	rows = buildCatalogBeastformRows([]*daggerheartv1.DaggerheartBeastformEntry{{Id: "beast-1", Name: "Dire Wolf", Tier: 1, Trait: "Pack"}})
	if len(rows) != 1 || rows[0].Cells[1] != "Pack" {
		t.Fatalf("buildCatalogBeastformRows() = %#v", rows)
	}

	rows = buildCatalogCompanionExperienceRows([]*daggerheartv1.DaggerheartCompanionExperienceEntry{{Id: "comp-1", Name: "Tracker", Description: "keen senses"}})
	if len(rows) != 1 || rows[0].Primary != "Tracker" {
		t.Fatalf("buildCatalogCompanionExperienceRows() = %#v", rows)
	}

	rows = buildCatalogEnvironmentRows([]*daggerheartv1.DaggerheartEnvironment{{Id: "env-1", Name: "Haunted Keep", Difficulty: 3}})
	if len(rows) != 1 || rows[0].Cells[1] != "3" {
		t.Fatalf("buildCatalogEnvironmentRows() = %#v", rows)
	}
}

func TestCatalogDetailBuilders(t *testing.T) {
	loc := i18nhttp.Printer(i18nhttp.Default())

	classDetail := buildCatalogClassDetail(
		templates.CatalogSectionClasses,
		"class-1",
		&daggerheartv1.DaggerheartClass{Id: "class-1", Name: "Sentinel", StartingHp: 6, StartingEvasion: 12},
		nil,
		loc,
	)
	if classDetail.ID != "class-1" || classDetail.Title == "" || classDetail.RawJSON == "" {
		t.Fatalf("buildCatalogClassDetail() = %#v", classDetail)
	}

	subclassDetail := buildCatalogSubclassDetail(
		templates.CatalogSectionSubclasses,
		"subclass-1",
		&daggerheartv1.DaggerheartSubclass{Id: "subclass-1", Name: "Warden"},
		nil,
		loc,
	)
	if subclassDetail.ID != "subclass-1" || len(subclassDetail.Fields) == 0 {
		t.Fatalf("buildCatalogSubclassDetail() = %#v", subclassDetail)
	}

	heritageDetail := buildCatalogHeritageDetail(
		templates.CatalogSectionHeritages,
		"heritage-1",
		&daggerheartv1.DaggerheartHeritage{Id: "heritage-1", Name: "Elder"},
		nil,
		loc,
	)
	if heritageDetail.ID != "heritage-1" {
		t.Fatalf("buildCatalogHeritageDetail() = %#v", heritageDetail)
	}

	expDetail := buildCatalogExperienceDetail(
		templates.CatalogSectionExperiences,
		"exp-1",
		&daggerheartv1.DaggerheartExperienceEntry{Id: "exp-1", Name: "Scholar", Description: "desc"},
		nil,
		loc,
	)
	if expDetail.ID != "exp-1" {
		t.Fatalf("buildCatalogExperienceDetail() = %#v", expDetail)
	}

	domainDetail := buildCatalogDomainDetail(
		templates.CatalogSectionDomains,
		"domain-1",
		&daggerheartv1.DaggerheartDomain{Id: "domain-1", Name: "Arcana", Description: "desc"},
		nil,
		loc,
	)
	if domainDetail.ID != "domain-1" {
		t.Fatalf("buildCatalogDomainDetail() = %#v", domainDetail)
	}

	cardDetail := buildCatalogDomainCardDetail(
		templates.CatalogSectionDomainCards,
		"card-1",
		&daggerheartv1.DaggerheartDomainCard{Id: "card-1", Name: "Flare", DomainId: "domain-1", Level: 2},
		nil,
		loc,
	)
	if cardDetail.ID != "card-1" {
		t.Fatalf("buildCatalogDomainCardDetail() = %#v", cardDetail)
	}

	itemDetail := buildCatalogItemDetail(
		templates.CatalogSectionItems,
		"item-1",
		&daggerheartv1.DaggerheartItem{Id: "item-1", Name: "Draught", Description: "desc"},
		nil,
		loc,
	)
	if itemDetail.ID != "item-1" {
		t.Fatalf("buildCatalogItemDetail() = %#v", itemDetail)
	}

	weaponDetail := buildCatalogWeaponDetail(
		templates.CatalogSectionWeapons,
		"weapon-1",
		&daggerheartv1.DaggerheartWeapon{Id: "weapon-1", Name: "Longsword", Tier: 1},
		nil,
		loc,
	)
	if weaponDetail.ID != "weapon-1" {
		t.Fatalf("buildCatalogWeaponDetail() = %#v", weaponDetail)
	}

	armorDetail := buildCatalogArmorDetail(
		templates.CatalogSectionArmor,
		"armor-1",
		&daggerheartv1.DaggerheartArmor{Id: "armor-1", Name: "Chainmail", Tier: 1, ArmorScore: 3},
		nil,
		loc,
	)
	if armorDetail.ID != "armor-1" {
		t.Fatalf("buildCatalogArmorDetail() = %#v", armorDetail)
	}

	lootDetail := buildCatalogLootDetail(
		templates.CatalogSectionLoot,
		"loot-1",
		&daggerheartv1.DaggerheartLootEntry{Id: "loot-1", Name: "Coin", Roll: 5},
		nil,
		loc,
	)
	if lootDetail.ID != "loot-1" {
		t.Fatalf("buildCatalogLootDetail() = %#v", lootDetail)
	}

	damageDetail := buildCatalogDamageTypeDetail(
		templates.CatalogSectionDamageTypes,
		"damage-1",
		&daggerheartv1.DaggerheartDamageTypeEntry{Id: "damage-1", Name: "Fire"},
		nil,
		loc,
	)
	if damageDetail.ID != "damage-1" {
		t.Fatalf("buildCatalogDamageTypeDetail() = %#v", damageDetail)
	}

	adversaryDetail := buildCatalogAdversaryDetail(
		templates.CatalogSectionAdversaries,
		"adv-1",
		&daggerheartv1.DaggerheartAdversaryEntry{Id: "adv-1", Name: "Razorclaw", Tier: 2, Role: "Brute"},
		nil,
		loc,
	)
	if adversaryDetail.ID != "adv-1" {
		t.Fatalf("buildCatalogAdversaryDetail() = %#v", adversaryDetail)
	}

	beastDetail := buildCatalogBeastformDetail(
		templates.CatalogSectionBeastforms,
		"beast-1",
		&daggerheartv1.DaggerheartBeastformEntry{Id: "beast-1", Name: "Dire Wolf", Tier: 1, Trait: "Pack"},
		nil,
		loc,
	)
	if beastDetail.ID != "beast-1" {
		t.Fatalf("buildCatalogBeastformDetail() = %#v", beastDetail)
	}

	companionDetail := buildCatalogCompanionExperienceDetail(
		templates.CatalogSectionCompanionExperiences,
		"comp-1",
		&daggerheartv1.DaggerheartCompanionExperienceEntry{Id: "comp-1", Name: "Tracker"},
		nil,
		loc,
	)
	if companionDetail.ID != "comp-1" {
		t.Fatalf("buildCatalogCompanionExperienceDetail() = %#v", companionDetail)
	}

	environmentDetail := buildCatalogEnvironmentDetail(
		templates.CatalogSectionEnvironments,
		"env-1",
		&daggerheartv1.DaggerheartEnvironment{Id: "env-1", Name: "Haunted Keep", Difficulty: 3},
		nil,
		loc,
	)
	if environmentDetail.ID != "env-1" {
		t.Fatalf("buildCatalogEnvironmentDetail() = %#v", environmentDetail)
	}
}

func TestCatalogHelpersShared(t *testing.T) {
	loc := i18nhttp.Printer(i18nhttp.Default())

	if label := catalogPrimaryLabel("", "id-1"); label != "id-1" {
		t.Fatalf("catalogPrimaryLabel() = %q", label)
	}
	if label := catalogPrimaryLabel("Name", "id-1"); label != "Name" {
		t.Fatalf("catalogPrimaryLabel() = %q", label)
	}

	row := catalogRow(templates.CatalogSectionClasses, "class-1", "Sentinel", []string{"6", "12"})
	if !strings.Contains(row.DetailURL, "/app/catalog/") || row.Primary != "Sentinel" {
		t.Fatalf("catalogRow() = %#v", row)
	}

	if msg := catalogDetailErrorMessage(nil, loc, false); msg != "" {
		t.Fatalf("catalogDetailErrorMessage(missing=false) = %q", msg)
	}
	if msg := catalogDetailErrorMessage(nil, loc, true); msg != loc.Sprintf("catalog.error.not_found") {
		t.Fatalf("catalogDetailErrorMessage(nil,missing=true) = %q", msg)
	}
	if msg := catalogDetailErrorMessage(status.Error(codes.NotFound, "missing"), loc, true); msg != loc.Sprintf("catalog.error.not_found") {
		t.Fatalf("catalogDetailErrorMessage(not found) = %q", msg)
	}
	if msg := catalogDetailErrorMessage(errors.New("boom"), loc, true); msg != loc.Sprintf("catalog.error.entry_unavailable") {
		t.Fatalf("catalogDetailErrorMessage(other error) = %q", msg)
	}

	view := catalogDetailView(
		templates.CatalogSectionClasses,
		"class-1",
		"",
		[]templates.CatalogDetailField{{Label: "L", Value: "V"}},
		&daggerheartv1.DaggerheartClass{Id: "class-1", Name: "Sentinel"},
		"",
		loc,
	)
	if view.Title == "" || view.BackURL == "" || view.RawJSON == "" {
		t.Fatalf("catalogDetailView() = %#v", view)
	}

	if raw := formatProtoJSON(nil); raw != "" {
		t.Fatalf("formatProtoJSON(nil) = %q", raw)
	}
	if raw := formatProtoJSON(&daggerheartv1.DaggerheartClass{Id: "class-1"}); !strings.Contains(raw, "class-1") {
		t.Fatalf("formatProtoJSON(class) = %q", raw)
	}

	if got := formatEnumValue(""); got != "" {
		t.Fatalf("formatEnumValue(empty) = %q", got)
	}
	if got := formatEnumValue("DAGGERHEART_WEAPON_CATEGORY_UNSPECIFIED"); got != "" {
		t.Fatalf("formatEnumValue(unspecified) = %q", got)
	}
	if got := formatEnumValue("DAGGERHEART_WEAPON_CATEGORY_MELEE"); got != "Melee" {
		t.Fatalf("formatEnumValue(melee) = %q", got)
	}

	if got := truncateText("hello world", 5); got != "hello..." {
		t.Fatalf("truncateText() = %q", got)
	}
	if got := truncateText("hello", 0); got != "" {
		t.Fatalf("truncateText(limit=0) = %q", got)
	}
}
