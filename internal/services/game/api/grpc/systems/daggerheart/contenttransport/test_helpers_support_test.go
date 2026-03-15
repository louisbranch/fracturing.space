package contenttransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	got := status.Code(err)
	if got != want {
		t.Fatalf("status code = %v, want %v (err=%v)", got, want, err)
	}
}

type fakeContentStore struct {
	classes              map[string]contentstore.DaggerheartClass
	subclasses           map[string]contentstore.DaggerheartSubclass
	heritages            map[string]contentstore.DaggerheartHeritage
	experiences          map[string]contentstore.DaggerheartExperienceEntry
	adversaries          map[string]contentstore.DaggerheartAdversaryEntry
	beastforms           map[string]contentstore.DaggerheartBeastformEntry
	companionExperiences map[string]contentstore.DaggerheartCompanionExperienceEntry
	lootEntries          map[string]contentstore.DaggerheartLootEntry
	damageTypes          map[string]contentstore.DaggerheartDamageTypeEntry
	domains              map[string]contentstore.DaggerheartDomain
	domainCards          map[string]contentstore.DaggerheartDomainCard
	weapons              map[string]contentstore.DaggerheartWeapon
	armor                map[string]contentstore.DaggerheartArmor
	items                map[string]contentstore.DaggerheartItem
	environments         map[string]contentstore.DaggerheartEnvironment
	contentStrings       []contentstore.DaggerheartContentString
}

func newFakeContentStore() *fakeContentStore {
	return &fakeContentStore{
		classes: map[string]contentstore.DaggerheartClass{
			"class-1": {ID: "class-1", Name: "Guardian"},
		},
		subclasses: map[string]contentstore.DaggerheartSubclass{
			"sub-1": {ID: "sub-1", Name: "Bladeweaver", ClassID: "class-1"},
		},
		heritages: map[string]contentstore.DaggerheartHeritage{
			"her-1": {ID: "her-1", Name: "Elf", Kind: "ancestry"},
		},
		experiences: map[string]contentstore.DaggerheartExperienceEntry{
			"exp-1": {ID: "exp-1", Name: "Wanderer"},
		},
		adversaries: map[string]contentstore.DaggerheartAdversaryEntry{
			"adv-1": {ID: "adv-1", Name: "Goblin"},
		},
		beastforms: map[string]contentstore.DaggerheartBeastformEntry{
			"beast-1": {ID: "beast-1", Name: "Wolf"},
		},
		companionExperiences: map[string]contentstore.DaggerheartCompanionExperienceEntry{
			"cexp-1": {ID: "cexp-1", Name: "Guard"},
		},
		lootEntries: map[string]contentstore.DaggerheartLootEntry{
			"loot-1": {ID: "loot-1", Name: "Gold"},
		},
		damageTypes: map[string]contentstore.DaggerheartDamageTypeEntry{
			"dt-1": {ID: "dt-1", Name: "Fire"},
		},
		domains: map[string]contentstore.DaggerheartDomain{
			"dom-1": {ID: "dom-1", Name: "Valor"},
		},
		domainCards: map[string]contentstore.DaggerheartDomainCard{
			"card-1": {ID: "card-1", Name: "Fireball", DomainID: "dom-1"},
		},
		weapons: map[string]contentstore.DaggerheartWeapon{
			"weapon-1": {ID: "weapon-1", Name: "Blade"},
		},
		armor: map[string]contentstore.DaggerheartArmor{
			"armor-1": {ID: "armor-1", Name: "Chain Mail"},
		},
		items: map[string]contentstore.DaggerheartItem{
			"item-1": {ID: "item-1", Name: "Potion", Kind: "equipment", Rarity: "common"},
		},
		environments: map[string]contentstore.DaggerheartEnvironment{
			"env-1": {ID: "env-1", Name: "Forest", Type: "social"},
		},
	}
}

func mapGet[T any](items map[string]T, id string) (T, error) {
	item, ok := items[id]
	if !ok {
		var zero T
		return zero, storage.ErrNotFound
	}
	return item, nil
}

func mapList[T any](items map[string]T) ([]T, error) {
	result := make([]T, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result, nil
}
