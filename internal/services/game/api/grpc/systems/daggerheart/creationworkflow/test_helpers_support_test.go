package creationworkflow

import (
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type testContentStore struct {
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
}

func newTestContentStore() *testContentStore {
	return &testContentStore{
		classes: map[string]contentstore.DaggerheartClass{
			"class-1": {
				ID:              "class-1",
				Name:            "Guardian",
				DomainIDs:       []string{"domain-1", "domain-2"},
				StartingHP:      7,
				StartingEvasion: 11,
			},
		},
		subclasses: map[string]contentstore.DaggerheartSubclass{
			"subclass-1": {ID: "subclass-1", Name: "Bulwark", ClassID: "class-1"},
		},
		heritages: map[string]contentstore.DaggerheartHeritage{
			"ancestry-1":  {ID: "ancestry-1", Name: "Elf", Kind: "ancestry"},
			"community-1": {ID: "community-1", Name: "Highborne", Kind: "community"},
		},
		experiences: map[string]contentstore.DaggerheartExperienceEntry{
			"experience-1": {ID: "experience-1", Name: "Archivist"},
		},
		adversaries: map[string]contentstore.DaggerheartAdversaryEntry{
			"adversary-1": {ID: "adversary-1", Name: "Goblin"},
		},
		beastforms: map[string]contentstore.DaggerheartBeastformEntry{
			"beastform-1": {ID: "beastform-1", Name: "Wolf"},
		},
		companionExperiences: map[string]contentstore.DaggerheartCompanionExperienceEntry{
			"companion-1": {ID: "companion-1", Name: "Guard"},
		},
		lootEntries: map[string]contentstore.DaggerheartLootEntry{
			"loot-1": {ID: "loot-1", Name: "Gold"},
		},
		damageTypes: map[string]contentstore.DaggerheartDamageTypeEntry{
			"damage-type-1": {ID: "damage-type-1", Name: "Fire"},
		},
		domains: map[string]contentstore.DaggerheartDomain{
			"domain-1": {ID: "domain-1", Name: "Valor"},
			"domain-2": {ID: "domain-2", Name: "Grace"},
			"domain-3": {ID: "domain-3", Name: "Bone"},
		},
		domainCards: map[string]contentstore.DaggerheartDomainCard{
			"card-1": {ID: "card-1", Name: "Aegis", DomainID: "domain-1", Level: 1},
			"card-2": {ID: "card-2", Name: "Stand Firm", DomainID: "domain-2", Level: 1},
			"card-3": {ID: "card-3", Name: "Wrong Domain", DomainID: "domain-3", Level: 1},
		},
		weapons: map[string]contentstore.DaggerheartWeapon{
			"weapon-primary-1":   {ID: "weapon-primary-1", Name: "Longsword", Category: "primary", Tier: 1, Burden: 1},
			"weapon-secondary-1": {ID: "weapon-secondary-1", Name: "Dagger", Category: "secondary", Tier: 1, Burden: 1},
			"weapon-heavy-1":     {ID: "weapon-heavy-1", Name: "Greatsword", Category: "primary", Tier: 1, Burden: 2},
		},
		armor: map[string]contentstore.DaggerheartArmor{
			"armor-1": {ID: "armor-1", Name: "Chain Mail", Tier: 1, ArmorScore: 2, BaseMajorThreshold: 7, BaseSevereThreshold: 13},
		},
		items: map[string]contentstore.DaggerheartItem{
			daggerheart.StartingPotionMinorHealthID:  {ID: daggerheart.StartingPotionMinorHealthID, Name: "Minor Health Potion"},
			daggerheart.StartingPotionMinorStaminaID: {ID: daggerheart.StartingPotionMinorStaminaID, Name: "Minor Stamina Potion"},
		},
		environments: map[string]contentstore.DaggerheartEnvironment{
			"environment-1": {ID: "environment-1", Name: "Forest"},
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
