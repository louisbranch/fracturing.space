package contenttransport

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

// daggerheartAssetMapContent stages the catalog rows needed to resolve entity
// assets without exposing store operations to the transport entrypoint.
type daggerheartAssetMapContent struct {
	classes      []contentstore.DaggerheartClass
	subclasses   []contentstore.DaggerheartSubclass
	heritages    []contentstore.DaggerheartHeritage
	domains      []contentstore.DaggerheartDomain
	domainCards  []contentstore.DaggerheartDomainCard
	adversaries  []contentstore.DaggerheartAdversaryEntry
	environments []contentstore.DaggerheartEnvironment
	weapons      []contentstore.DaggerheartWeapon
	armor        []contentstore.DaggerheartArmor
	items        []contentstore.DaggerheartItem
}

// loadDaggerheartAssetMapContent fetches the catalog families that participate
// in published asset selection.
func loadDaggerheartAssetMapContent(ctx context.Context, store contentstore.DaggerheartContentReadStore) (daggerheartAssetMapContent, error) {
	classes, err := store.ListDaggerheartClasses(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list classes: %w", err)
	}
	subclasses, err := store.ListDaggerheartSubclasses(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list subclasses: %w", err)
	}
	heritages, err := store.ListDaggerheartHeritages(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list heritages: %w", err)
	}
	domains, err := store.ListDaggerheartDomains(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list domains: %w", err)
	}
	domainCards, err := store.ListDaggerheartDomainCards(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list domain cards: %w", err)
	}
	adversaries, err := store.ListDaggerheartAdversaryEntries(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list adversaries: %w", err)
	}
	environments, err := store.ListDaggerheartEnvironments(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list environments: %w", err)
	}
	weapons, err := store.ListDaggerheartWeapons(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list weapons: %w", err)
	}
	armor, err := store.ListDaggerheartArmor(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list armor: %w", err)
	}
	items, err := store.ListDaggerheartItems(ctx)
	if err != nil {
		return daggerheartAssetMapContent{}, fmt.Errorf("list items: %w", err)
	}

	return daggerheartAssetMapContent{
		classes:      classes,
		subclasses:   subclasses,
		heritages:    heritages,
		domains:      domains,
		domainCards:  domainCards,
		adversaries:  adversaries,
		environments: environments,
		weapons:      weapons,
		armor:        armor,
		items:        items,
	}, nil
}
