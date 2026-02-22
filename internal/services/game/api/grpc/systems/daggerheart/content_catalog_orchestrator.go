package daggerheart

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type contentCatalog struct {
	store                storage.DaggerheartContentReadStore
	locale               commonv1.Locale
	classes              []storage.DaggerheartClass
	subclasses           []storage.DaggerheartSubclass
	heritages            []storage.DaggerheartHeritage
	experiences          []storage.DaggerheartExperienceEntry
	adversaries          []storage.DaggerheartAdversaryEntry
	beastforms           []storage.DaggerheartBeastformEntry
	companionExperiences []storage.DaggerheartCompanionExperienceEntry
	lootEntries          []storage.DaggerheartLootEntry
	damageTypes          []storage.DaggerheartDamageTypeEntry
	domains              []storage.DaggerheartDomain
	domainCards          []storage.DaggerheartDomainCard
	weapons              []storage.DaggerheartWeapon
	armor                []storage.DaggerheartArmor
	items                []storage.DaggerheartItem
	environments         []storage.DaggerheartEnvironment
}

func newContentCatalog(store storage.DaggerheartContentReadStore, locale commonv1.Locale) *contentCatalog {
	return &contentCatalog{store: store, locale: locale}
}

func (catalog *contentCatalog) run(ctx context.Context) error {
	return runContentCatalogSteps(ctx, catalog.steps())
}

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

func (catalog *contentCatalog) listClasses(ctx context.Context) error {
	classes, err := catalog.store.ListDaggerheartClasses(ctx)
	if err != nil {
		return err
	}
	catalog.classes = classes
	return nil
}

func (catalog *contentCatalog) listSubclasses(ctx context.Context) error {
	subclasses, err := catalog.store.ListDaggerheartSubclasses(ctx)
	if err != nil {
		return err
	}
	catalog.subclasses = subclasses
	return nil
}

func (catalog *contentCatalog) listHeritages(ctx context.Context) error {
	heritages, err := catalog.store.ListDaggerheartHeritages(ctx)
	if err != nil {
		return err
	}
	catalog.heritages = heritages
	return nil
}

func (catalog *contentCatalog) listExperiences(ctx context.Context) error {
	experiences, err := catalog.store.ListDaggerheartExperiences(ctx)
	if err != nil {
		return err
	}
	catalog.experiences = experiences
	return nil
}

func (catalog *contentCatalog) listAdversaries(ctx context.Context) error {
	adversaries, err := catalog.store.ListDaggerheartAdversaryEntries(ctx)
	if err != nil {
		return err
	}
	catalog.adversaries = adversaries
	return nil
}

func (catalog *contentCatalog) listBeastforms(ctx context.Context) error {
	beastforms, err := catalog.store.ListDaggerheartBeastforms(ctx)
	if err != nil {
		return err
	}
	catalog.beastforms = beastforms
	return nil
}

func (catalog *contentCatalog) listCompanionExperiences(ctx context.Context) error {
	experiences, err := catalog.store.ListDaggerheartCompanionExperiences(ctx)
	if err != nil {
		return err
	}
	catalog.companionExperiences = experiences
	return nil
}

func (catalog *contentCatalog) listLootEntries(ctx context.Context) error {
	entries, err := catalog.store.ListDaggerheartLootEntries(ctx)
	if err != nil {
		return err
	}
	catalog.lootEntries = entries
	return nil
}

func (catalog *contentCatalog) listDamageTypes(ctx context.Context) error {
	types, err := catalog.store.ListDaggerheartDamageTypes(ctx)
	if err != nil {
		return err
	}
	catalog.damageTypes = types
	return nil
}

func (catalog *contentCatalog) listDomains(ctx context.Context) error {
	domains, err := catalog.store.ListDaggerheartDomains(ctx)
	if err != nil {
		return err
	}
	catalog.domains = domains
	return nil
}

func (catalog *contentCatalog) listDomainCards(ctx context.Context) error {
	cards, err := catalog.store.ListDaggerheartDomainCards(ctx)
	if err != nil {
		return err
	}
	catalog.domainCards = cards
	return nil
}

func (catalog *contentCatalog) listWeapons(ctx context.Context) error {
	weapons, err := catalog.store.ListDaggerheartWeapons(ctx)
	if err != nil {
		return err
	}
	catalog.weapons = weapons
	return nil
}

func (catalog *contentCatalog) listArmor(ctx context.Context) error {
	armor, err := catalog.store.ListDaggerheartArmor(ctx)
	if err != nil {
		return err
	}
	catalog.armor = armor
	return nil
}

func (catalog *contentCatalog) listItems(ctx context.Context) error {
	items, err := catalog.store.ListDaggerheartItems(ctx)
	if err != nil {
		return err
	}
	catalog.items = items
	return nil
}

func (catalog *contentCatalog) listEnvironments(ctx context.Context) error {
	environments, err := catalog.store.ListDaggerheartEnvironments(ctx)
	if err != nil {
		return err
	}
	catalog.environments = environments
	return nil
}

func (catalog *contentCatalog) localizeClasses(ctx context.Context) error {
	return localizeClasses(ctx, catalog.store, catalog.locale, catalog.classes)
}

func (catalog *contentCatalog) localizeSubclasses(ctx context.Context) error {
	return localizeSubclasses(ctx, catalog.store, catalog.locale, catalog.subclasses)
}

func (catalog *contentCatalog) localizeHeritages(ctx context.Context) error {
	return localizeHeritages(ctx, catalog.store, catalog.locale, catalog.heritages)
}

func (catalog *contentCatalog) localizeExperiences(ctx context.Context) error {
	return localizeExperiences(ctx, catalog.store, catalog.locale, catalog.experiences)
}

func (catalog *contentCatalog) localizeAdversaries(ctx context.Context) error {
	return localizeAdversaries(ctx, catalog.store, catalog.locale, catalog.adversaries)
}

func (catalog *contentCatalog) localizeBeastforms(ctx context.Context) error {
	return localizeBeastforms(ctx, catalog.store, catalog.locale, catalog.beastforms)
}

func (catalog *contentCatalog) localizeCompanionExperiences(ctx context.Context) error {
	return localizeCompanionExperiences(ctx, catalog.store, catalog.locale, catalog.companionExperiences)
}

func (catalog *contentCatalog) localizeLootEntries(ctx context.Context) error {
	return localizeLootEntries(ctx, catalog.store, catalog.locale, catalog.lootEntries)
}

func (catalog *contentCatalog) localizeDamageTypes(ctx context.Context) error {
	return localizeDamageTypes(ctx, catalog.store, catalog.locale, catalog.damageTypes)
}

func (catalog *contentCatalog) localizeDomains(ctx context.Context) error {
	return localizeDomains(ctx, catalog.store, catalog.locale, catalog.domains)
}

func (catalog *contentCatalog) localizeDomainCards(ctx context.Context) error {
	return localizeDomainCards(ctx, catalog.store, catalog.locale, catalog.domainCards)
}

func (catalog *contentCatalog) localizeWeapons(ctx context.Context) error {
	return localizeWeapons(ctx, catalog.store, catalog.locale, catalog.weapons)
}

func (catalog *contentCatalog) localizeArmor(ctx context.Context) error {
	return localizeArmor(ctx, catalog.store, catalog.locale, catalog.armor)
}

func (catalog *contentCatalog) localizeItems(ctx context.Context) error {
	return localizeItems(ctx, catalog.store, catalog.locale, catalog.items)
}

func (catalog *contentCatalog) localizeEnvironments(ctx context.Context) error {
	return localizeEnvironments(ctx, catalog.store, catalog.locale, catalog.environments)
}
