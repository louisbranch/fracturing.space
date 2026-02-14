package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestDaggerheartClassCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartClass{
		ID:              "class-warrior",
		Name:            "Warrior",
		StartingEvasion: 8,
		StartingHP:      18,
		StartingItems:   []string{"Sword", "Shield", "Torch"},
		Features: []storage.DaggerheartFeature{
			{ID: "feat-1", Name: "Mighty Strike", Description: "Deal extra damage", Level: 1},
			{ID: "feat-2", Name: "Shield Wall", Description: "Block attacks", Level: 3},
		},
		HopeFeature: storage.DaggerheartHopeFeature{
			Name:        "Battle Cry",
			Description: "Inspire allies",
			HopeCost:    2,
		},
		DomainIDs: []string{"dom-blade", "dom-valor"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.PutDaggerheartClass(context.Background(), expected); err != nil {
		t.Fatalf("put class: %v", err)
	}

	got, err := store.GetDaggerheartClass(context.Background(), "class-warrior")
	if err != nil {
		t.Fatalf("get class: %v", err)
	}
	if got.ID != expected.ID || got.Name != expected.Name {
		t.Fatalf("expected identity to match")
	}
	if got.StartingEvasion != expected.StartingEvasion || got.StartingHP != expected.StartingHP {
		t.Fatalf("expected starting stats to match")
	}
	if len(got.StartingItems) != 3 || got.StartingItems[0] != "Sword" {
		t.Fatalf("expected starting items to match, got %v", got.StartingItems)
	}
	if len(got.Features) != 2 || got.Features[0].Name != "Mighty Strike" {
		t.Fatalf("expected features to match")
	}
	if got.HopeFeature.Name != "Battle Cry" || got.HopeFeature.HopeCost != 2 {
		t.Fatalf("expected hope feature to match")
	}
	if len(got.DomainIDs) != 2 || got.DomainIDs[0] != "dom-blade" {
		t.Fatalf("expected domain ids to match, got %v", got.DomainIDs)
	}

	list, err := store.ListDaggerheartClasses(context.Background())
	if err != nil {
		t.Fatalf("list classes: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 class, got %d", len(list))
	}

	if err := store.DeleteDaggerheartClass(context.Background(), "class-warrior"); err != nil {
		t.Fatalf("delete class: %v", err)
	}
	_, err = store.GetDaggerheartClass(context.Background(), "class-warrior")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestDaggerheartSubclassCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartSubclass{
		ID:             "sub-berserker",
		Name:           "Berserker",
		SpellcastTrait: "Strength",
		FoundationFeatures: []storage.DaggerheartFeature{
			{ID: "ff-1", Name: "Rage", Description: "Enter a rage", Level: 1},
		},
		SpecializationFeatures: []storage.DaggerheartFeature{
			{ID: "sf-1", Name: "Frenzy", Description: "Attack multiple times", Level: 5},
		},
		MasteryFeatures: []storage.DaggerheartFeature{
			{ID: "mf-1", Name: "Unstoppable", Description: "Cannot be stopped", Level: 9},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.PutDaggerheartSubclass(context.Background(), expected); err != nil {
		t.Fatalf("put subclass: %v", err)
	}

	got, err := store.GetDaggerheartSubclass(context.Background(), "sub-berserker")
	if err != nil {
		t.Fatalf("get subclass: %v", err)
	}
	if got.SpellcastTrait != "Strength" {
		t.Fatalf("expected spellcast trait %q, got %q", "Strength", got.SpellcastTrait)
	}
	if len(got.FoundationFeatures) != 1 || got.FoundationFeatures[0].Name != "Rage" {
		t.Fatalf("expected foundation features to match")
	}
	if len(got.SpecializationFeatures) != 1 || got.SpecializationFeatures[0].Name != "Frenzy" {
		t.Fatalf("expected specialization features to match")
	}
	if len(got.MasteryFeatures) != 1 || got.MasteryFeatures[0].Name != "Unstoppable" {
		t.Fatalf("expected mastery features to match")
	}

	list, err := store.ListDaggerheartSubclasses(context.Background())
	if err != nil {
		t.Fatalf("list subclasses: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 subclass, got %d", len(list))
	}

	if err := store.DeleteDaggerheartSubclass(context.Background(), "sub-berserker"); err != nil {
		t.Fatalf("delete subclass: %v", err)
	}
	_, err = store.GetDaggerheartSubclass(context.Background(), "sub-berserker")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartHeritageCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartHeritage{
		ID:   "her-elf",
		Name: "Elf",
		Kind: "ancestry",
		Features: []storage.DaggerheartFeature{
			{ID: "hf-1", Name: "Keen Senses", Description: "Advantage on perception", Level: 1},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.PutDaggerheartHeritage(context.Background(), expected); err != nil {
		t.Fatalf("put heritage: %v", err)
	}

	got, err := store.GetDaggerheartHeritage(context.Background(), "her-elf")
	if err != nil {
		t.Fatalf("get heritage: %v", err)
	}
	if got.Kind != "ancestry" {
		t.Fatalf("expected kind %q, got %q", "ancestry", got.Kind)
	}
	if len(got.Features) != 1 || got.Features[0].Name != "Keen Senses" {
		t.Fatalf("expected features to match")
	}

	list, err := store.ListDaggerheartHeritages(context.Background())
	if err != nil {
		t.Fatalf("list heritages: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 heritage, got %d", len(list))
	}

	if err := store.DeleteDaggerheartHeritage(context.Background(), "her-elf"); err != nil {
		t.Fatalf("delete heritage: %v", err)
	}
	_, err = store.GetDaggerheartHeritage(context.Background(), "her-elf")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartExperienceCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartExperienceEntry{
		ID:          "exp-scholar",
		Name:        "Scholar",
		Description: "Studied at the academy",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutDaggerheartExperience(context.Background(), expected); err != nil {
		t.Fatalf("put experience: %v", err)
	}

	got, err := store.GetDaggerheartExperience(context.Background(), "exp-scholar")
	if err != nil {
		t.Fatalf("get experience: %v", err)
	}
	if got.Name != expected.Name || got.Description != expected.Description {
		t.Fatalf("expected name/description to match")
	}

	list, err := store.ListDaggerheartExperiences(context.Background())
	if err != nil {
		t.Fatalf("list experiences: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(list))
	}

	if err := store.DeleteDaggerheartExperience(context.Background(), "exp-scholar"); err != nil {
		t.Fatalf("delete experience: %v", err)
	}
	_, err = store.GetDaggerheartExperience(context.Background(), "exp-scholar")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartAdversaryEntryCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartAdversaryEntry{
		ID:              "adv-orc",
		Name:            "Orc Warrior",
		Tier:            2,
		Role:            "bruiser",
		Description:     "A fierce orc",
		Motives:         "Conquest",
		Difficulty:      5,
		MajorThreshold:  8,
		SevereThreshold: 15,
		HP:              12,
		Stress:          4,
		Armor:           2,
		AttackModifier:  3,
		StandardAttack: storage.DaggerheartAdversaryAttack{
			Name:        "Greataxe",
			Range:       "melee",
			DamageDice:  []storage.DaggerheartDamageDie{{Sides: 8, Count: 2}},
			DamageBonus: 3,
			DamageType:  "physical",
		},
		Experiences: []storage.DaggerheartAdversaryExperience{
			{Name: "Intimidation", Modifier: 4},
		},
		Features: []storage.DaggerheartAdversaryFeature{
			{ID: "af-1", Name: "Cleave", Kind: "action", Description: "Attack two targets", CostType: "action", Cost: 1},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.PutDaggerheartAdversaryEntry(context.Background(), expected); err != nil {
		t.Fatalf("put adversary entry: %v", err)
	}

	got, err := store.GetDaggerheartAdversaryEntry(context.Background(), "adv-orc")
	if err != nil {
		t.Fatalf("get adversary entry: %v", err)
	}
	if got.Name != expected.Name || got.Tier != expected.Tier || got.Role != expected.Role {
		t.Fatalf("expected identity/tier/role to match")
	}
	if got.HP != expected.HP || got.Stress != expected.Stress || got.Armor != expected.Armor {
		t.Fatalf("expected combat stats to match")
	}
	if got.AttackModifier != expected.AttackModifier {
		t.Fatalf("expected attack modifier %d, got %d", expected.AttackModifier, got.AttackModifier)
	}
	if got.StandardAttack.Name != "Greataxe" || got.StandardAttack.DamageBonus != 3 {
		t.Fatalf("expected standard attack to match")
	}
	if len(got.StandardAttack.DamageDice) != 1 || got.StandardAttack.DamageDice[0].Sides != 8 {
		t.Fatalf("expected damage dice to match")
	}
	if len(got.Experiences) != 1 || got.Experiences[0].Name != "Intimidation" {
		t.Fatalf("expected experiences to match")
	}
	if len(got.Features) != 1 || got.Features[0].Name != "Cleave" {
		t.Fatalf("expected features to match")
	}

	list, err := store.ListDaggerheartAdversaryEntries(context.Background())
	if err != nil {
		t.Fatalf("list adversary entries: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 adversary entry, got %d", len(list))
	}

	if err := store.DeleteDaggerheartAdversaryEntry(context.Background(), "adv-orc"); err != nil {
		t.Fatalf("delete adversary entry: %v", err)
	}
	_, err = store.GetDaggerheartAdversaryEntry(context.Background(), "adv-orc")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartBeastformCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartBeastformEntry{
		ID:           "beast-wolf",
		Name:         "Dire Wolf",
		Tier:         1,
		Examples:     "Wolf, Dog",
		Trait:        "Agility",
		TraitBonus:   2,
		EvasionBonus: 1,
		Attack: storage.DaggerheartBeastformAttack{
			Range:       "melee",
			Trait:       "Agility",
			DamageDice:  []storage.DaggerheartDamageDie{{Sides: 6, Count: 1}},
			DamageBonus: 2,
			DamageType:  "physical",
		},
		Advantages: []string{"Pack Tactics", "Keen Smell"},
		Features: []storage.DaggerheartBeastformFeature{
			{ID: "bf-1", Name: "Pounce", Description: "Leap and attack"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.PutDaggerheartBeastform(context.Background(), expected); err != nil {
		t.Fatalf("put beastform: %v", err)
	}

	got, err := store.GetDaggerheartBeastform(context.Background(), "beast-wolf")
	if err != nil {
		t.Fatalf("get beastform: %v", err)
	}
	if got.Name != expected.Name || got.Tier != expected.Tier {
		t.Fatalf("expected name/tier to match")
	}
	if got.Attack.Range != "melee" || got.Attack.DamageBonus != 2 {
		t.Fatalf("expected attack to match")
	}
	if len(got.Advantages) != 2 || got.Advantages[0] != "Pack Tactics" {
		t.Fatalf("expected advantages to match, got %v", got.Advantages)
	}
	if len(got.Features) != 1 || got.Features[0].Name != "Pounce" {
		t.Fatalf("expected features to match")
	}

	list, err := store.ListDaggerheartBeastforms(context.Background())
	if err != nil {
		t.Fatalf("list beastforms: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 beastform, got %d", len(list))
	}

	if err := store.DeleteDaggerheartBeastform(context.Background(), "beast-wolf"); err != nil {
		t.Fatalf("delete beastform: %v", err)
	}
	_, err = store.GetDaggerheartBeastform(context.Background(), "beast-wolf")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartCompanionExperienceCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartCompanionExperienceEntry{
		ID:          "cexp-loyal",
		Name:        "Loyal Companion",
		Description: "A steadfast ally",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutDaggerheartCompanionExperience(context.Background(), expected); err != nil {
		t.Fatalf("put companion experience: %v", err)
	}

	got, err := store.GetDaggerheartCompanionExperience(context.Background(), "cexp-loyal")
	if err != nil {
		t.Fatalf("get companion experience: %v", err)
	}
	if got.Name != expected.Name || got.Description != expected.Description {
		t.Fatalf("expected name/description to match")
	}

	list, err := store.ListDaggerheartCompanionExperiences(context.Background())
	if err != nil {
		t.Fatalf("list companion experiences: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 companion experience, got %d", len(list))
	}

	if err := store.DeleteDaggerheartCompanionExperience(context.Background(), "cexp-loyal"); err != nil {
		t.Fatalf("delete companion experience: %v", err)
	}
	_, err = store.GetDaggerheartCompanionExperience(context.Background(), "cexp-loyal")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartLootEntryCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartLootEntry{
		ID:          "loot-gem",
		Name:        "Ruby Gem",
		Roll:        7,
		Description: "A precious ruby",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutDaggerheartLootEntry(context.Background(), expected); err != nil {
		t.Fatalf("put loot entry: %v", err)
	}

	got, err := store.GetDaggerheartLootEntry(context.Background(), "loot-gem")
	if err != nil {
		t.Fatalf("get loot entry: %v", err)
	}
	if got.Name != expected.Name || got.Roll != expected.Roll {
		t.Fatalf("expected name/roll to match")
	}
	if got.Description != expected.Description {
		t.Fatalf("expected description to match")
	}

	list, err := store.ListDaggerheartLootEntries(context.Background())
	if err != nil {
		t.Fatalf("list loot entries: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 loot entry, got %d", len(list))
	}

	if err := store.DeleteDaggerheartLootEntry(context.Background(), "loot-gem"); err != nil {
		t.Fatalf("delete loot entry: %v", err)
	}
	_, err = store.GetDaggerheartLootEntry(context.Background(), "loot-gem")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartDamageTypeCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartDamageTypeEntry{
		ID:          "dt-fire",
		Name:        "Fire",
		Description: "Burns things",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutDaggerheartDamageType(context.Background(), expected); err != nil {
		t.Fatalf("put damage type: %v", err)
	}

	got, err := store.GetDaggerheartDamageType(context.Background(), "dt-fire")
	if err != nil {
		t.Fatalf("get damage type: %v", err)
	}
	if got.Name != expected.Name || got.Description != expected.Description {
		t.Fatalf("expected name/description to match")
	}

	list, err := store.ListDaggerheartDamageTypes(context.Background())
	if err != nil {
		t.Fatalf("list damage types: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 damage type, got %d", len(list))
	}

	if err := store.DeleteDaggerheartDamageType(context.Background(), "dt-fire"); err != nil {
		t.Fatalf("delete damage type: %v", err)
	}
	_, err = store.GetDaggerheartDamageType(context.Background(), "dt-fire")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartDomainCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartDomain{
		ID:          "dom-blade",
		Name:        "Blade",
		Description: "The domain of swords and combat",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutDaggerheartDomain(context.Background(), expected); err != nil {
		t.Fatalf("put domain: %v", err)
	}

	got, err := store.GetDaggerheartDomain(context.Background(), "dom-blade")
	if err != nil {
		t.Fatalf("get domain: %v", err)
	}
	if got.Name != expected.Name || got.Description != expected.Description {
		t.Fatalf("expected name/description to match")
	}

	list, err := store.ListDaggerheartDomains(context.Background())
	if err != nil {
		t.Fatalf("list domains: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(list))
	}

	if err := store.DeleteDaggerheartDomain(context.Background(), "dom-blade"); err != nil {
		t.Fatalf("delete domain: %v", err)
	}
	_, err = store.GetDaggerheartDomain(context.Background(), "dom-blade")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartDomainCardCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	// Seed a domain first
	if err := store.PutDaggerheartDomain(context.Background(), storage.DaggerheartDomain{
		ID: "dom-arcana", Name: "Arcana", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed domain: %v", err)
	}

	expected := storage.DaggerheartDomainCard{
		ID:          "card-fireball",
		Name:        "Fireball",
		DomainID:    "dom-arcana",
		Level:       3,
		Type:        "spell",
		RecallCost:  2,
		UsageLimit:  "1/session",
		FeatureText: "Hurl a ball of fire dealing 3d6 damage",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutDaggerheartDomainCard(context.Background(), expected); err != nil {
		t.Fatalf("put domain card: %v", err)
	}

	got, err := store.GetDaggerheartDomainCard(context.Background(), "card-fireball")
	if err != nil {
		t.Fatalf("get domain card: %v", err)
	}
	if got.Name != expected.Name || got.DomainID != expected.DomainID {
		t.Fatalf("expected name/domain to match")
	}
	if got.Level != expected.Level || got.Type != expected.Type {
		t.Fatalf("expected level/type to match")
	}
	if got.RecallCost != expected.RecallCost || got.UsageLimit != expected.UsageLimit {
		t.Fatalf("expected recall cost/usage limit to match")
	}
	if got.FeatureText != expected.FeatureText {
		t.Fatalf("expected feature text to match")
	}

	// Add another card in a different domain (seed domain first for FK)
	if err := store.PutDaggerheartDomain(context.Background(), storage.DaggerheartDomain{
		ID: "dom-grace", Name: "Grace", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed dom-grace: %v", err)
	}
	if err := store.PutDaggerheartDomainCard(context.Background(), storage.DaggerheartDomainCard{
		ID: "card-heal", Name: "Heal", DomainID: "dom-grace", Level: 1, Type: "spell",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put second card: %v", err)
	}

	all, err := store.ListDaggerheartDomainCards(context.Background())
	if err != nil {
		t.Fatalf("list all domain cards: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 domain cards, got %d", len(all))
	}

	byDomain, err := store.ListDaggerheartDomainCardsByDomain(context.Background(), "dom-arcana")
	if err != nil {
		t.Fatalf("list domain cards by domain: %v", err)
	}
	if len(byDomain) != 1 || byDomain[0].ID != "card-fireball" {
		t.Fatalf("expected 1 card for dom-arcana, got %d", len(byDomain))
	}

	if err := store.DeleteDaggerheartDomainCard(context.Background(), "card-fireball"); err != nil {
		t.Fatalf("delete domain card: %v", err)
	}
	_, err = store.GetDaggerheartDomainCard(context.Background(), "card-fireball")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartWeaponCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartWeapon{
		ID:         "wpn-longsword",
		Name:       "Longsword",
		Category:   "martial",
		Tier:       1,
		Trait:      "Finesse",
		Range:      "melee",
		DamageDice: []storage.DaggerheartDamageDie{{Sides: 8, Count: 1}, {Sides: 6, Count: 1}},
		DamageType: "physical",
		Burden:     2,
		Feature:    "Versatile",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := store.PutDaggerheartWeapon(context.Background(), expected); err != nil {
		t.Fatalf("put weapon: %v", err)
	}

	got, err := store.GetDaggerheartWeapon(context.Background(), "wpn-longsword")
	if err != nil {
		t.Fatalf("get weapon: %v", err)
	}
	if got.Name != expected.Name || got.Category != expected.Category {
		t.Fatalf("expected name/category to match")
	}
	if got.Trait != expected.Trait || got.Range != expected.Range {
		t.Fatalf("expected trait/range to match")
	}
	if len(got.DamageDice) != 2 || got.DamageDice[0].Sides != 8 || got.DamageDice[1].Count != 1 {
		t.Fatalf("expected damage dice to match, got %v", got.DamageDice)
	}
	if got.DamageType != expected.DamageType || got.Burden != expected.Burden {
		t.Fatalf("expected damage type/burden to match")
	}

	list, err := store.ListDaggerheartWeapons(context.Background())
	if err != nil {
		t.Fatalf("list weapons: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 weapon, got %d", len(list))
	}

	if err := store.DeleteDaggerheartWeapon(context.Background(), "wpn-longsword"); err != nil {
		t.Fatalf("delete weapon: %v", err)
	}
	_, err = store.GetDaggerheartWeapon(context.Background(), "wpn-longsword")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartArmorCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartArmor{
		ID:                  "arm-chain",
		Name:                "Chain Mail",
		Tier:                2,
		BaseMajorThreshold:  7,
		BaseSevereThreshold: 14,
		ArmorScore:          3,
		Feature:             "Heavy",
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := store.PutDaggerheartArmor(context.Background(), expected); err != nil {
		t.Fatalf("put armor: %v", err)
	}

	got, err := store.GetDaggerheartArmor(context.Background(), "arm-chain")
	if err != nil {
		t.Fatalf("get armor: %v", err)
	}
	if got.Name != expected.Name || got.Tier != expected.Tier {
		t.Fatalf("expected name/tier to match")
	}
	if got.BaseMajorThreshold != expected.BaseMajorThreshold || got.BaseSevereThreshold != expected.BaseSevereThreshold {
		t.Fatalf("expected thresholds to match")
	}
	if got.ArmorScore != expected.ArmorScore {
		t.Fatalf("expected armor score %d, got %d", expected.ArmorScore, got.ArmorScore)
	}

	list, err := store.ListDaggerheartArmor(context.Background())
	if err != nil {
		t.Fatalf("list armor: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 armor, got %d", len(list))
	}

	if err := store.DeleteDaggerheartArmor(context.Background(), "arm-chain"); err != nil {
		t.Fatalf("delete armor: %v", err)
	}
	_, err = store.GetDaggerheartArmor(context.Background(), "arm-chain")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartItemCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartItem{
		ID:          "item-potion",
		Name:        "Healing Potion",
		Rarity:      "common",
		Kind:        "consumable",
		StackMax:    5,
		Description: "Restores health",
		EffectText:  "Heal 1d8+2 HP",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutDaggerheartItem(context.Background(), expected); err != nil {
		t.Fatalf("put item: %v", err)
	}

	got, err := store.GetDaggerheartItem(context.Background(), "item-potion")
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if got.Name != expected.Name || got.Rarity != expected.Rarity {
		t.Fatalf("expected name/rarity to match")
	}
	if got.Kind != expected.Kind || got.StackMax != expected.StackMax {
		t.Fatalf("expected kind/stack max to match")
	}
	if got.Description != expected.Description || got.EffectText != expected.EffectText {
		t.Fatalf("expected description/effect text to match")
	}

	list, err := store.ListDaggerheartItems(context.Background())
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 item, got %d", len(list))
	}

	if err := store.DeleteDaggerheartItem(context.Background(), "item-potion"); err != nil {
		t.Fatalf("delete item: %v", err)
	}
	_, err = store.GetDaggerheartItem(context.Background(), "item-potion")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartEnvironmentCRUD(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartEnvironment{
		ID:                    "env-dungeon",
		Name:                  "Dark Dungeon",
		Tier:                  2,
		Type:                  "underground",
		Difficulty:            6,
		Impulses:              []string{"Trap the intruders", "Collapse the passage"},
		PotentialAdversaryIDs: []string{"adv-goblin", "adv-skeleton"},
		Features: []storage.DaggerheartFeature{
			{ID: "ef-1", Name: "Darkness", Description: "Dim lighting everywhere", Level: 1},
		},
		Prompts:   []string{"What echoes in the dark?", "Who built this place?"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.PutDaggerheartEnvironment(context.Background(), expected); err != nil {
		t.Fatalf("put environment: %v", err)
	}

	got, err := store.GetDaggerheartEnvironment(context.Background(), "env-dungeon")
	if err != nil {
		t.Fatalf("get environment: %v", err)
	}
	if got.Name != expected.Name || got.Tier != expected.Tier {
		t.Fatalf("expected name/tier to match")
	}
	if got.Type != expected.Type || got.Difficulty != expected.Difficulty {
		t.Fatalf("expected type/difficulty to match")
	}
	if len(got.Impulses) != 2 || got.Impulses[0] != "Trap the intruders" {
		t.Fatalf("expected impulses to match, got %v", got.Impulses)
	}
	if len(got.PotentialAdversaryIDs) != 2 || got.PotentialAdversaryIDs[0] != "adv-goblin" {
		t.Fatalf("expected potential adversary ids to match, got %v", got.PotentialAdversaryIDs)
	}
	if len(got.Features) != 1 || got.Features[0].Name != "Darkness" {
		t.Fatalf("expected features to match")
	}
	if len(got.Prompts) != 2 || got.Prompts[0] != "What echoes in the dark?" {
		t.Fatalf("expected prompts to match, got %v", got.Prompts)
	}

	list, err := store.ListDaggerheartEnvironments(context.Background())
	if err != nil {
		t.Fatalf("list environments: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(list))
	}

	if err := store.DeleteDaggerheartEnvironment(context.Background(), "env-dungeon"); err != nil {
		t.Fatalf("delete environment: %v", err)
	}
	_, err = store.GetDaggerheartEnvironment(context.Background(), "env-dungeon")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestDaggerheartContentString(t *testing.T) {
	store := openTestContentStore(t)
	now := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)

	expected := storage.DaggerheartContentString{
		ContentID:   "class-warrior",
		ContentType: "class",
		Field:       "name",
		Locale:      "en-US",
		Text:        "Warrior",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutDaggerheartContentString(context.Background(), expected); err != nil {
		t.Fatalf("put content string: %v", err)
	}

	// Test auto-timestamp when zero
	auto := storage.DaggerheartContentString{
		ContentID:   "class-warrior",
		ContentType: "class",
		Field:       "description",
		Locale:      "en-US",
		Text:        "A mighty warrior",
	}
	if err := store.PutDaggerheartContentString(context.Background(), auto); err != nil {
		t.Fatalf("put content string with auto timestamp: %v", err)
	}
}
