package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// fakeContentStore records Put calls without persisting anything.
type fakeContentStore struct {
	classes              map[string]storage.DaggerheartClass
	subclasses           map[string]storage.DaggerheartSubclass
	heritages            map[string]storage.DaggerheartHeritage
	experiences          map[string]storage.DaggerheartExperienceEntry
	adversaries          map[string]storage.DaggerheartAdversaryEntry
	beastforms           map[string]storage.DaggerheartBeastformEntry
	companionExperiences map[string]storage.DaggerheartCompanionExperienceEntry
	lootEntries          map[string]storage.DaggerheartLootEntry
	damageTypes          map[string]storage.DaggerheartDamageTypeEntry
	domains              map[string]storage.DaggerheartDomain
	domainCards          map[string]storage.DaggerheartDomainCard
	weapons              map[string]storage.DaggerheartWeapon
	armor                map[string]storage.DaggerheartArmor
	items                map[string]storage.DaggerheartItem
	environments         map[string]storage.DaggerheartEnvironment
	contentStrings       []storage.DaggerheartContentString
}

func newFakeContentStore() *fakeContentStore {
	return &fakeContentStore{
		classes:              make(map[string]storage.DaggerheartClass),
		subclasses:           make(map[string]storage.DaggerheartSubclass),
		heritages:            make(map[string]storage.DaggerheartHeritage),
		experiences:          make(map[string]storage.DaggerheartExperienceEntry),
		adversaries:          make(map[string]storage.DaggerheartAdversaryEntry),
		beastforms:           make(map[string]storage.DaggerheartBeastformEntry),
		companionExperiences: make(map[string]storage.DaggerheartCompanionExperienceEntry),
		lootEntries:          make(map[string]storage.DaggerheartLootEntry),
		damageTypes:          make(map[string]storage.DaggerheartDamageTypeEntry),
		domains:              make(map[string]storage.DaggerheartDomain),
		domainCards:          make(map[string]storage.DaggerheartDomainCard),
		weapons:              make(map[string]storage.DaggerheartWeapon),
		armor:                make(map[string]storage.DaggerheartArmor),
		items:                make(map[string]storage.DaggerheartItem),
		environments:         make(map[string]storage.DaggerheartEnvironment),
	}
}

func (f *fakeContentStore) PutDaggerheartClass(_ context.Context, c storage.DaggerheartClass) error {
	f.classes[c.ID] = c
	return nil
}
func (f *fakeContentStore) GetDaggerheartClass(_ context.Context, id string) (storage.DaggerheartClass, error) {
	c, ok := f.classes[id]
	if !ok {
		return c, fmt.Errorf("not found")
	}
	return c, nil
}
func (f *fakeContentStore) ListDaggerheartClasses(_ context.Context) ([]storage.DaggerheartClass, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartClass(_ context.Context, _ string) error { return nil }

func (f *fakeContentStore) PutDaggerheartSubclass(_ context.Context, s storage.DaggerheartSubclass) error {
	f.subclasses[s.ID] = s
	return nil
}
func (f *fakeContentStore) GetDaggerheartSubclass(_ context.Context, id string) (storage.DaggerheartSubclass, error) {
	s, ok := f.subclasses[id]
	if !ok {
		return s, fmt.Errorf("not found")
	}
	return s, nil
}
func (f *fakeContentStore) ListDaggerheartSubclasses(_ context.Context) ([]storage.DaggerheartSubclass, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartSubclass(_ context.Context, _ string) error { return nil }

func (f *fakeContentStore) PutDaggerheartHeritage(_ context.Context, h storage.DaggerheartHeritage) error {
	f.heritages[h.ID] = h
	return nil
}
func (f *fakeContentStore) GetDaggerheartHeritage(_ context.Context, id string) (storage.DaggerheartHeritage, error) {
	h, ok := f.heritages[id]
	if !ok {
		return h, fmt.Errorf("not found")
	}
	return h, nil
}
func (f *fakeContentStore) ListDaggerheartHeritages(_ context.Context) ([]storage.DaggerheartHeritage, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartHeritage(_ context.Context, _ string) error { return nil }

func (f *fakeContentStore) PutDaggerheartExperience(_ context.Context, e storage.DaggerheartExperienceEntry) error {
	f.experiences[e.ID] = e
	return nil
}
func (f *fakeContentStore) GetDaggerheartExperience(_ context.Context, id string) (storage.DaggerheartExperienceEntry, error) {
	e, ok := f.experiences[id]
	if !ok {
		return e, fmt.Errorf("not found")
	}
	return e, nil
}
func (f *fakeContentStore) ListDaggerheartExperiences(_ context.Context) ([]storage.DaggerheartExperienceEntry, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartExperience(_ context.Context, _ string) error {
	return nil
}

func (f *fakeContentStore) PutDaggerheartAdversaryEntry(_ context.Context, a storage.DaggerheartAdversaryEntry) error {
	f.adversaries[a.ID] = a
	return nil
}
func (f *fakeContentStore) GetDaggerheartAdversaryEntry(_ context.Context, id string) (storage.DaggerheartAdversaryEntry, error) {
	a, ok := f.adversaries[id]
	if !ok {
		return a, fmt.Errorf("not found")
	}
	return a, nil
}
func (f *fakeContentStore) ListDaggerheartAdversaryEntries(_ context.Context) ([]storage.DaggerheartAdversaryEntry, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartAdversaryEntry(_ context.Context, _ string) error {
	return nil
}

func (f *fakeContentStore) PutDaggerheartBeastform(_ context.Context, b storage.DaggerheartBeastformEntry) error {
	f.beastforms[b.ID] = b
	return nil
}
func (f *fakeContentStore) GetDaggerheartBeastform(_ context.Context, id string) (storage.DaggerheartBeastformEntry, error) {
	b, ok := f.beastforms[id]
	if !ok {
		return b, fmt.Errorf("not found")
	}
	return b, nil
}
func (f *fakeContentStore) ListDaggerheartBeastforms(_ context.Context) ([]storage.DaggerheartBeastformEntry, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartBeastform(_ context.Context, _ string) error { return nil }

func (f *fakeContentStore) PutDaggerheartCompanionExperience(_ context.Context, ce storage.DaggerheartCompanionExperienceEntry) error {
	f.companionExperiences[ce.ID] = ce
	return nil
}
func (f *fakeContentStore) GetDaggerheartCompanionExperience(_ context.Context, id string) (storage.DaggerheartCompanionExperienceEntry, error) {
	ce, ok := f.companionExperiences[id]
	if !ok {
		return ce, fmt.Errorf("not found")
	}
	return ce, nil
}
func (f *fakeContentStore) ListDaggerheartCompanionExperiences(_ context.Context) ([]storage.DaggerheartCompanionExperienceEntry, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartCompanionExperience(_ context.Context, _ string) error {
	return nil
}

func (f *fakeContentStore) PutDaggerheartLootEntry(_ context.Context, l storage.DaggerheartLootEntry) error {
	f.lootEntries[l.ID] = l
	return nil
}
func (f *fakeContentStore) GetDaggerheartLootEntry(_ context.Context, id string) (storage.DaggerheartLootEntry, error) {
	l, ok := f.lootEntries[id]
	if !ok {
		return l, fmt.Errorf("not found")
	}
	return l, nil
}
func (f *fakeContentStore) ListDaggerheartLootEntries(_ context.Context) ([]storage.DaggerheartLootEntry, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartLootEntry(_ context.Context, _ string) error { return nil }

func (f *fakeContentStore) PutDaggerheartDamageType(_ context.Context, d storage.DaggerheartDamageTypeEntry) error {
	f.damageTypes[d.ID] = d
	return nil
}
func (f *fakeContentStore) GetDaggerheartDamageType(_ context.Context, id string) (storage.DaggerheartDamageTypeEntry, error) {
	d, ok := f.damageTypes[id]
	if !ok {
		return d, fmt.Errorf("not found")
	}
	return d, nil
}
func (f *fakeContentStore) ListDaggerheartDamageTypes(_ context.Context) ([]storage.DaggerheartDamageTypeEntry, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartDamageType(_ context.Context, _ string) error {
	return nil
}

func (f *fakeContentStore) PutDaggerheartDomain(_ context.Context, d storage.DaggerheartDomain) error {
	f.domains[d.ID] = d
	return nil
}
func (f *fakeContentStore) GetDaggerheartDomain(_ context.Context, id string) (storage.DaggerheartDomain, error) {
	d, ok := f.domains[id]
	if !ok {
		return d, fmt.Errorf("not found")
	}
	return d, nil
}
func (f *fakeContentStore) ListDaggerheartDomains(_ context.Context) ([]storage.DaggerheartDomain, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartDomain(_ context.Context, _ string) error { return nil }

func (f *fakeContentStore) PutDaggerheartDomainCard(_ context.Context, c storage.DaggerheartDomainCard) error {
	f.domainCards[c.ID] = c
	return nil
}
func (f *fakeContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (storage.DaggerheartDomainCard, error) {
	c, ok := f.domainCards[id]
	if !ok {
		return c, fmt.Errorf("not found")
	}
	return c, nil
}
func (f *fakeContentStore) ListDaggerheartDomainCards(_ context.Context) ([]storage.DaggerheartDomainCard, error) {
	return nil, nil
}
func (f *fakeContentStore) ListDaggerheartDomainCardsByDomain(_ context.Context, _ string) ([]storage.DaggerheartDomainCard, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartDomainCard(_ context.Context, _ string) error {
	return nil
}

func (f *fakeContentStore) PutDaggerheartWeapon(_ context.Context, w storage.DaggerheartWeapon) error {
	f.weapons[w.ID] = w
	return nil
}
func (f *fakeContentStore) GetDaggerheartWeapon(_ context.Context, id string) (storage.DaggerheartWeapon, error) {
	w, ok := f.weapons[id]
	if !ok {
		return w, fmt.Errorf("not found")
	}
	return w, nil
}
func (f *fakeContentStore) ListDaggerheartWeapons(_ context.Context) ([]storage.DaggerheartWeapon, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartWeapon(_ context.Context, _ string) error { return nil }

func (f *fakeContentStore) PutDaggerheartArmor(_ context.Context, a storage.DaggerheartArmor) error {
	f.armor[a.ID] = a
	return nil
}
func (f *fakeContentStore) GetDaggerheartArmor(_ context.Context, id string) (storage.DaggerheartArmor, error) {
	a, ok := f.armor[id]
	if !ok {
		return a, fmt.Errorf("not found")
	}
	return a, nil
}
func (f *fakeContentStore) ListDaggerheartArmor(_ context.Context) ([]storage.DaggerheartArmor, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartArmor(_ context.Context, _ string) error { return nil }

func (f *fakeContentStore) PutDaggerheartItem(_ context.Context, item storage.DaggerheartItem) error {
	f.items[item.ID] = item
	return nil
}
func (f *fakeContentStore) GetDaggerheartItem(_ context.Context, id string) (storage.DaggerheartItem, error) {
	item, ok := f.items[id]
	if !ok {
		return item, fmt.Errorf("not found")
	}
	return item, nil
}
func (f *fakeContentStore) ListDaggerheartItems(_ context.Context) ([]storage.DaggerheartItem, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartItem(_ context.Context, _ string) error { return nil }

func (f *fakeContentStore) PutDaggerheartEnvironment(_ context.Context, e storage.DaggerheartEnvironment) error {
	f.environments[e.ID] = e
	return nil
}
func (f *fakeContentStore) GetDaggerheartEnvironment(_ context.Context, id string) (storage.DaggerheartEnvironment, error) {
	e, ok := f.environments[id]
	if !ok {
		return e, fmt.Errorf("not found")
	}
	return e, nil
}
func (f *fakeContentStore) ListDaggerheartEnvironments(_ context.Context) ([]storage.DaggerheartEnvironment, error) {
	return nil, nil
}
func (f *fakeContentStore) DeleteDaggerheartEnvironment(_ context.Context, _ string) error {
	return nil
}

func (f *fakeContentStore) PutDaggerheartContentString(_ context.Context, s storage.DaggerheartContentString) error {
	f.contentStrings = append(f.contentStrings, s)
	return nil
}

// --- Pure mapping function tests ---

func TestToStorageFeatures(t *testing.T) {
	input := []featureRecord{
		{ID: "f1", Name: "Shield Bash", Description: "Knock back", Level: 2},
		{ID: "f2", Name: "Heal", Description: "Restore HP", Level: 1},
	}
	got := toStorageFeatures(input)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "f1" || got[0].Name != "Shield Bash" || got[0].Level != 2 {
		t.Errorf("got[0] = %+v", got[0])
	}

	// Empty input returns empty (not nil) slice.
	empty := toStorageFeatures(nil)
	if len(empty) != 0 {
		t.Errorf("expected empty, got %d", len(empty))
	}
}

func TestToStorageHopeFeature(t *testing.T) {
	input := hopeFeatureRecord{Name: "Hope Strike", Description: "Powerful attack", HopeCost: 3}
	got := toStorageHopeFeature(input)
	if got.Name != "Hope Strike" || got.HopeCost != 3 {
		t.Errorf("got %+v", got)
	}
}

func TestToStorageDamageDice(t *testing.T) {
	input := []damageDieRecord{{Sides: 6, Count: 2}, {Sides: 8, Count: 1}}
	got := toStorageDamageDice(input)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Sides != 6 || got[0].Count != 2 {
		t.Errorf("got[0] = %+v", got[0])
	}
}

func TestToStorageAdversaryAttack(t *testing.T) {
	input := adversaryAttackRecord{
		Name: "Slash", Range: "melee",
		DamageDice:  []damageDieRecord{{Sides: 6, Count: 1}},
		DamageBonus: 2, DamageType: "physical",
	}
	got := toStorageAdversaryAttack(input)
	if got.Name != "Slash" || got.DamageBonus != 2 {
		t.Errorf("got %+v", got)
	}
	if len(got.DamageDice) != 1 {
		t.Errorf("dice len = %d", len(got.DamageDice))
	}
}

func TestToStorageAdversaryExperiences(t *testing.T) {
	input := []adversaryExperienceRecord{{Name: "Stealth", Modifier: 3}}
	got := toStorageAdversaryExperiences(input)
	if len(got) != 1 || got[0].Name != "Stealth" || got[0].Modifier != 3 {
		t.Errorf("got %+v", got)
	}
}

func TestToStorageAdversaryFeatures(t *testing.T) {
	input := []adversaryFeatureRecord{
		{ID: "af1", Name: "Tough", Kind: "passive", Description: "Extra armor", CostType: "stress", Cost: 1},
	}
	got := toStorageAdversaryFeatures(input)
	if len(got) != 1 || got[0].ID != "af1" || got[0].CostType != "stress" {
		t.Errorf("got %+v", got)
	}
}

func TestToStorageBeastformAttack(t *testing.T) {
	input := beastformAttackRecord{
		Range: "close", Trait: "ferocity",
		DamageDice:  []damageDieRecord{{Sides: 8, Count: 2}},
		DamageBonus: 1, DamageType: "physical",
	}
	got := toStorageBeastformAttack(input)
	if got.Trait != "ferocity" || got.DamageBonus != 1 {
		t.Errorf("got %+v", got)
	}
}

func TestToStorageBeastformFeatures(t *testing.T) {
	input := []beastformFeatureRecord{{ID: "bf1", Name: "Pack Tactics", Description: "Advantage"}}
	got := toStorageBeastformFeatures(input)
	if len(got) != 1 || got[0].ID != "bf1" {
		t.Errorf("got %+v", got)
	}
}

// --- Upsert function tests (with fake store) ---

func TestUpsertClasses(t *testing.T) {
	store := newFakeContentStore()
	ctx := context.Background()
	now := time.Now()

	items := []classRecord{{
		ID: "cls-1", Name: "Guardian",
		StartingEvasion: 8, StartingHP: 20,
		StartingItems: []string{"sword", "shield"},
		Features: []featureRecord{
			{ID: "f1", Name: "Block", Description: "Reduce damage", Level: 1},
		},
		HopeFeature: hopeFeatureRecord{Name: "Last Stand", Description: "Final push", HopeCost: 2},
		DomainIDs:   []string{"valor"},
	}}

	if err := upsertClasses(ctx, store, items, "en-US", true, now); err != nil {
		t.Fatalf("upsertClasses: %v", err)
	}
	if _, ok := store.classes["cls-1"]; !ok {
		t.Fatal("class not stored")
	}
	if len(store.contentStrings) == 0 {
		t.Fatal("expected content strings")
	}
}

func TestUpsertClasses_EmptyID(t *testing.T) {
	store := newFakeContentStore()
	err := upsertClasses(context.Background(), store, []classRecord{{ID: "  "}}, "en-US", true, time.Now())
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestUpsertClasses_LocaleOnly(t *testing.T) {
	store := newFakeContentStore()
	items := []classRecord{{
		ID: "cls-1", Name: "Guardian",
		Features: []featureRecord{{ID: "f1", Name: "Block", Description: "Desc"}},
	}}
	if err := upsertClasses(context.Background(), store, items, "pt-BR", false, time.Now()); err != nil {
		t.Fatalf("upsertClasses locale-only: %v", err)
	}
	if len(store.classes) != 0 {
		t.Error("expected no class stored for non-base locale")
	}
	if len(store.contentStrings) == 0 {
		t.Error("expected content strings for locale")
	}
}

func TestUpsertSubclasses(t *testing.T) {
	store := newFakeContentStore()
	items := []subclassRecord{{
		ID: "sub-1", Name: "Bladeweaver", SpellcastTrait: "agility",
		FoundationFeatures:     []featureRecord{{ID: "ff1", Name: "Blade", Description: "Desc"}},
		SpecializationFeatures: []featureRecord{{ID: "sf1", Name: "Spec", Description: "Desc"}},
		MasteryFeatures:        []featureRecord{{ID: "mf1", Name: "Master", Description: "Desc"}},
	}}
	if err := upsertSubclasses(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertSubclasses: %v", err)
	}
	if _, ok := store.subclasses["sub-1"]; !ok {
		t.Fatal("subclass not stored")
	}
}

func TestUpsertSubclasses_EmptyID(t *testing.T) {
	err := upsertSubclasses(context.Background(), newFakeContentStore(), []subclassRecord{{ID: ""}}, "en-US", true, time.Now())
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestUpsertHeritages(t *testing.T) {
	store := newFakeContentStore()
	items := []heritageRecord{{
		ID: "her-1", Name: "Elf", Kind: "ancestry",
		Features: []featureRecord{{ID: "hf1", Name: "Darkvision", Description: "See in dark"}},
	}}
	if err := upsertHeritages(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertHeritages: %v", err)
	}
	if h := store.heritages["her-1"]; h.Kind != "ancestry" {
		t.Errorf("heritage kind = %q", h.Kind)
	}
}

func TestUpsertExperiences(t *testing.T) {
	store := newFakeContentStore()
	items := []experienceRecord{{ID: "exp-1", Name: "Wanderer", Description: "Traveled far"}}
	if err := upsertExperiences(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertExperiences: %v", err)
	}
	if _, ok := store.experiences["exp-1"]; !ok {
		t.Fatal("experience not stored")
	}
}

func TestUpsertAdversaries(t *testing.T) {
	store := newFakeContentStore()
	items := []adversaryRecord{{
		ID: "adv-1", Name: "Goblin", Tier: 1, Role: "solo",
		Description: "A small creature", Motives: "Greed",
		Difficulty: 3, MajorThreshold: 5, SevereThreshold: 10,
		HP: 8, Stress: 2, Armor: 1, AttackModifier: 1,
		StandardAttack: adversaryAttackRecord{
			Name: "Stab", Range: "melee",
			DamageDice: []damageDieRecord{{Sides: 6, Count: 1}},
		},
		Experiences: []adversaryExperienceRecord{{Name: "Stealth", Modifier: 2}},
		Features: []adversaryFeatureRecord{
			{ID: "af1", Name: "Sneaky", Kind: "passive", Description: "Hard to spot"},
		},
	}}
	if err := upsertAdversaries(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertAdversaries: %v", err)
	}
	if _, ok := store.adversaries["adv-1"]; !ok {
		t.Fatal("adversary not stored")
	}
}

func TestUpsertBeastforms(t *testing.T) {
	store := newFakeContentStore()
	items := []beastformRecord{{
		ID: "beast-1", Name: "Wolf", Tier: 1,
		Examples: "Gray Wolf", Trait: "ferocity", TraitBonus: 2, EvasionBonus: 1,
		Attack: beastformAttackRecord{
			Range: "close", Trait: "ferocity",
			DamageDice: []damageDieRecord{{Sides: 6, Count: 1}},
		},
		Advantages: []string{"pack tactics"},
		Features: []beastformFeatureRecord{
			{ID: "bf1", Name: "Howl", Description: "Frighten nearby"},
		},
	}}
	if err := upsertBeastforms(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertBeastforms: %v", err)
	}
	if b := store.beastforms["beast-1"]; len(b.Advantages) != 1 {
		t.Errorf("advantages = %v", b.Advantages)
	}
}

func TestUpsertCompanionExperiences(t *testing.T) {
	store := newFakeContentStore()
	items := []companionExperienceRecord{{ID: "cexp-1", Name: "Guard", Description: "Protects allies"}}
	if err := upsertCompanionExperiences(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertCompanionExperiences: %v", err)
	}
	if _, ok := store.companionExperiences["cexp-1"]; !ok {
		t.Fatal("companion experience not stored")
	}
}

func TestUpsertLootEntries(t *testing.T) {
	store := newFakeContentStore()
	items := []lootEntryRecord{{ID: "loot-1", Name: "Gold", Roll: 5, Description: "Shiny coins"}}
	if err := upsertLootEntries(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertLootEntries: %v", err)
	}
	if _, ok := store.lootEntries["loot-1"]; !ok {
		t.Fatal("loot entry not stored")
	}
}

func TestUpsertDamageTypes(t *testing.T) {
	store := newFakeContentStore()
	items := []damageTypeRecord{{ID: "dt-1", Name: "Fire", Description: "Burns things"}}
	if err := upsertDamageTypes(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertDamageTypes: %v", err)
	}
	if _, ok := store.damageTypes["dt-1"]; !ok {
		t.Fatal("damage type not stored")
	}
}

func TestUpsertDomains(t *testing.T) {
	store := newFakeContentStore()
	items := []domainRecord{{ID: "dom-1", Name: "Valor", Description: "Courage and honor"}}
	if err := upsertDomains(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertDomains: %v", err)
	}
	if _, ok := store.domains["dom-1"]; !ok {
		t.Fatal("domain not stored")
	}
}

func TestUpsertDomainCards(t *testing.T) {
	store := newFakeContentStore()
	items := []domainCardRecord{{
		ID: "card-1", Name: "Fireball", DomainID: "dom-1",
		Level: 3, Type: "spell", RecallCost: 2,
		UsageLimit: "once per rest", FeatureText: "Deal 3d6 fire damage",
	}}
	if err := upsertDomainCards(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertDomainCards: %v", err)
	}
	if c := store.domainCards["card-1"]; c.Level != 3 {
		t.Errorf("level = %d", c.Level)
	}
}

func TestUpsertWeapons(t *testing.T) {
	store := newFakeContentStore()
	items := []weaponRecord{{
		ID: "weap-1", Name: "Blade", Category: "one-handed", Tier: 1,
		Trait: "agility", Range: "melee",
		DamageDice: []damageDieRecord{{Sides: 6, Count: 1}},
		DamageType: "physical", Burden: 1, Feature: "Quick draw",
	}}
	if err := upsertWeapons(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertWeapons: %v", err)
	}
	if w := store.weapons["weap-1"]; len(w.DamageDice) != 1 {
		t.Errorf("damage dice = %v", w.DamageDice)
	}
}

func TestUpsertArmor(t *testing.T) {
	store := newFakeContentStore()
	items := []armorRecord{{
		ID: "armor-1", Name: "Chain Mail", Tier: 2,
		BaseMajorThreshold: 7, BaseSevereThreshold: 14,
		ArmorScore: 3, Feature: "Heavy",
	}}
	if err := upsertArmor(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertArmor: %v", err)
	}
	if a := store.armor["armor-1"]; a.ArmorScore != 3 {
		t.Errorf("armor score = %d", a.ArmorScore)
	}
}

func TestUpsertItems(t *testing.T) {
	store := newFakeContentStore()
	items := []itemRecord{{
		ID: "item-1", Name: "Potion", Rarity: "common",
		Kind: "consumable", StackMax: 5,
		Description: "Heals wounds", EffectText: "Restore 1d6 HP",
	}}
	if err := upsertItems(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertItems: %v", err)
	}
	if i := store.items["item-1"]; i.StackMax != 5 {
		t.Errorf("stack max = %d", i.StackMax)
	}
}

func TestUpsertEnvironments(t *testing.T) {
	store := newFakeContentStore()
	items := []environmentRecord{{
		ID: "env-1", Name: "Forest", Tier: 1, Type: "wilderness",
		Difficulty:            2,
		Impulses:              []string{"hide", "ambush"},
		PotentialAdversaryIDs: []string{"adv-1"},
		Features: []featureRecord{
			{ID: "ef1", Name: "Dense Canopy", Description: "Blocks line of sight"},
		},
		Prompts: []string{"What lurks in the shadows?"},
	}}
	if err := upsertEnvironments(context.Background(), store, items, "en-US", true, time.Now()); err != nil {
		t.Fatalf("upsertEnvironments: %v", err)
	}
	if e := store.environments["env-1"]; len(e.Impulses) != 2 {
		t.Errorf("impulses = %v", e.Impulses)
	}
}

// --- ID validation tests ---

func TestUpsertEmptyIDs(t *testing.T) {
	store := newFakeContentStore()
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"heritage", func() error { return upsertHeritages(ctx, store, []heritageRecord{{ID: ""}}, "en", true, now) }},
		{"experience", func() error { return upsertExperiences(ctx, store, []experienceRecord{{ID: ""}}, "en", true, now) }},
		{"adversary", func() error { return upsertAdversaries(ctx, store, []adversaryRecord{{ID: ""}}, "en", true, now) }},
		{"beastform", func() error { return upsertBeastforms(ctx, store, []beastformRecord{{ID: ""}}, "en", true, now) }},
		{"companion experience", func() error {
			return upsertCompanionExperiences(ctx, store, []companionExperienceRecord{{ID: ""}}, "en", true, now)
		}},
		{"loot entry", func() error { return upsertLootEntries(ctx, store, []lootEntryRecord{{ID: ""}}, "en", true, now) }},
		{"damage type", func() error { return upsertDamageTypes(ctx, store, []damageTypeRecord{{ID: ""}}, "en", true, now) }},
		{"domain", func() error { return upsertDomains(ctx, store, []domainRecord{{ID: ""}}, "en", true, now) }},
		{"domain card", func() error { return upsertDomainCards(ctx, store, []domainCardRecord{{ID: ""}}, "en", true, now) }},
		{"weapon", func() error { return upsertWeapons(ctx, store, []weaponRecord{{ID: ""}}, "en", true, now) }},
		{"armor", func() error { return upsertArmor(ctx, store, []armorRecord{{ID: ""}}, "en", true, now) }},
		{"item", func() error { return upsertItems(ctx, store, []itemRecord{{ID: ""}}, "en", true, now) }},
		{"environment", func() error { return upsertEnvironments(ctx, store, []environmentRecord{{ID: ""}}, "en", true, now) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err == nil {
				t.Fatal("expected error for empty ID")
			}
		})
	}
}

func TestUpsertFeatureStrings_EmptyID(t *testing.T) {
	store := newFakeContentStore()
	err := upsertFeatureStrings(context.Background(), store, featureRecord{ID: ""}, "en", time.Now())
	if err == nil {
		t.Fatal("expected error for empty feature ID")
	}
}

func TestUpsertAdversaryFeatureStrings_EmptyID(t *testing.T) {
	store := newFakeContentStore()
	err := upsertAdversaryFeatureStrings(context.Background(), store, adversaryFeatureRecord{ID: ""}, "en", time.Now())
	if err == nil {
		t.Fatal("expected error for empty adversary feature ID")
	}
}

func TestUpsertBeastformFeatureStrings_EmptyID(t *testing.T) {
	store := newFakeContentStore()
	err := upsertBeastformFeatureStrings(context.Background(), store, beastformFeatureRecord{ID: ""}, "en", time.Now())
	if err == nil {
		t.Fatal("expected error for empty beastform feature ID")
	}
}

// --- putContentString tests ---

func TestPutContentString_SkipsEmpty(t *testing.T) {
	store := newFakeContentStore()
	err := putContentString(context.Background(), store, "id", "type", "field", "en", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(store.contentStrings) != 0 {
		t.Error("expected no strings stored for empty text")
	}
}

func TestPutContentString_StoresNonEmpty(t *testing.T) {
	store := newFakeContentStore()
	err := putContentString(context.Background(), store, "id", "class", "name", "en", "Guardian", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(store.contentStrings) != 1 {
		t.Fatalf("expected 1 string, got %d", len(store.contentStrings))
	}
	if store.contentStrings[0].Text != "Guardian" {
		t.Errorf("text = %q", store.contentStrings[0].Text)
	}
}

// --- upsertLocale integration test ---

func TestUpsertLocale_AllContentTypes(t *testing.T) {
	store := newFakeContentStore()
	ctx := context.Background()
	now := time.Now()

	payloads := localePayloads{
		Domains:              &domainPayload{Items: []domainRecord{{ID: "dom-1", Name: "Valor", Description: "Courage"}}},
		DomainCards:          &domainCardPayload{Items: []domainCardRecord{{ID: "card-1", Name: "Shield", DomainID: "dom-1", UsageLimit: "1/rest", FeatureText: "Block"}}},
		Classes:              &classPayload{Items: []classRecord{{ID: "cls-1", Name: "Guardian", Features: []featureRecord{{ID: "f1", Name: "Block", Description: "Reduce"}}}}},
		Subclasses:           &subclassPayload{Items: []subclassRecord{{ID: "sub-1", Name: "Bladeweaver"}}},
		Heritages:            &heritagePayload{Items: []heritageRecord{{ID: "her-1", Name: "Elf", Kind: "ancestry"}}},
		Experiences:          &experiencePayload{Items: []experienceRecord{{ID: "exp-1", Name: "Wanderer", Description: "Traveled"}}},
		Adversaries:          &adversaryPayload{Items: []adversaryRecord{{ID: "adv-1", Name: "Goblin", Features: []adversaryFeatureRecord{{ID: "af1", Name: "Sneaky", Description: "D"}}}}},
		Beastforms:           &beastformPayload{Items: []beastformRecord{{ID: "beast-1", Name: "Wolf", Advantages: []string{"pack"}, Features: []beastformFeatureRecord{{ID: "bf1", Name: "Howl", Description: "D"}}}}},
		CompanionExperiences: &companionExperiencePayload{Items: []companionExperienceRecord{{ID: "cexp-1", Name: "Guard", Description: "Protects"}}},
		LootEntries:          &lootEntryPayload{Items: []lootEntryRecord{{ID: "loot-1", Name: "Gold", Description: "Coins"}}},
		DamageTypes:          &damageTypePayload{Items: []damageTypeRecord{{ID: "dt-1", Name: "Fire", Description: "Burns"}}},
		Weapons:              &weaponPayload{Items: []weaponRecord{{ID: "weap-1", Name: "Blade", Feature: "Quick"}}},
		Armor:                &armorPayload{Items: []armorRecord{{ID: "armor-1", Name: "Chain Mail", Feature: "Heavy"}}},
		Items:                &itemPayload{Items: []itemRecord{{ID: "item-1", Name: "Potion", Description: "Heals", EffectText: "Restore HP"}}},
		Environments:         &environmentPayload{Items: []environmentRecord{{ID: "env-1", Name: "Forest", Impulses: []string{"hide"}, Prompts: []string{"What lurks?"}, Features: []featureRecord{{ID: "ef1", Name: "Canopy", Description: "Blocks"}}}}},
	}

	if err := upsertLocale(ctx, store, "en-US", true, payloads, now); err != nil {
		t.Fatalf("upsertLocale: %v", err)
	}

	// Verify all content types stored.
	if len(store.classes) != 1 {
		t.Errorf("classes = %d", len(store.classes))
	}
	if len(store.subclasses) != 1 {
		t.Errorf("subclasses = %d", len(store.subclasses))
	}
	if len(store.heritages) != 1 {
		t.Errorf("heritages = %d", len(store.heritages))
	}
	if len(store.experiences) != 1 {
		t.Errorf("experiences = %d", len(store.experiences))
	}
	if len(store.adversaries) != 1 {
		t.Errorf("adversaries = %d", len(store.adversaries))
	}
	if len(store.beastforms) != 1 {
		t.Errorf("beastforms = %d", len(store.beastforms))
	}
	if len(store.companionExperiences) != 1 {
		t.Errorf("companionExperiences = %d", len(store.companionExperiences))
	}
	if len(store.lootEntries) != 1 {
		t.Errorf("lootEntries = %d", len(store.lootEntries))
	}
	if len(store.damageTypes) != 1 {
		t.Errorf("damageTypes = %d", len(store.damageTypes))
	}
	if len(store.domains) != 1 {
		t.Errorf("domains = %d", len(store.domains))
	}
	if len(store.domainCards) != 1 {
		t.Errorf("domainCards = %d", len(store.domainCards))
	}
	if len(store.weapons) != 1 {
		t.Errorf("weapons = %d", len(store.weapons))
	}
	if len(store.armor) != 1 {
		t.Errorf("armor = %d", len(store.armor))
	}
	if len(store.items) != 1 {
		t.Errorf("items = %d", len(store.items))
	}
	if len(store.environments) != 1 {
		t.Errorf("environments = %d", len(store.environments))
	}
	if len(store.contentStrings) == 0 {
		t.Error("expected content strings to be stored")
	}
}

func TestUpsertLocale_NilPayloads(t *testing.T) {
	store := newFakeContentStore()
	if err := upsertLocale(context.Background(), store, "en-US", true, localePayloads{}, time.Now()); err != nil {
		t.Fatalf("upsertLocale with nil payloads: %v", err)
	}
}

// --- readLocalePayloads test ---

func TestReadLocalePayloads_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	payloads, err := readLocalePayloads(dir)
	if err != nil {
		t.Fatalf("readLocalePayloads: %v", err)
	}
	if payloads.Classes != nil {
		t.Error("expected nil classes for empty dir")
	}
}

// --- validateLocalePayloads additional tests ---

func TestValidateLocalePayloads_AllTypes(t *testing.T) {
	locale := "en-US"
	valid := func() localePayloads {
		base := func() *classPayload {
			return &classPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale}
		}
		return localePayloads{
			Classes:              base(),
			Subclasses:           &subclassPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			Heritages:            &heritagePayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			Experiences:          &experiencePayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			Adversaries:          &adversaryPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			Beastforms:           &beastformPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			CompanionExperiences: &companionExperiencePayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			LootEntries:          &lootEntryPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			DamageTypes:          &damageTypePayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			Domains:              &domainPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			DomainCards:          &domainCardPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			Weapons:              &weaponPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			Armor:                &armorPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			Items:                &itemPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
			Environments:         &environmentPayload{SystemID: defaultSystemID, SystemVersion: defaultSystemVer, Source: "src", Locale: locale},
		}
	}

	if err := validateLocalePayloads(locale, valid()); err != nil {
		t.Fatalf("expected valid: %v", err)
	}

	// Bad version.
	bad := valid()
	bad.Weapons.SystemVersion = "v99"
	if err := validateLocalePayloads(locale, bad); err == nil {
		t.Fatal("expected error for bad system version")
	}

	// Empty source.
	bad2 := valid()
	bad2.Armor.Source = "  "
	if err := validateLocalePayloads(locale, bad2); err == nil {
		t.Fatal("expected error for empty source")
	}

	// Locale mismatch.
	bad3 := valid()
	bad3.Items.Locale = "fr-FR"
	if err := validateLocalePayloads(locale, bad3); err == nil {
		t.Fatal("expected error for locale mismatch")
	}
}
