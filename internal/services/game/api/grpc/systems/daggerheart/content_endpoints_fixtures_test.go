package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// fakeContentStore implements contentstore.DaggerheartContentStore for testing content
// service endpoints. Only the methods needed for tests are implemented with
// real behavior; others return nil/empty to satisfy the interface.
type fakeContentStore struct {
	classes              map[string]contentstore.DaggerheartClass
	subclasses           map[string]contentstore.DaggerheartSubclass
	heritages            map[string]contentstore.DaggerheartHeritage
	experiences          map[string]contentstore.DaggerheartExperienceEntry
	adversaryEntries     map[string]contentstore.DaggerheartAdversaryEntry
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
	contentStrings       map[fakeContentStringKey]contentstore.DaggerheartContentString
}

type fakeContentStringKey struct {
	ContentID string
	Field     string
	Locale    string
}

func newFakeContentStore() *fakeContentStore {
	return &fakeContentStore{
		classes:              make(map[string]contentstore.DaggerheartClass),
		subclasses:           make(map[string]contentstore.DaggerheartSubclass),
		heritages:            make(map[string]contentstore.DaggerheartHeritage),
		experiences:          make(map[string]contentstore.DaggerheartExperienceEntry),
		adversaryEntries:     make(map[string]contentstore.DaggerheartAdversaryEntry),
		beastforms:           make(map[string]contentstore.DaggerheartBeastformEntry),
		companionExperiences: make(map[string]contentstore.DaggerheartCompanionExperienceEntry),
		lootEntries:          make(map[string]contentstore.DaggerheartLootEntry),
		damageTypes:          make(map[string]contentstore.DaggerheartDamageTypeEntry),
		domains:              make(map[string]contentstore.DaggerheartDomain),
		domainCards:          make(map[string]contentstore.DaggerheartDomainCard),
		weapons:              make(map[string]contentstore.DaggerheartWeapon),
		armor:                make(map[string]contentstore.DaggerheartArmor),
		items:                make(map[string]contentstore.DaggerheartItem),
		environments:         make(map[string]contentstore.DaggerheartEnvironment),
		contentStrings:       make(map[fakeContentStringKey]contentstore.DaggerheartContentString),
	}
}

func (s *fakeContentStore) PutDaggerheartClass(_ context.Context, c contentstore.DaggerheartClass) error {
	s.classes[c.ID] = c
	return nil
}
func (s *fakeContentStore) GetDaggerheartClass(_ context.Context, id string) (contentstore.DaggerheartClass, error) {
	c, ok := s.classes[id]
	if !ok {
		return contentstore.DaggerheartClass{}, storage.ErrNotFound
	}
	return c, nil
}
func (s *fakeContentStore) ListDaggerheartClasses(_ context.Context) ([]contentstore.DaggerheartClass, error) {
	result := make([]contentstore.DaggerheartClass, 0, len(s.classes))
	for _, c := range s.classes {
		result = append(result, c)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartClass(_ context.Context, id string) error {
	delete(s.classes, id)
	return nil
}

func (s *fakeContentStore) PutDaggerheartSubclass(_ context.Context, c contentstore.DaggerheartSubclass) error {
	s.subclasses[c.ID] = c
	return nil
}
func (s *fakeContentStore) GetDaggerheartSubclass(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	c, ok := s.subclasses[id]
	if !ok {
		return contentstore.DaggerheartSubclass{}, storage.ErrNotFound
	}
	return c, nil
}
func (s *fakeContentStore) ListDaggerheartSubclasses(_ context.Context) ([]contentstore.DaggerheartSubclass, error) {
	result := make([]contentstore.DaggerheartSubclass, 0, len(s.subclasses))
	for _, c := range s.subclasses {
		result = append(result, c)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartSubclass(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartHeritage(_ context.Context, h contentstore.DaggerheartHeritage) error {
	s.heritages[h.ID] = h
	return nil
}
func (s *fakeContentStore) GetDaggerheartHeritage(_ context.Context, id string) (contentstore.DaggerheartHeritage, error) {
	h, ok := s.heritages[id]
	if !ok {
		return contentstore.DaggerheartHeritage{}, storage.ErrNotFound
	}
	return h, nil
}
func (s *fakeContentStore) ListDaggerheartHeritages(_ context.Context) ([]contentstore.DaggerheartHeritage, error) {
	result := make([]contentstore.DaggerheartHeritage, 0, len(s.heritages))
	for _, h := range s.heritages {
		result = append(result, h)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartHeritage(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartExperience(_ context.Context, e contentstore.DaggerheartExperienceEntry) error {
	s.experiences[e.ID] = e
	return nil
}
func (s *fakeContentStore) GetDaggerheartExperience(_ context.Context, id string) (contentstore.DaggerheartExperienceEntry, error) {
	e, ok := s.experiences[id]
	if !ok {
		return contentstore.DaggerheartExperienceEntry{}, storage.ErrNotFound
	}
	return e, nil
}
func (s *fakeContentStore) ListDaggerheartExperiences(_ context.Context) ([]contentstore.DaggerheartExperienceEntry, error) {
	result := make([]contentstore.DaggerheartExperienceEntry, 0, len(s.experiences))
	for _, e := range s.experiences {
		result = append(result, e)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartExperience(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartAdversaryEntry(_ context.Context, a contentstore.DaggerheartAdversaryEntry) error {
	s.adversaryEntries[a.ID] = a
	return nil
}
func (s *fakeContentStore) GetDaggerheartAdversaryEntry(_ context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error) {
	a, ok := s.adversaryEntries[id]
	if !ok {
		return contentstore.DaggerheartAdversaryEntry{}, storage.ErrNotFound
	}
	return a, nil
}
func (s *fakeContentStore) ListDaggerheartAdversaryEntries(_ context.Context) ([]contentstore.DaggerheartAdversaryEntry, error) {
	result := make([]contentstore.DaggerheartAdversaryEntry, 0, len(s.adversaryEntries))
	for _, a := range s.adversaryEntries {
		result = append(result, a)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartAdversaryEntry(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartBeastform(_ context.Context, b contentstore.DaggerheartBeastformEntry) error {
	s.beastforms[b.ID] = b
	return nil
}
func (s *fakeContentStore) GetDaggerheartBeastform(_ context.Context, id string) (contentstore.DaggerheartBeastformEntry, error) {
	b, ok := s.beastforms[id]
	if !ok {
		return contentstore.DaggerheartBeastformEntry{}, storage.ErrNotFound
	}
	return b, nil
}
func (s *fakeContentStore) ListDaggerheartBeastforms(_ context.Context) ([]contentstore.DaggerheartBeastformEntry, error) {
	result := make([]contentstore.DaggerheartBeastformEntry, 0, len(s.beastforms))
	for _, b := range s.beastforms {
		result = append(result, b)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartBeastform(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartCompanionExperience(_ context.Context, e contentstore.DaggerheartCompanionExperienceEntry) error {
	s.companionExperiences[e.ID] = e
	return nil
}
func (s *fakeContentStore) GetDaggerheartCompanionExperience(_ context.Context, id string) (contentstore.DaggerheartCompanionExperienceEntry, error) {
	e, ok := s.companionExperiences[id]
	if !ok {
		return contentstore.DaggerheartCompanionExperienceEntry{}, storage.ErrNotFound
	}
	return e, nil
}
func (s *fakeContentStore) ListDaggerheartCompanionExperiences(_ context.Context) ([]contentstore.DaggerheartCompanionExperienceEntry, error) {
	result := make([]contentstore.DaggerheartCompanionExperienceEntry, 0, len(s.companionExperiences))
	for _, e := range s.companionExperiences {
		result = append(result, e)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartCompanionExperience(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartLootEntry(_ context.Context, l contentstore.DaggerheartLootEntry) error {
	s.lootEntries[l.ID] = l
	return nil
}
func (s *fakeContentStore) GetDaggerheartLootEntry(_ context.Context, id string) (contentstore.DaggerheartLootEntry, error) {
	l, ok := s.lootEntries[id]
	if !ok {
		return contentstore.DaggerheartLootEntry{}, storage.ErrNotFound
	}
	return l, nil
}
func (s *fakeContentStore) ListDaggerheartLootEntries(_ context.Context) ([]contentstore.DaggerheartLootEntry, error) {
	result := make([]contentstore.DaggerheartLootEntry, 0, len(s.lootEntries))
	for _, l := range s.lootEntries {
		result = append(result, l)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartLootEntry(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartDamageType(_ context.Context, d contentstore.DaggerheartDamageTypeEntry) error {
	s.damageTypes[d.ID] = d
	return nil
}
func (s *fakeContentStore) GetDaggerheartDamageType(_ context.Context, id string) (contentstore.DaggerheartDamageTypeEntry, error) {
	d, ok := s.damageTypes[id]
	if !ok {
		return contentstore.DaggerheartDamageTypeEntry{}, storage.ErrNotFound
	}
	return d, nil
}
func (s *fakeContentStore) ListDaggerheartDamageTypes(_ context.Context) ([]contentstore.DaggerheartDamageTypeEntry, error) {
	result := make([]contentstore.DaggerheartDamageTypeEntry, 0, len(s.damageTypes))
	for _, d := range s.damageTypes {
		result = append(result, d)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartDamageType(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) PutDaggerheartDomain(_ context.Context, d contentstore.DaggerheartDomain) error {
	s.domains[d.ID] = d
	return nil
}
func (s *fakeContentStore) GetDaggerheartDomain(_ context.Context, id string) (contentstore.DaggerheartDomain, error) {
	d, ok := s.domains[id]
	if !ok {
		return contentstore.DaggerheartDomain{}, storage.ErrNotFound
	}
	return d, nil
}
func (s *fakeContentStore) ListDaggerheartDomains(_ context.Context) ([]contentstore.DaggerheartDomain, error) {
	result := make([]contentstore.DaggerheartDomain, 0, len(s.domains))
	for _, d := range s.domains {
		result = append(result, d)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartDomain(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartDomainCard(_ context.Context, c contentstore.DaggerheartDomainCard) error {
	s.domainCards[c.ID] = c
	return nil
}
func (s *fakeContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (contentstore.DaggerheartDomainCard, error) {
	c, ok := s.domainCards[id]
	if !ok {
		return contentstore.DaggerheartDomainCard{}, storage.ErrNotFound
	}
	return c, nil
}
func (s *fakeContentStore) ListDaggerheartDomainCards(_ context.Context) ([]contentstore.DaggerheartDomainCard, error) {
	result := make([]contentstore.DaggerheartDomainCard, 0, len(s.domainCards))
	for _, c := range s.domainCards {
		result = append(result, c)
	}
	return result, nil
}
func (s *fakeContentStore) ListDaggerheartDomainCardsByDomain(_ context.Context, domainID string) ([]contentstore.DaggerheartDomainCard, error) {
	var result []contentstore.DaggerheartDomainCard
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

func (s *fakeContentStore) PutDaggerheartWeapon(_ context.Context, w contentstore.DaggerheartWeapon) error {
	s.weapons[w.ID] = w
	return nil
}
func (s *fakeContentStore) GetDaggerheartWeapon(_ context.Context, id string) (contentstore.DaggerheartWeapon, error) {
	w, ok := s.weapons[id]
	if !ok {
		return contentstore.DaggerheartWeapon{}, storage.ErrNotFound
	}
	return w, nil
}
func (s *fakeContentStore) ListDaggerheartWeapons(_ context.Context) ([]contentstore.DaggerheartWeapon, error) {
	result := make([]contentstore.DaggerheartWeapon, 0, len(s.weapons))
	for _, w := range s.weapons {
		result = append(result, w)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartWeapon(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartArmor(_ context.Context, a contentstore.DaggerheartArmor) error {
	s.armor[a.ID] = a
	return nil
}
func (s *fakeContentStore) GetDaggerheartArmor(_ context.Context, id string) (contentstore.DaggerheartArmor, error) {
	a, ok := s.armor[id]
	if !ok {
		return contentstore.DaggerheartArmor{}, storage.ErrNotFound
	}
	return a, nil
}
func (s *fakeContentStore) ListDaggerheartArmor(_ context.Context) ([]contentstore.DaggerheartArmor, error) {
	result := make([]contentstore.DaggerheartArmor, 0, len(s.armor))
	for _, a := range s.armor {
		result = append(result, a)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartArmor(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartItem(_ context.Context, i contentstore.DaggerheartItem) error {
	s.items[i.ID] = i
	return nil
}
func (s *fakeContentStore) GetDaggerheartItem(_ context.Context, id string) (contentstore.DaggerheartItem, error) {
	i, ok := s.items[id]
	if !ok {
		return contentstore.DaggerheartItem{}, storage.ErrNotFound
	}
	return i, nil
}
func (s *fakeContentStore) ListDaggerheartItems(_ context.Context) ([]contentstore.DaggerheartItem, error) {
	result := make([]contentstore.DaggerheartItem, 0, len(s.items))
	for _, i := range s.items {
		result = append(result, i)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartItem(_ context.Context, _ string) error { return nil }

func (s *fakeContentStore) PutDaggerheartEnvironment(_ context.Context, e contentstore.DaggerheartEnvironment) error {
	s.environments[e.ID] = e
	return nil
}
func (s *fakeContentStore) GetDaggerheartEnvironment(_ context.Context, id string) (contentstore.DaggerheartEnvironment, error) {
	e, ok := s.environments[id]
	if !ok {
		return contentstore.DaggerheartEnvironment{}, storage.ErrNotFound
	}
	return e, nil
}
func (s *fakeContentStore) ListDaggerheartEnvironments(_ context.Context) ([]contentstore.DaggerheartEnvironment, error) {
	result := make([]contentstore.DaggerheartEnvironment, 0, len(s.environments))
	for _, e := range s.environments {
		result = append(result, e)
	}
	return result, nil
}
func (s *fakeContentStore) DeleteDaggerheartEnvironment(_ context.Context, _ string) error {
	return nil
}

func (s *fakeContentStore) ListDaggerheartContentStrings(_ context.Context, contentType string, contentIDs []string, locale string) ([]contentstore.DaggerheartContentString, error) {
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
	results := make([]contentstore.DaggerheartContentString, 0, len(contentIDs))
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

func (s *fakeContentStore) PutDaggerheartContentString(_ context.Context, entry contentstore.DaggerheartContentString) error {
	if s == nil {
		return nil
	}
	key := fakeContentStringKey{ContentID: entry.ContentID, Field: entry.Field, Locale: entry.Locale}
	s.contentStrings[key] = entry
	return nil
}

func newContentTestService() *DaggerheartContentService {
	cs := newFakeContentStore()
	cs.classes["class-1"] = contentstore.DaggerheartClass{ID: "class-1", Name: "Guardian"}
	cs.classes["class-2"] = contentstore.DaggerheartClass{ID: "class-2", Name: "Sorcerer"}
	cs.subclasses["sub-1"] = contentstore.DaggerheartSubclass{ID: "sub-1", Name: "Bladeweaver"}
	cs.heritages["her-1"] = contentstore.DaggerheartHeritage{ID: "her-1", Name: "Elf", Kind: "ancestry"}
	cs.experiences["exp-1"] = contentstore.DaggerheartExperienceEntry{ID: "exp-1", Name: "Wanderer"}
	cs.adversaryEntries["adv-1"] = contentstore.DaggerheartAdversaryEntry{ID: "adv-1", Name: "Goblin"}
	cs.beastforms["beast-1"] = contentstore.DaggerheartBeastformEntry{ID: "beast-1", Name: "Wolf"}
	cs.companionExperiences["cexp-1"] = contentstore.DaggerheartCompanionExperienceEntry{ID: "cexp-1", Name: "Guard"}
	cs.lootEntries["loot-1"] = contentstore.DaggerheartLootEntry{ID: "loot-1", Name: "Gold"}
	cs.damageTypes["dt-1"] = contentstore.DaggerheartDamageTypeEntry{ID: "dt-1", Name: "Fire"}
	cs.domains["dom-1"] = contentstore.DaggerheartDomain{ID: "dom-1", Name: "Valor"}
	cs.domainCards["card-1"] = contentstore.DaggerheartDomainCard{ID: "card-1", Name: "Fireball", DomainID: "dom-1"}
	cs.weapons["weap-1"] = contentstore.DaggerheartWeapon{ID: "weap-1", Name: "Blade"}
	cs.armor["armor-1"] = contentstore.DaggerheartArmor{ID: "armor-1", Name: "Chain Mail"}
	cs.items["item-1"] = contentstore.DaggerheartItem{ID: "item-1", Name: "Potion"}
	cs.environments["env-1"] = contentstore.DaggerheartEnvironment{ID: "env-1", Name: "Forest"}

	svc, err := NewDaggerheartContentService(cs)
	if err != nil {
		panic(err)
	}
	return svc
}
