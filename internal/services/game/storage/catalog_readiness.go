package storage

import (
	"context"
	"fmt"
)

// DaggerheartCatalogSection identifies one required section of Daggerheart
// catalog content.
type DaggerheartCatalogSection string

const (
	// DaggerheartCatalogSectionClasses identifies class catalog rows.
	DaggerheartCatalogSectionClasses DaggerheartCatalogSection = "classes"
	// DaggerheartCatalogSectionSubclasses identifies subclass catalog rows.
	DaggerheartCatalogSectionSubclasses DaggerheartCatalogSection = "subclasses"
	// DaggerheartCatalogSectionHeritages identifies heritage catalog rows.
	DaggerheartCatalogSectionHeritages DaggerheartCatalogSection = "heritages"
	// DaggerheartCatalogSectionExperiences identifies experience catalog rows.
	DaggerheartCatalogSectionExperiences DaggerheartCatalogSection = "experiences"
	// DaggerheartCatalogSectionAdversaryEntries identifies adversary catalog rows.
	DaggerheartCatalogSectionAdversaryEntries DaggerheartCatalogSection = "adversary_entries"
	// DaggerheartCatalogSectionBeastforms identifies beastform catalog rows.
	DaggerheartCatalogSectionBeastforms DaggerheartCatalogSection = "beastforms"
	// DaggerheartCatalogSectionCompanionExperiences identifies companion experience rows.
	DaggerheartCatalogSectionCompanionExperiences DaggerheartCatalogSection = "companion_experiences"
	// DaggerheartCatalogSectionLootEntries identifies loot-entry catalog rows.
	DaggerheartCatalogSectionLootEntries DaggerheartCatalogSection = "loot_entries"
	// DaggerheartCatalogSectionDamageTypes identifies damage-type catalog rows.
	DaggerheartCatalogSectionDamageTypes DaggerheartCatalogSection = "damage_types"
	// DaggerheartCatalogSectionDomains identifies domain catalog rows.
	DaggerheartCatalogSectionDomains DaggerheartCatalogSection = "domains"
	// DaggerheartCatalogSectionDomainCards identifies domain-card catalog rows.
	DaggerheartCatalogSectionDomainCards DaggerheartCatalogSection = "domain_cards"
	// DaggerheartCatalogSectionWeapons identifies weapon catalog rows.
	DaggerheartCatalogSectionWeapons DaggerheartCatalogSection = "weapons"
	// DaggerheartCatalogSectionArmor identifies armor catalog rows.
	DaggerheartCatalogSectionArmor DaggerheartCatalogSection = "armor"
	// DaggerheartCatalogSectionItems identifies item catalog rows.
	DaggerheartCatalogSectionItems DaggerheartCatalogSection = "items"
	// DaggerheartCatalogSectionEnvironments identifies environment catalog rows.
	DaggerheartCatalogSectionEnvironments DaggerheartCatalogSection = "environments"
)

// DaggerheartCatalogReadiness reports whether all required Daggerheart catalog
// sections are populated.
type DaggerheartCatalogReadiness struct {
	Ready           bool
	MissingSections []DaggerheartCatalogSection
}

// MissingSectionNames returns missing section identifiers in deterministic order.
func (r DaggerheartCatalogReadiness) MissingSectionNames() []string {
	names := make([]string, 0, len(r.MissingSections))
	for _, section := range r.MissingSections {
		names = append(names, string(section))
	}
	return names
}

// DaggerheartCatalogReadinessStore exposes only the list operations needed to
// evaluate whether the Daggerheart content catalog is populated.
type DaggerheartCatalogReadinessStore interface {
	ListDaggerheartClasses(ctx context.Context) ([]DaggerheartClass, error)
	ListDaggerheartSubclasses(ctx context.Context) ([]DaggerheartSubclass, error)
	ListDaggerheartHeritages(ctx context.Context) ([]DaggerheartHeritage, error)
	ListDaggerheartExperiences(ctx context.Context) ([]DaggerheartExperienceEntry, error)
	ListDaggerheartAdversaryEntries(ctx context.Context) ([]DaggerheartAdversaryEntry, error)
	ListDaggerheartBeastforms(ctx context.Context) ([]DaggerheartBeastformEntry, error)
	ListDaggerheartCompanionExperiences(ctx context.Context) ([]DaggerheartCompanionExperienceEntry, error)
	ListDaggerheartLootEntries(ctx context.Context) ([]DaggerheartLootEntry, error)
	ListDaggerheartDamageTypes(ctx context.Context) ([]DaggerheartDamageTypeEntry, error)
	ListDaggerheartDomains(ctx context.Context) ([]DaggerheartDomain, error)
	ListDaggerheartDomainCards(ctx context.Context) ([]DaggerheartDomainCard, error)
	ListDaggerheartWeapons(ctx context.Context) ([]DaggerheartWeapon, error)
	ListDaggerheartArmor(ctx context.Context) ([]DaggerheartArmor, error)
	ListDaggerheartItems(ctx context.Context) ([]DaggerheartItem, error)
	ListDaggerheartEnvironments(ctx context.Context) ([]DaggerheartEnvironment, error)
}

// EvaluateDaggerheartCatalogReadiness checks every required Daggerheart catalog
// section and reports whether each section has at least one row.
func EvaluateDaggerheartCatalogReadiness(ctx context.Context, store DaggerheartCatalogReadinessStore) (DaggerheartCatalogReadiness, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if store == nil {
		return DaggerheartCatalogReadiness{}, fmt.Errorf("catalog readiness store is required")
	}

	checks := []struct {
		section DaggerheartCatalogSection
		count   func(context.Context) (int, error)
	}{
		{
			section: DaggerheartCatalogSectionClasses,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartClasses(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionSubclasses,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartSubclasses(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionHeritages,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartHeritages(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionExperiences,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartExperiences(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionAdversaryEntries,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartAdversaryEntries(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionBeastforms,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartBeastforms(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionCompanionExperiences,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartCompanionExperiences(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionLootEntries,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartLootEntries(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionDamageTypes,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartDamageTypes(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionDomains,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartDomains(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionDomainCards,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartDomainCards(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionWeapons,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartWeapons(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionArmor,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartArmor(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionItems,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartItems(ctx)
				return len(items), err
			},
		},
		{
			section: DaggerheartCatalogSectionEnvironments,
			count: func(ctx context.Context) (int, error) {
				items, err := store.ListDaggerheartEnvironments(ctx)
				return len(items), err
			},
		},
	}

	missing := make([]DaggerheartCatalogSection, 0, len(checks))
	for _, check := range checks {
		count, err := check.count(ctx)
		if err != nil {
			return DaggerheartCatalogReadiness{}, fmt.Errorf("list daggerheart %s: %w", check.section, err)
		}
		if count == 0 {
			missing = append(missing, check.section)
		}
	}

	return DaggerheartCatalogReadiness{
		Ready:           len(missing) == 0,
		MissingSections: missing,
	}, nil
}
