package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// fakeContentStore implements storage.DaggerheartContentStore for testing content
// service endpoints. Only the methods needed for tests are implemented with
// real behavior; others return nil/empty to satisfy the interface.
type fakeContentStore struct {
	classes              map[string]storage.DaggerheartClass
	subclasses           map[string]storage.DaggerheartSubclass
	heritages            map[string]storage.DaggerheartHeritage
	experiences          map[string]storage.DaggerheartExperienceEntry
	adversaryEntries     map[string]storage.DaggerheartAdversaryEntry
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
	contentStrings       map[fakeContentStringKey]storage.DaggerheartContentString
}

type fakeContentStringKey struct {
	ContentID string
	Field     string
	Locale    string
}

func newFakeContentStore() *fakeContentStore {
	return &fakeContentStore{
		classes:              make(map[string]storage.DaggerheartClass),
		subclasses:           make(map[string]storage.DaggerheartSubclass),
		heritages:            make(map[string]storage.DaggerheartHeritage),
		experiences:          make(map[string]storage.DaggerheartExperienceEntry),
		adversaryEntries:     make(map[string]storage.DaggerheartAdversaryEntry),
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
		contentStrings:       make(map[fakeContentStringKey]storage.DaggerheartContentString),
	}
}

func (s *fakeContentStore) PutDaggerheartClass(_ context.Context, c storage.DaggerheartClass) error {
	s.classes[c.ID] = c
	return nil
}
func (s *fakeContentStore) GetDaggerheartClass(_ context.Context, id string) (storage.DaggerheartClass, error) {
	c, ok := s.classes[id]
	if !ok {
		return storage.DaggerheartClass{}, storage.ErrNotFound
	}
	return c, nil
}
func (s *fakeContentStore) ListDaggerheartClasses(_ context.Context) ([]storage.DaggerheartClass, error) {
	result := make([]storage.DaggerheartClass, 0, len(s.classes))
	for _, c := range s.classes {
		result = append(result, c)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartClass(_ context.Context, id string) error {
	delete(s.classes, id)
	return nil
}

func (s *fakeContentStore) PutDaggerheartSubclass(_ context.Context, c storage.DaggerheartSubclass) error {
	s.subclasses[c.ID] = c
	return nil
}
func (s *fakeContentStore) GetDaggerheartSubclass(_ context.Context, id string) (storage.DaggerheartSubclass, error) {
	c, ok := s.subclasses[id]
	if !ok {
		return storage.DaggerheartSubclass{}, storage.ErrNotFound
	}
	return c, nil
}
func (s *fakeContentStore) ListDaggerheartSubclasses(_ context.Context) ([]storage.DaggerheartSubclass, error) {
	result := make([]storage.DaggerheartSubclass, 0, len(s.subclasses))
	for _, c := range s.subclasses {
		result = append(result, c)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartSubclass(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartHeritage(_ context.Context, h storage.DaggerheartHeritage) error {
	s.heritages[h.ID] = h
	return nil
}
func (s *fakeContentStore) GetDaggerheartHeritage(_ context.Context, id string) (storage.DaggerheartHeritage, error) {
	h, ok := s.heritages[id]
	if !ok {
		return storage.DaggerheartHeritage{}, storage.ErrNotFound
	}
	return h, nil
}
func (s *fakeContentStore) ListDaggerheartHeritages(_ context.Context) ([]storage.DaggerheartHeritage, error) {
	result := make([]storage.DaggerheartHeritage, 0, len(s.heritages))
	for _, h := range s.heritages {
		result = append(result, h)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartHeritage(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartExperience(_ context.Context, e storage.DaggerheartExperienceEntry) error {
	s.experiences[e.ID] = e
	return nil
}
func (s *fakeContentStore) GetDaggerheartExperience(_ context.Context, id string) (storage.DaggerheartExperienceEntry, error) {
	e, ok := s.experiences[id]
	if !ok {
		return storage.DaggerheartExperienceEntry{}, storage.ErrNotFound
	}
	return e, nil
}
func (s *fakeContentStore) ListDaggerheartExperiences(_ context.Context) ([]storage.DaggerheartExperienceEntry, error) {
	result := make([]storage.DaggerheartExperienceEntry, 0, len(s.experiences))
	for _, e := range s.experiences {
		result = append(result, e)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartExperience(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartAdversaryEntry(_ context.Context, a storage.DaggerheartAdversaryEntry) error {
	s.adversaryEntries[a.ID] = a
	return nil
}
func (s *fakeContentStore) GetDaggerheartAdversaryEntry(_ context.Context, id string) (storage.DaggerheartAdversaryEntry, error) {
	a, ok := s.adversaryEntries[id]
	if !ok {
		return storage.DaggerheartAdversaryEntry{}, storage.ErrNotFound
	}
	return a, nil
}
func (s *fakeContentStore) ListDaggerheartAdversaryEntries(_ context.Context) ([]storage.DaggerheartAdversaryEntry, error) {
	result := make([]storage.DaggerheartAdversaryEntry, 0, len(s.adversaryEntries))
	for _, a := range s.adversaryEntries {
		result = append(result, a)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartAdversaryEntry(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartBeastform(_ context.Context, b storage.DaggerheartBeastformEntry) error {
	s.beastforms[b.ID] = b
	return nil
}
func (s *fakeContentStore) GetDaggerheartBeastform(_ context.Context, id string) (storage.DaggerheartBeastformEntry, error) {
	b, ok := s.beastforms[id]
	if !ok {
		return storage.DaggerheartBeastformEntry{}, storage.ErrNotFound
	}
	return b, nil
}
func (s *fakeContentStore) ListDaggerheartBeastforms(_ context.Context) ([]storage.DaggerheartBeastformEntry, error) {
	result := make([]storage.DaggerheartBeastformEntry, 0, len(s.beastforms))
	for _, b := range s.beastforms {
		result = append(result, b)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartBeastform(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartCompanionExperience(_ context.Context, e storage.DaggerheartCompanionExperienceEntry) error {
	s.companionExperiences[e.ID] = e
	return nil
}
func (s *fakeContentStore) GetDaggerheartCompanionExperience(_ context.Context, id string) (storage.DaggerheartCompanionExperienceEntry, error) {
	e, ok := s.companionExperiences[id]
	if !ok {
		return storage.DaggerheartCompanionExperienceEntry{}, storage.ErrNotFound
	}
	return e, nil
}
func (s *fakeContentStore) ListDaggerheartCompanionExperiences(_ context.Context) ([]storage.DaggerheartCompanionExperienceEntry, error) {
	result := make([]storage.DaggerheartCompanionExperienceEntry, 0, len(s.companionExperiences))
	for _, e := range s.companionExperiences {
		result = append(result, e)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartCompanionExperience(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartLootEntry(_ context.Context, l storage.DaggerheartLootEntry) error {
	s.lootEntries[l.ID] = l
	return nil
}
func (s *fakeContentStore) GetDaggerheartLootEntry(_ context.Context, id string) (storage.DaggerheartLootEntry, error) {
	l, ok := s.lootEntries[id]
	if !ok {
		return storage.DaggerheartLootEntry{}, storage.ErrNotFound
	}
	return l, nil
}
func (s *fakeContentStore) ListDaggerheartLootEntries(_ context.Context) ([]storage.DaggerheartLootEntry, error) {
	result := make([]storage.DaggerheartLootEntry, 0, len(s.lootEntries))
	for _, l := range s.lootEntries {
		result = append(result, l)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartLootEntry(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartDamageType(_ context.Context, d storage.DaggerheartDamageTypeEntry) error {
	s.damageTypes[d.ID] = d
	return nil
}
func (s *fakeContentStore) GetDaggerheartDamageType(_ context.Context, id string) (storage.DaggerheartDamageTypeEntry, error) {
	d, ok := s.damageTypes[id]
	if !ok {
		return storage.DaggerheartDamageTypeEntry{}, storage.ErrNotFound
	}
	return d, nil
}
func (s *fakeContentStore) ListDaggerheartDamageTypes(_ context.Context) ([]storage.DaggerheartDamageTypeEntry, error) {
	result := make([]storage.DaggerheartDamageTypeEntry, 0, len(s.damageTypes))
	for _, d := range s.damageTypes {
		result = append(result, d)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartDamageType(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartDomain(_ context.Context, d storage.DaggerheartDomain) error {
	s.domains[d.ID] = d
	return nil
}
func (s *fakeContentStore) GetDaggerheartDomain(_ context.Context, id string) (storage.DaggerheartDomain, error) {
	d, ok := s.domains[id]
	if !ok {
		return storage.DaggerheartDomain{}, storage.ErrNotFound
	}
	return d, nil
}
func (s *fakeContentStore) ListDaggerheartDomains(_ context.Context) ([]storage.DaggerheartDomain, error) {
	result := make([]storage.DaggerheartDomain, 0, len(s.domains))
	for _, d := range s.domains {
		result = append(result, d)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartDomain(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartDomainCard(_ context.Context, c storage.DaggerheartDomainCard) error {
	s.domainCards[c.ID] = c
	return nil
}
func (s *fakeContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (storage.DaggerheartDomainCard, error) {
	c, ok := s.domainCards[id]
	if !ok {
		return storage.DaggerheartDomainCard{}, storage.ErrNotFound
	}
	return c, nil
}
func (s *fakeContentStore) ListDaggerheartDomainCards(_ context.Context) ([]storage.DaggerheartDomainCard, error) {
	result := make([]storage.DaggerheartDomainCard, 0, len(s.domainCards))
	for _, c := range s.domainCards {
		result = append(result, c)
	}
	return result, nil
}
func (s *fakeContentStore) ListDaggerheartDomainCardsByDomain(_ context.Context, domainID string) ([]storage.DaggerheartDomainCard, error) {
	var result []storage.DaggerheartDomainCard
	for _, c := range s.domainCards {
		if c.DomainID == domainID {
			result = append(result, c)
		}
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartDomainCard(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartWeapon(_ context.Context, w storage.DaggerheartWeapon) error {
	s.weapons[w.ID] = w
	return nil
}
func (s *fakeContentStore) GetDaggerheartWeapon(_ context.Context, id string) (storage.DaggerheartWeapon, error) {
	w, ok := s.weapons[id]
	if !ok {
		return storage.DaggerheartWeapon{}, storage.ErrNotFound
	}
	return w, nil
}
func (s *fakeContentStore) ListDaggerheartWeapons(_ context.Context) ([]storage.DaggerheartWeapon, error) {
	result := make([]storage.DaggerheartWeapon, 0, len(s.weapons))
	for _, w := range s.weapons {
		result = append(result, w)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartWeapon(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartArmor(_ context.Context, a storage.DaggerheartArmor) error {
	s.armor[a.ID] = a
	return nil
}
func (s *fakeContentStore) GetDaggerheartArmor(_ context.Context, id string) (storage.DaggerheartArmor, error) {
	a, ok := s.armor[id]
	if !ok {
		return storage.DaggerheartArmor{}, storage.ErrNotFound
	}
	return a, nil
}
func (s *fakeContentStore) ListDaggerheartArmor(_ context.Context) ([]storage.DaggerheartArmor, error) {
	result := make([]storage.DaggerheartArmor, 0, len(s.armor))
	for _, a := range s.armor {
		result = append(result, a)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartArmor(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartItem(_ context.Context, i storage.DaggerheartItem) error {
	s.items[i.ID] = i
	return nil
}
func (s *fakeContentStore) GetDaggerheartItem(_ context.Context, id string) (storage.DaggerheartItem, error) {
	i, ok := s.items[id]
	if !ok {
		return storage.DaggerheartItem{}, storage.ErrNotFound
	}
	return i, nil
}
func (s *fakeContentStore) ListDaggerheartItems(_ context.Context) ([]storage.DaggerheartItem, error) {
	result := make([]storage.DaggerheartItem, 0, len(s.items))
	for _, i := range s.items {
		result = append(result, i)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartItem(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartEnvironment(_ context.Context, e storage.DaggerheartEnvironment) error {
	s.environments[e.ID] = e
	return nil
}
func (s *fakeContentStore) GetDaggerheartEnvironment(_ context.Context, id string) (storage.DaggerheartEnvironment, error) {
	e, ok := s.environments[id]
	if !ok {
		return storage.DaggerheartEnvironment{}, storage.ErrNotFound
	}
	return e, nil
}
func (s *fakeContentStore) ListDaggerheartEnvironments(_ context.Context) ([]storage.DaggerheartEnvironment, error) {
	result := make([]storage.DaggerheartEnvironment, 0, len(s.environments))
	for _, e := range s.environments {
		result = append(result, e)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartEnvironment(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) ListDaggerheartContentStrings(_ context.Context, contentType string, contentIDs []string, locale string) ([]storage.DaggerheartContentString, error) {
	if s == nil {
		return nil, nil
	}
	if len(contentIDs) == 0 {
		return nil, nil
	}
	idSet := make(map[string]struct{}, len(contentIDs))
	for _, id := range contentIDs {
		idSet[id] = struct{}{}
	}
	results := make([]storage.DaggerheartContentString, 0, len(contentIDs))
	for _, entry := range s.contentStrings {
		if entry.ContentType != contentType || entry.Locale != locale {
			continue
		}
		if _, ok := idSet[entry.ContentID]; !ok {
			continue
		}
		results = append(results, entry)
	}
	return results, nil
}

func (s *fakeContentStore) PutDaggerheartContentString(_ context.Context, entry storage.DaggerheartContentString) error {
	if s == nil {
		return nil
	}
	key := fakeContentStringKey{ContentID: entry.ContentID, Field: entry.Field, Locale: entry.Locale}
	s.contentStrings[key] = entry
	return nil
}

func newContentTestService() *DaggerheartContentService {
	cs := newFakeContentStore()
	cs.classes["class-1"] = storage.DaggerheartClass{ID: "class-1", Name: "Guardian"}
	cs.classes["class-2"] = storage.DaggerheartClass{ID: "class-2", Name: "Sorcerer"}
	cs.subclasses["sub-1"] = storage.DaggerheartSubclass{ID: "sub-1", Name: "Bladeweaver"}
	cs.heritages["her-1"] = storage.DaggerheartHeritage{ID: "her-1", Name: "Elf", Kind: "ancestry"}
	cs.experiences["exp-1"] = storage.DaggerheartExperienceEntry{ID: "exp-1", Name: "Wanderer"}
	cs.adversaryEntries["adv-1"] = storage.DaggerheartAdversaryEntry{ID: "adv-1", Name: "Goblin"}
	cs.beastforms["beast-1"] = storage.DaggerheartBeastformEntry{ID: "beast-1", Name: "Wolf"}
	cs.companionExperiences["cexp-1"] = storage.DaggerheartCompanionExperienceEntry{ID: "cexp-1", Name: "Guard"}
	cs.lootEntries["loot-1"] = storage.DaggerheartLootEntry{ID: "loot-1", Name: "Gold"}
	cs.damageTypes["dt-1"] = storage.DaggerheartDamageTypeEntry{ID: "dt-1", Name: "Fire"}
	cs.domains["dom-1"] = storage.DaggerheartDomain{ID: "dom-1", Name: "Valor"}
	cs.domainCards["card-1"] = storage.DaggerheartDomainCard{ID: "card-1", Name: "Fireball", DomainID: "dom-1"}
	cs.weapons["weap-1"] = storage.DaggerheartWeapon{ID: "weap-1", Name: "Blade"}
	cs.armor["armor-1"] = storage.DaggerheartArmor{ID: "armor-1", Name: "Chain Mail"}
	cs.items["item-1"] = storage.DaggerheartItem{ID: "item-1", Name: "Potion"}
	cs.environments["env-1"] = storage.DaggerheartEnvironment{ID: "env-1", Name: "Forest"}

	svc, err := NewDaggerheartContentService(Stores{DaggerheartContent: cs})
	if err != nil {
		panic(err)
	}
	return svc
}
