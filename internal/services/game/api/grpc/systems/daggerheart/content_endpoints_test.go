package daggerheart

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
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

	return NewDaggerheartContentService(Stores{DaggerheartContent: cs})
}

// --- GetContentCatalog tests ---

func TestGetContentCatalog_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	_, err := svc.GetContentCatalog(context.Background(), &pb.GetDaggerheartContentCatalogRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestGetContentCatalog_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetContentCatalog(context.Background(), &pb.GetDaggerheartContentCatalogRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	catalog := resp.GetCatalog()
	if len(catalog.GetClasses()) != 2 {
		t.Errorf("classes = %d, want 2", len(catalog.GetClasses()))
	}
	if len(catalog.GetSubclasses()) != 1 {
		t.Errorf("subclasses = %d, want 1", len(catalog.GetSubclasses()))
	}
	if len(catalog.GetHeritages()) != 1 {
		t.Errorf("heritages = %d, want 1", len(catalog.GetHeritages()))
	}
	if len(catalog.GetWeapons()) != 1 {
		t.Errorf("weapons = %d, want 1", len(catalog.GetWeapons()))
	}
	if len(catalog.GetEnvironments()) != 1 {
		t.Errorf("environments = %d, want 1", len(catalog.GetEnvironments()))
	}
}

// --- GetClass / ListClasses ---

func TestGetClass_NilRequest(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.GetClass(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetClass_EmptyID(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.GetClass(context.Background(), &pb.GetDaggerheartClassRequest{Id: ""})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetClass_NotFound(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.GetClass(context.Background(), &pb.GetDaggerheartClassRequest{Id: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetClass_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetClass(context.Background(), &pb.GetDaggerheartClassRequest{Id: "class-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetClass().GetName() != "Guardian" {
		t.Errorf("name = %q, want Guardian", resp.GetClass().GetName())
	}
}

func TestGetClass_LocaleOverride(t *testing.T) {
	svc := newContentTestService()
	store, ok := svc.stores.DaggerheartContent.(*fakeContentStore)
	if !ok {
		t.Fatalf("expected fake content store, got %T", svc.stores.DaggerheartContent)
	}
	locale := i18n.LocaleString(commonv1.Locale_LOCALE_PT_BR)
	if err := store.PutDaggerheartContentString(context.Background(), storage.DaggerheartContentString{
		ContentID:   "class-1",
		ContentType: "class",
		Field:       "name",
		Locale:      locale,
		Text:        "Guardiao",
	}); err != nil {
		t.Fatalf("put content string: %v", err)
	}

	resp, err := svc.GetClass(context.Background(), &pb.GetDaggerheartClassRequest{
		Id:     "class-1",
		Locale: commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetClass().GetName() != "Guardiao" {
		t.Errorf("name = %q, want Guardiao", resp.GetClass().GetName())
	}
}

func TestListClasses_NilRequest(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.ListClasses(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListClasses_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetClasses()) != 2 {
		t.Errorf("classes = %d, want 2", len(resp.GetClasses()))
	}
	if resp.GetTotalSize() != 2 {
		t.Errorf("total_size = %d, want 2", resp.GetTotalSize())
	}
}

func TestListClasses_WithPagination(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetClasses()) != 1 {
		t.Errorf("classes = %d, want 1", len(resp.GetClasses()))
	}
	if resp.GetNextPageToken() == "" {
		t.Error("expected next_page_token")
	}
}

// --- GetSubclass / ListSubclasses ---

func TestGetSubclass_NilRequest(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.GetSubclass(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSubclass_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetSubclass(context.Background(), &pb.GetDaggerheartSubclassRequest{Id: "sub-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSubclass().GetName() != "Bladeweaver" {
		t.Errorf("name = %q, want Bladeweaver", resp.GetSubclass().GetName())
	}
}

func TestListSubclasses_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListSubclasses(context.Background(), &pb.ListDaggerheartSubclassesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetSubclasses()) != 1 {
		t.Errorf("subclasses = %d, want 1", len(resp.GetSubclasses()))
	}
}

// --- GetHeritage / ListHeritages ---

func TestGetHeritage_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetHeritage(context.Background(), &pb.GetDaggerheartHeritageRequest{Id: "her-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetHeritage().GetName() != "Elf" {
		t.Errorf("name = %q, want Elf", resp.GetHeritage().GetName())
	}
}

func TestListHeritages_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListHeritages(context.Background(), &pb.ListDaggerheartHeritagesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetHeritages()) != 1 {
		t.Errorf("heritages = %d, want 1", len(resp.GetHeritages()))
	}
}

// --- GetExperience / ListExperiences ---

func TestGetExperience_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetExperience(context.Background(), &pb.GetDaggerheartExperienceRequest{Id: "exp-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetExperience().GetName() != "Wanderer" {
		t.Errorf("name = %q, want Wanderer", resp.GetExperience().GetName())
	}
}

func TestListExperiences_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListExperiences(context.Background(), &pb.ListDaggerheartExperiencesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetExperiences()) != 1 {
		t.Errorf("experiences = %d, want 1", len(resp.GetExperiences()))
	}
}

// --- GetAdversary / ListAdversaries (content) ---

func TestGetContentAdversary_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetAdversary(context.Background(), &pb.GetDaggerheartAdversaryRequest{Id: "adv-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetAdversary().GetName() != "Goblin" {
		t.Errorf("name = %q, want Goblin", resp.GetAdversary().GetName())
	}
}

func TestListContentAdversaries_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListAdversaries(context.Background(), &pb.ListDaggerheartAdversariesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetAdversaries()) != 1 {
		t.Errorf("adversaries = %d, want 1", len(resp.GetAdversaries()))
	}
}

// --- GetBeastform / ListBeastforms ---

func TestGetBeastform_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetBeastform(context.Background(), &pb.GetDaggerheartBeastformRequest{Id: "beast-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetBeastform().GetName() != "Wolf" {
		t.Errorf("name = %q, want Wolf", resp.GetBeastform().GetName())
	}
}

func TestListBeastforms_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListBeastforms(context.Background(), &pb.ListDaggerheartBeastformsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetBeastforms()) != 1 {
		t.Errorf("beastforms = %d, want 1", len(resp.GetBeastforms()))
	}
}

// --- GetCompanionExperience / ListCompanionExperiences ---

func TestGetCompanionExperience_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetCompanionExperience(context.Background(), &pb.GetDaggerheartCompanionExperienceRequest{Id: "cexp-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetExperience().GetName() != "Guard" {
		t.Errorf("name = %q, want Guard", resp.GetExperience().GetName())
	}
}

func TestListCompanionExperiences_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListCompanionExperiences(context.Background(), &pb.ListDaggerheartCompanionExperiencesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetExperiences()) != 1 {
		t.Errorf("companion experiences = %d, want 1", len(resp.GetExperiences()))
	}
}

// --- GetLootEntry / ListLootEntries ---

func TestGetLootEntry_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetLootEntry(context.Background(), &pb.GetDaggerheartLootEntryRequest{Id: "loot-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetEntry().GetName() != "Gold" {
		t.Errorf("name = %q, want Gold", resp.GetEntry().GetName())
	}
}

func TestListLootEntries_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListLootEntries(context.Background(), &pb.ListDaggerheartLootEntriesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetEntries()) != 1 {
		t.Errorf("loot entries = %d, want 1", len(resp.GetEntries()))
	}
}

// --- GetDamageType / ListDamageTypes ---

func TestGetDamageTypeEntry_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetDamageType(context.Background(), &pb.GetDaggerheartDamageTypeRequest{Id: "dt-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetDamageType().GetName() != "Fire" {
		t.Errorf("name = %q, want Fire", resp.GetDamageType().GetName())
	}
}

func TestListDamageTypes_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDamageTypes(context.Background(), &pb.ListDaggerheartDamageTypesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDamageTypes()) != 1 {
		t.Errorf("damage types = %d, want 1", len(resp.GetDamageTypes()))
	}
}

// --- GetDomain / ListDomains ---

func TestGetDomain_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetDomain(context.Background(), &pb.GetDaggerheartDomainRequest{Id: "dom-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetDomain().GetName() != "Valor" {
		t.Errorf("name = %q, want Valor", resp.GetDomain().GetName())
	}
}

func TestListDomains_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDomains(context.Background(), &pb.ListDaggerheartDomainsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDomains()) != 1 {
		t.Errorf("domains = %d, want 1", len(resp.GetDomains()))
	}
}

// --- GetDomainCard / ListDomainCards ---

func TestGetDomainCard_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetDomainCard(context.Background(), &pb.GetDaggerheartDomainCardRequest{Id: "card-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetDomainCard().GetName() != "Fireball" {
		t.Errorf("name = %q, want Fireball", resp.GetDomainCard().GetName())
	}
}

func TestListDomainCards_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDomainCards(context.Background(), &pb.ListDaggerheartDomainCardsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDomainCards()) != 1 {
		t.Errorf("domain cards = %d, want 1", len(resp.GetDomainCards()))
	}
}

// --- GetWeapon / ListWeapons ---

func TestGetWeapon_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetWeapon(context.Background(), &pb.GetDaggerheartWeaponRequest{Id: "weap-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetWeapon().GetName() != "Blade" {
		t.Errorf("name = %q, want Blade", resp.GetWeapon().GetName())
	}
}

func TestListWeapons_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListWeapons(context.Background(), &pb.ListDaggerheartWeaponsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetWeapons()) != 1 {
		t.Errorf("weapons = %d, want 1", len(resp.GetWeapons()))
	}
}

// --- GetArmor / ListArmor ---

func TestGetArmor_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetArmor(context.Background(), &pb.GetDaggerheartArmorRequest{Id: "armor-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetArmor().GetName() != "Chain Mail" {
		t.Errorf("name = %q, want Chain Mail", resp.GetArmor().GetName())
	}
}

func TestListArmor_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListArmor(context.Background(), &pb.ListDaggerheartArmorRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetArmor()) != 1 {
		t.Errorf("armor = %d, want 1", len(resp.GetArmor()))
	}
}

// --- GetItem / ListItems ---

func TestGetItem_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetItem(context.Background(), &pb.GetDaggerheartItemRequest{Id: "item-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetItem().GetName() != "Potion" {
		t.Errorf("name = %q, want Potion", resp.GetItem().GetName())
	}
}

func TestListItems_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListItems(context.Background(), &pb.ListDaggerheartItemsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetItems()) != 1 {
		t.Errorf("items = %d, want 1", len(resp.GetItems()))
	}
}

// --- GetEnvironment / ListEnvironments ---

func TestGetEnvironment_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetEnvironment(context.Background(), &pb.GetDaggerheartEnvironmentRequest{Id: "env-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetEnvironment().GetName() != "Forest" {
		t.Errorf("name = %q, want Forest", resp.GetEnvironment().GetName())
	}
}

func TestListEnvironments_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListEnvironments(context.Background(), &pb.ListDaggerheartEnvironmentsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetEnvironments()) != 1 {
		t.Errorf("environments = %d, want 1", len(resp.GetEnvironments()))
	}
}

// --- NewDaggerheartContentService ---

func TestNewDaggerheartContentService(t *testing.T) {
	cs := newFakeContentStore()
	svc := NewDaggerheartContentService(Stores{DaggerheartContent: cs})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

// --- Edge case: nil requests for Get methods ---

func TestGetContentEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetSubclass", func() error { _, err := svc.GetSubclass(context.Background(), nil); return err }},
		{"GetHeritage", func() error { _, err := svc.GetHeritage(context.Background(), nil); return err }},
		{"GetExperience", func() error { _, err := svc.GetExperience(context.Background(), nil); return err }},
		{"GetAdversary", func() error {
			_, err := svc.GetAdversary(context.Background(), nil)
			return err
		}},
		{"GetBeastform", func() error { _, err := svc.GetBeastform(context.Background(), nil); return err }},
		{"GetCompanionExperience", func() error {
			_, err := svc.GetCompanionExperience(context.Background(), nil)
			return err
		}},
		{"GetLootEntry", func() error { _, err := svc.GetLootEntry(context.Background(), nil); return err }},
		{"GetDamageType", func() error { _, err := svc.GetDamageType(context.Background(), nil); return err }},
		{"GetDomain", func() error { _, err := svc.GetDomain(context.Background(), nil); return err }},
		{"GetDomainCard", func() error { _, err := svc.GetDomainCard(context.Background(), nil); return err }},
		{"GetWeapon", func() error { _, err := svc.GetWeapon(context.Background(), nil); return err }},
		{"GetArmor", func() error { _, err := svc.GetArmor(context.Background(), nil); return err }},
		{"GetItem", func() error { _, err := svc.GetItem(context.Background(), nil); return err }},
		{"GetEnvironment", func() error { _, err := svc.GetEnvironment(context.Background(), nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

// --- Edge case: nil requests for List methods ---

func TestListClasses_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	classes := resp.GetClasses()
	if len(classes) != 2 {
		t.Fatalf("classes = %d, want 2", len(classes))
	}
	// Descending: Sorcerer before Guardian.
	if classes[0].GetName() != "Sorcerer" {
		t.Errorf("first class = %q, want Sorcerer", classes[0].GetName())
	}
}

func TestListClasses_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		Filter: `name = "Guardian"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetClasses()) != 1 {
		t.Fatalf("classes = %d, want 1", len(resp.GetClasses()))
	}
	if resp.GetClasses()[0].GetName() != "Guardian" {
		t.Errorf("class = %q, want Guardian", resp.GetClasses()[0].GetName())
	}
}

func TestListClasses_InvalidFilter(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		Filter: "invalid @@@ filter",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListClasses_InvalidOrderBy(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		OrderBy: "unknown_column",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListClasses_PaginationSecondPage(t *testing.T) {
	svc := newContentTestService()
	firstPage, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if firstPage.GetNextPageToken() == "" {
		t.Fatal("expected next_page_token")
	}

	secondPage, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		PageSize:  1,
		PageToken: firstPage.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(secondPage.GetClasses()) != 1 {
		t.Errorf("classes on second page = %d, want 1", len(secondPage.GetClasses()))
	}
	// Second page should have a different class than first page.
	if secondPage.GetClasses()[0].GetId() == firstPage.GetClasses()[0].GetId() {
		t.Error("second page returned same class as first page")
	}
}

func TestListSubclasses_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListSubclasses(context.Background(), &pb.ListDaggerheartSubclassesRequest{
		Filter: `name = "Bladeweaver"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetSubclasses()) != 1 {
		t.Fatalf("subclasses = %d, want 1", len(resp.GetSubclasses()))
	}
}

func TestListHeritages_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListHeritages(context.Background(), &pb.ListDaggerheartHeritagesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetHeritages()) != 1 {
		t.Fatalf("heritages = %d, want 1", len(resp.GetHeritages()))
	}
}

func TestListExperiences_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListExperiences(context.Background(), &pb.ListDaggerheartExperiencesRequest{
		Filter: `name = "Wanderer"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetExperiences()) != 1 {
		t.Fatalf("experiences = %d, want 1", len(resp.GetExperiences()))
	}
}

func TestListAdversaries_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListAdversaries(context.Background(), &pb.ListDaggerheartAdversariesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetAdversaries()) != 1 {
		t.Fatalf("adversaries = %d, want 1", len(resp.GetAdversaries()))
	}
}

func TestListWeapons_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListWeapons(context.Background(), &pb.ListDaggerheartWeaponsRequest{
		Filter: `name = "Blade"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetWeapons()) != 1 {
		t.Fatalf("weapons = %d, want 1", len(resp.GetWeapons()))
	}
}

func TestListArmor_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListArmor(context.Background(), &pb.ListDaggerheartArmorRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetArmor()) != 1 {
		t.Fatalf("armor = %d, want 1", len(resp.GetArmor()))
	}
}

func TestListItems_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListItems(context.Background(), &pb.ListDaggerheartItemsRequest{
		Filter: `name = "Potion"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("items = %d, want 1", len(resp.GetItems()))
	}
}

func TestListEnvironments_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListEnvironments(context.Background(), &pb.ListDaggerheartEnvironmentsRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetEnvironments()) != 1 {
		t.Fatalf("environments = %d, want 1", len(resp.GetEnvironments()))
	}
}

func TestListDomains_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDomains(context.Background(), &pb.ListDaggerheartDomainsRequest{
		Filter: `name = "Valor"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDomains()) != 1 {
		t.Fatalf("domains = %d, want 1", len(resp.GetDomains()))
	}
}

func TestListDomainCards_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDomainCards(context.Background(), &pb.ListDaggerheartDomainCardsRequest{
		Filter: `name = "Fireball"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDomainCards()) != 1 {
		t.Fatalf("domain cards = %d, want 1", len(resp.GetDomainCards()))
	}
}

func TestListDamageTypes_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDamageTypes(context.Background(), &pb.ListDaggerheartDamageTypesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDamageTypes()) != 1 {
		t.Fatalf("damage types = %d, want 1", len(resp.GetDamageTypes()))
	}
}

func TestListBeastforms_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListBeastforms(context.Background(), &pb.ListDaggerheartBeastformsRequest{
		Filter: `name = "Wolf"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetBeastforms()) != 1 {
		t.Fatalf("beastforms = %d, want 1", len(resp.GetBeastforms()))
	}
}

func TestListCompanionExperiences_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListCompanionExperiences(context.Background(), &pb.ListDaggerheartCompanionExperiencesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetExperiences()) != 1 {
		t.Fatalf("companion experiences = %d, want 1", len(resp.GetExperiences()))
	}
}

func TestListLootEntries_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListLootEntries(context.Background(), &pb.ListDaggerheartLootEntriesRequest{
		Filter: `name = "Gold"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetEntries()) != 1 {
		t.Fatalf("loot entries = %d, want 1", len(resp.GetEntries()))
	}
}

func TestGetContentCatalog_WithTypes(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetContentCatalog(context.Background(), &pb.GetDaggerheartContentCatalogRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	catalog := resp.GetCatalog()
	if catalog == nil {
		t.Fatal("expected non-nil catalog")
	}
	// Verify catalog has content types from the seeded data.
	if len(catalog.GetClasses()) == 0 {
		t.Error("expected non-empty classes in catalog")
	}
	if len(catalog.GetWeapons()) == 0 {
		t.Error("expected non-empty weapons in catalog")
	}
}

func TestListContentEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListSubclasses", func() error { _, err := svc.ListSubclasses(context.Background(), nil); return err }},
		{"ListHeritages", func() error { _, err := svc.ListHeritages(context.Background(), nil); return err }},
		{"ListExperiences", func() error { _, err := svc.ListExperiences(context.Background(), nil); return err }},
		{"ListAdversaries", func() error {
			_, err := svc.ListAdversaries(context.Background(), nil)
			return err
		}},
		{"ListBeastforms", func() error { _, err := svc.ListBeastforms(context.Background(), nil); return err }},
		{"ListCompanionExperiences", func() error {
			_, err := svc.ListCompanionExperiences(context.Background(), nil)
			return err
		}},
		{"ListLootEntries", func() error { _, err := svc.ListLootEntries(context.Background(), nil); return err }},
		{"ListDamageTypes", func() error { _, err := svc.ListDamageTypes(context.Background(), nil); return err }},
		{"ListDomains", func() error { _, err := svc.ListDomains(context.Background(), nil); return err }},
		{"ListDomainCards", func() error { _, err := svc.ListDomainCards(context.Background(), nil); return err }},
		{"ListWeapons", func() error { _, err := svc.ListWeapons(context.Background(), nil); return err }},
		{"ListArmor", func() error { _, err := svc.ListArmor(context.Background(), nil); return err }},
		{"ListItems", func() error { _, err := svc.ListItems(context.Background(), nil); return err }},
		{"ListEnvironments", func() error { _, err := svc.ListEnvironments(context.Background(), nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

// TestListContentEndpoints_NoStore verifies each List* endpoint returns Internal when no store is configured.
func TestListContentEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{} // no stores
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListClasses", func() error { _, err := svc.ListClasses(ctx, &pb.ListDaggerheartClassesRequest{}); return err }},
		{"ListSubclasses", func() error { _, err := svc.ListSubclasses(ctx, &pb.ListDaggerheartSubclassesRequest{}); return err }},
		{"ListHeritages", func() error { _, err := svc.ListHeritages(ctx, &pb.ListDaggerheartHeritagesRequest{}); return err }},
		{"ListExperiences", func() error { _, err := svc.ListExperiences(ctx, &pb.ListDaggerheartExperiencesRequest{}); return err }},
		{"ListAdversaries", func() error { _, err := svc.ListAdversaries(ctx, &pb.ListDaggerheartAdversariesRequest{}); return err }},
		{"ListBeastforms", func() error { _, err := svc.ListBeastforms(ctx, &pb.ListDaggerheartBeastformsRequest{}); return err }},
		{"ListCompanionExperiences", func() error {
			_, err := svc.ListCompanionExperiences(ctx, &pb.ListDaggerheartCompanionExperiencesRequest{})
			return err
		}},
		{"ListLootEntries", func() error { _, err := svc.ListLootEntries(ctx, &pb.ListDaggerheartLootEntriesRequest{}); return err }},
		{"ListDamageTypes", func() error { _, err := svc.ListDamageTypes(ctx, &pb.ListDaggerheartDamageTypesRequest{}); return err }},
		{"ListDomains", func() error { _, err := svc.ListDomains(ctx, &pb.ListDaggerheartDomainsRequest{}); return err }},
		{"ListDomainCards", func() error { _, err := svc.ListDomainCards(ctx, &pb.ListDaggerheartDomainCardsRequest{}); return err }},
		{"ListWeapons", func() error { _, err := svc.ListWeapons(ctx, &pb.ListDaggerheartWeaponsRequest{}); return err }},
		{"ListArmor", func() error { _, err := svc.ListArmor(ctx, &pb.ListDaggerheartArmorRequest{}); return err }},
		{"ListItems", func() error { _, err := svc.ListItems(ctx, &pb.ListDaggerheartItemsRequest{}); return err }},
		{"ListEnvironments", func() error {
			_, err := svc.ListEnvironments(ctx, &pb.ListDaggerheartEnvironmentsRequest{})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

// TestGetContentEndpoints_NoStore verifies each Get* endpoint returns Internal when no store is configured.
func TestGetContentEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{} // no stores
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetClass", func() error { _, err := svc.GetClass(ctx, &pb.GetDaggerheartClassRequest{Id: "x"}); return err }},
		{"GetSubclass", func() error { _, err := svc.GetSubclass(ctx, &pb.GetDaggerheartSubclassRequest{Id: "x"}); return err }},
		{"GetHeritage", func() error { _, err := svc.GetHeritage(ctx, &pb.GetDaggerheartHeritageRequest{Id: "x"}); return err }},
		{"GetExperience", func() error {
			_, err := svc.GetExperience(ctx, &pb.GetDaggerheartExperienceRequest{Id: "x"})
			return err
		}},
		{"GetAdversary", func() error { _, err := svc.GetAdversary(ctx, &pb.GetDaggerheartAdversaryRequest{Id: "x"}); return err }},
		{"GetBeastform", func() error { _, err := svc.GetBeastform(ctx, &pb.GetDaggerheartBeastformRequest{Id: "x"}); return err }},
		{"GetCompanionExperience", func() error {
			_, err := svc.GetCompanionExperience(ctx, &pb.GetDaggerheartCompanionExperienceRequest{Id: "x"})
			return err
		}},
		{"GetLootEntry", func() error { _, err := svc.GetLootEntry(ctx, &pb.GetDaggerheartLootEntryRequest{Id: "x"}); return err }},
		{"GetDamageType", func() error {
			_, err := svc.GetDamageType(ctx, &pb.GetDaggerheartDamageTypeRequest{Id: "x"})
			return err
		}},
		{"GetDomain", func() error { _, err := svc.GetDomain(ctx, &pb.GetDaggerheartDomainRequest{Id: "x"}); return err }},
		{"GetDomainCard", func() error {
			_, err := svc.GetDomainCard(ctx, &pb.GetDaggerheartDomainCardRequest{Id: "x"})
			return err
		}},
		{"GetWeapon", func() error { _, err := svc.GetWeapon(ctx, &pb.GetDaggerheartWeaponRequest{Id: "x"}); return err }},
		{"GetArmor", func() error { _, err := svc.GetArmor(ctx, &pb.GetDaggerheartArmorRequest{Id: "x"}); return err }},
		{"GetItem", func() error { _, err := svc.GetItem(ctx, &pb.GetDaggerheartItemRequest{Id: "x"}); return err }},
		{"GetEnvironment", func() error {
			_, err := svc.GetEnvironment(ctx, &pb.GetDaggerheartEnvironmentRequest{Id: "x"})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

// TestGetContentEndpoints_EmptyID verifies each Get* endpoint returns InvalidArgument for empty IDs.
func TestGetContentEndpoints_EmptyID(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetSubclass", func() error { _, err := svc.GetSubclass(ctx, &pb.GetDaggerheartSubclassRequest{Id: ""}); return err }},
		{"GetHeritage", func() error { _, err := svc.GetHeritage(ctx, &pb.GetDaggerheartHeritageRequest{Id: ""}); return err }},
		{"GetExperience", func() error {
			_, err := svc.GetExperience(ctx, &pb.GetDaggerheartExperienceRequest{Id: ""})
			return err
		}},
		{"GetAdversary", func() error { _, err := svc.GetAdversary(ctx, &pb.GetDaggerheartAdversaryRequest{Id: ""}); return err }},
		{"GetBeastform", func() error { _, err := svc.GetBeastform(ctx, &pb.GetDaggerheartBeastformRequest{Id: ""}); return err }},
		{"GetCompanionExperience", func() error {
			_, err := svc.GetCompanionExperience(ctx, &pb.GetDaggerheartCompanionExperienceRequest{Id: ""})
			return err
		}},
		{"GetLootEntry", func() error { _, err := svc.GetLootEntry(ctx, &pb.GetDaggerheartLootEntryRequest{Id: ""}); return err }},
		{"GetDamageType", func() error {
			_, err := svc.GetDamageType(ctx, &pb.GetDaggerheartDamageTypeRequest{Id: ""})
			return err
		}},
		{"GetDomain", func() error { _, err := svc.GetDomain(ctx, &pb.GetDaggerheartDomainRequest{Id: ""}); return err }},
		{"GetDomainCard", func() error {
			_, err := svc.GetDomainCard(ctx, &pb.GetDaggerheartDomainCardRequest{Id: ""})
			return err
		}},
		{"GetWeapon", func() error { _, err := svc.GetWeapon(ctx, &pb.GetDaggerheartWeaponRequest{Id: ""}); return err }},
		{"GetArmor", func() error { _, err := svc.GetArmor(ctx, &pb.GetDaggerheartArmorRequest{Id: ""}); return err }},
		{"GetItem", func() error { _, err := svc.GetItem(ctx, &pb.GetDaggerheartItemRequest{Id: ""}); return err }},
		{"GetEnvironment", func() error {
			_, err := svc.GetEnvironment(ctx, &pb.GetDaggerheartEnvironmentRequest{Id: ""})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

// TestGetContentEndpoints_NotFound verifies each Get* endpoint returns NotFound for missing IDs.
func TestGetContentEndpoints_NotFound(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetSubclass", func() error {
			_, err := svc.GetSubclass(ctx, &pb.GetDaggerheartSubclassRequest{Id: "missing"})
			return err
		}},
		{"GetHeritage", func() error {
			_, err := svc.GetHeritage(ctx, &pb.GetDaggerheartHeritageRequest{Id: "missing"})
			return err
		}},
		{"GetExperience", func() error {
			_, err := svc.GetExperience(ctx, &pb.GetDaggerheartExperienceRequest{Id: "missing"})
			return err
		}},
		{"GetAdversary", func() error {
			_, err := svc.GetAdversary(ctx, &pb.GetDaggerheartAdversaryRequest{Id: "missing"})
			return err
		}},
		{"GetBeastform", func() error {
			_, err := svc.GetBeastform(ctx, &pb.GetDaggerheartBeastformRequest{Id: "missing"})
			return err
		}},
		{"GetCompanionExperience", func() error {
			_, err := svc.GetCompanionExperience(ctx, &pb.GetDaggerheartCompanionExperienceRequest{Id: "missing"})
			return err
		}},
		{"GetLootEntry", func() error {
			_, err := svc.GetLootEntry(ctx, &pb.GetDaggerheartLootEntryRequest{Id: "missing"})
			return err
		}},
		{"GetDamageType", func() error {
			_, err := svc.GetDamageType(ctx, &pb.GetDaggerheartDamageTypeRequest{Id: "missing"})
			return err
		}},
		{"GetDomain", func() error { _, err := svc.GetDomain(ctx, &pb.GetDaggerheartDomainRequest{Id: "missing"}); return err }},
		{"GetDomainCard", func() error {
			_, err := svc.GetDomainCard(ctx, &pb.GetDaggerheartDomainCardRequest{Id: "missing"})
			return err
		}},
		{"GetWeapon", func() error { _, err := svc.GetWeapon(ctx, &pb.GetDaggerheartWeaponRequest{Id: "missing"}); return err }},
		{"GetArmor", func() error { _, err := svc.GetArmor(ctx, &pb.GetDaggerheartArmorRequest{Id: "missing"}); return err }},
		{"GetItem", func() error { _, err := svc.GetItem(ctx, &pb.GetDaggerheartItemRequest{Id: "missing"}); return err }},
		{"GetEnvironment", func() error {
			_, err := svc.GetEnvironment(ctx, &pb.GetDaggerheartEnvironmentRequest{Id: "missing"})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.NotFound)
		})
	}
}

// TestGetContentEndpoints_NilRequest verifies each Get* endpoint returns InvalidArgument for nil requests.
func TestGetContentEndpoints_NilRequest(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetSubclass", func() error { _, err := svc.GetSubclass(ctx, nil); return err }},
		{"GetHeritage", func() error { _, err := svc.GetHeritage(ctx, nil); return err }},
		{"GetExperience", func() error { _, err := svc.GetExperience(ctx, nil); return err }},
		{"GetAdversary", func() error { _, err := svc.GetAdversary(ctx, nil); return err }},
		{"GetBeastform", func() error { _, err := svc.GetBeastform(ctx, nil); return err }},
		{"GetCompanionExperience", func() error { _, err := svc.GetCompanionExperience(ctx, nil); return err }},
		{"GetLootEntry", func() error { _, err := svc.GetLootEntry(ctx, nil); return err }},
		{"GetDamageType", func() error { _, err := svc.GetDamageType(ctx, nil); return err }},
		{"GetDomain", func() error { _, err := svc.GetDomain(ctx, nil); return err }},
		{"GetDomainCard", func() error { _, err := svc.GetDomainCard(ctx, nil); return err }},
		{"GetWeapon", func() error { _, err := svc.GetWeapon(ctx, nil); return err }},
		{"GetArmor", func() error { _, err := svc.GetArmor(ctx, nil); return err }},
		{"GetItem", func() error { _, err := svc.GetItem(ctx, nil); return err }},
		{"GetEnvironment", func() error { _, err := svc.GetEnvironment(ctx, nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}
