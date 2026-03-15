package contenttransport

import "context"

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
