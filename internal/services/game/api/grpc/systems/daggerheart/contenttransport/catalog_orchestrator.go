package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

// contentCatalog stages catalog rows so the catalog endpoint can assemble one
// localized response without exposing store details to callers.
type contentCatalog struct {
	store                contentstore.DaggerheartContentReadStore
	locale               commonv1.Locale
	classes              []contentstore.DaggerheartClass
	subclasses           []contentstore.DaggerheartSubclass
	heritages            []contentstore.DaggerheartHeritage
	experiences          []contentstore.DaggerheartExperienceEntry
	adversaries          []contentstore.DaggerheartAdversaryEntry
	beastforms           []contentstore.DaggerheartBeastformEntry
	companionExperiences []contentstore.DaggerheartCompanionExperienceEntry
	lootEntries          []contentstore.DaggerheartLootEntry
	damageTypes          []contentstore.DaggerheartDamageTypeEntry
	domains              []contentstore.DaggerheartDomain
	domainCards          []contentstore.DaggerheartDomainCard
	weapons              []contentstore.DaggerheartWeapon
	armor                []contentstore.DaggerheartArmor
	items                []contentstore.DaggerheartItem
	environments         []contentstore.DaggerheartEnvironment
}

// newContentCatalog binds one store and locale to a catalog assembly run.
func newContentCatalog(store contentstore.DaggerheartContentReadStore, locale commonv1.Locale) *contentCatalog {
	return &contentCatalog{store: store, locale: locale}
}

// run executes the catalog load/localize pipeline in stable step order.
func (catalog *contentCatalog) run(ctx context.Context) error {
	return runContentCatalogSteps(ctx, catalog.steps())
}

// steps lists the catalog pipeline used by the catalog endpoint.
func (catalog *contentCatalog) steps() []contentCatalogStep {
	return []contentCatalogStep{
		{name: "list classes", run: catalog.listClasses},
		{name: "list subclasses", run: catalog.listSubclasses},
		{name: "list heritages", run: catalog.listHeritages},
		{name: "list experiences", run: catalog.listExperiences},
		{name: "list adversaries", run: catalog.listAdversaries},
		{name: "list beastforms", run: catalog.listBeastforms},
		{name: "list companion experiences", run: catalog.listCompanionExperiences},
		{name: "list loot entries", run: catalog.listLootEntries},
		{name: "list damage types", run: catalog.listDamageTypes},
		{name: "list domains", run: catalog.listDomains},
		{name: "list domain cards", run: catalog.listDomainCards},
		{name: "list weapons", run: catalog.listWeapons},
		{name: "list armor", run: catalog.listArmor},
		{name: "list items", run: catalog.listItems},
		{name: "list environments", run: catalog.listEnvironments},
		{name: "localize classes", run: catalog.localizeClasses},
		{name: "localize subclasses", run: catalog.localizeSubclasses},
		{name: "localize heritages", run: catalog.localizeHeritages},
		{name: "localize experiences", run: catalog.localizeExperiences},
		{name: "localize adversaries", run: catalog.localizeAdversaries},
		{name: "localize beastforms", run: catalog.localizeBeastforms},
		{name: "localize companion experiences", run: catalog.localizeCompanionExperiences},
		{name: "localize loot entries", run: catalog.localizeLootEntries},
		{name: "localize damage types", run: catalog.localizeDamageTypes},
		{name: "localize domains", run: catalog.localizeDomains},
		{name: "localize domain cards", run: catalog.localizeDomainCards},
		{name: "localize weapons", run: catalog.localizeWeapons},
		{name: "localize armor", run: catalog.localizeArmor},
		{name: "localize items", run: catalog.localizeItems},
		{name: "localize environments", run: catalog.localizeEnvironments},
	}
}

// proto materializes the assembled catalog into the transport response shape.
func (catalog *contentCatalog) proto() *pb.DaggerheartContentCatalog {
	return &pb.DaggerheartContentCatalog{
		Classes:              toProtoDaggerheartClasses(catalog.classes),
		Subclasses:           toProtoDaggerheartSubclasses(catalog.subclasses),
		Heritages:            toProtoDaggerheartHeritages(catalog.heritages),
		Experiences:          toProtoDaggerheartExperiences(catalog.experiences),
		Adversaries:          toProtoDaggerheartAdversaryEntries(catalog.adversaries),
		Beastforms:           toProtoDaggerheartBeastforms(catalog.beastforms),
		CompanionExperiences: toProtoDaggerheartCompanionExperiences(catalog.companionExperiences),
		LootEntries:          toProtoDaggerheartLootEntries(catalog.lootEntries),
		DamageTypes:          toProtoDaggerheartDamageTypes(catalog.damageTypes),
		Domains:              toProtoDaggerheartDomains(catalog.domains),
		DomainCards:          toProtoDaggerheartDomainCards(catalog.domainCards),
		Weapons:              toProtoDaggerheartWeapons(catalog.weapons),
		Armor:                toProtoDaggerheartArmorList(catalog.armor),
		Items:                toProtoDaggerheartItems(catalog.items),
		Environments:         toProtoDaggerheartEnvironments(catalog.environments),
	}
}
