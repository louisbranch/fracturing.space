package contenttransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

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

func (s *fakeContentStore) GetDaggerheartClass(_ context.Context, id string) (contentstore.DaggerheartClass, error) {
	return mapGet(s.classes, id)
}
func (s *fakeContentStore) ListDaggerheartClasses(_ context.Context) ([]contentstore.DaggerheartClass, error) {
	return mapList(s.classes)
}
func (s *fakeContentStore) GetDaggerheartSubclass(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	return mapGet(s.subclasses, id)
}
func (s *fakeContentStore) ListDaggerheartSubclasses(_ context.Context) ([]contentstore.DaggerheartSubclass, error) {
	return mapList(s.subclasses)
}
func (s *fakeContentStore) GetDaggerheartHeritage(_ context.Context, id string) (contentstore.DaggerheartHeritage, error) {
	return mapGet(s.heritages, id)
}
func (s *fakeContentStore) ListDaggerheartHeritages(_ context.Context) ([]contentstore.DaggerheartHeritage, error) {
	return mapList(s.heritages)
}
func (s *fakeContentStore) GetDaggerheartExperience(_ context.Context, id string) (contentstore.DaggerheartExperienceEntry, error) {
	return mapGet(s.experiences, id)
}
func (s *fakeContentStore) ListDaggerheartExperiences(_ context.Context) ([]contentstore.DaggerheartExperienceEntry, error) {
	return mapList(s.experiences)
}
func (s *fakeContentStore) GetDaggerheartAdversaryEntry(_ context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error) {
	return mapGet(s.adversaries, id)
}
func (s *fakeContentStore) ListDaggerheartAdversaryEntries(_ context.Context) ([]contentstore.DaggerheartAdversaryEntry, error) {
	return mapList(s.adversaries)
}
func (s *fakeContentStore) GetDaggerheartBeastform(_ context.Context, id string) (contentstore.DaggerheartBeastformEntry, error) {
	return mapGet(s.beastforms, id)
}
func (s *fakeContentStore) ListDaggerheartBeastforms(_ context.Context) ([]contentstore.DaggerheartBeastformEntry, error) {
	return mapList(s.beastforms)
}
func (s *fakeContentStore) GetDaggerheartCompanionExperience(_ context.Context, id string) (contentstore.DaggerheartCompanionExperienceEntry, error) {
	return mapGet(s.companionExperiences, id)
}
func (s *fakeContentStore) ListDaggerheartCompanionExperiences(_ context.Context) ([]contentstore.DaggerheartCompanionExperienceEntry, error) {
	return mapList(s.companionExperiences)
}
func (s *fakeContentStore) GetDaggerheartLootEntry(_ context.Context, id string) (contentstore.DaggerheartLootEntry, error) {
	return mapGet(s.lootEntries, id)
}
func (s *fakeContentStore) ListDaggerheartLootEntries(_ context.Context) ([]contentstore.DaggerheartLootEntry, error) {
	return mapList(s.lootEntries)
}
func (s *fakeContentStore) GetDaggerheartDamageType(_ context.Context, id string) (contentstore.DaggerheartDamageTypeEntry, error) {
	return mapGet(s.damageTypes, id)
}
func (s *fakeContentStore) ListDaggerheartDamageTypes(_ context.Context) ([]contentstore.DaggerheartDamageTypeEntry, error) {
	return mapList(s.damageTypes)
}
func (s *fakeContentStore) GetDaggerheartEnvironment(_ context.Context, id string) (contentstore.DaggerheartEnvironment, error) {
	return mapGet(s.environments, id)
}
func (s *fakeContentStore) ListDaggerheartEnvironments(_ context.Context) ([]contentstore.DaggerheartEnvironment, error) {
	return mapList(s.environments)
}
func (s *fakeContentStore) GetDaggerheartDomain(_ context.Context, id string) (contentstore.DaggerheartDomain, error) {
	return mapGet(s.domains, id)
}
func (s *fakeContentStore) ListDaggerheartDomains(_ context.Context) ([]contentstore.DaggerheartDomain, error) {
	return mapList(s.domains)
}
func (s *fakeContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (contentstore.DaggerheartDomainCard, error) {
	return mapGet(s.domainCards, id)
}
func (s *fakeContentStore) ListDaggerheartDomainCards(_ context.Context) ([]contentstore.DaggerheartDomainCard, error) {
	return mapList(s.domainCards)
}
func (s *fakeContentStore) ListDaggerheartDomainCardsByDomain(_ context.Context, domainID string) ([]contentstore.DaggerheartDomainCard, error) {
	cards := make([]contentstore.DaggerheartDomainCard, 0, len(s.domainCards))
	for _, card := range s.domainCards {
		if card.DomainID == domainID {
			cards = append(cards, card)
		}
	}
	return cards, nil
}
func (s *fakeContentStore) GetDaggerheartWeapon(_ context.Context, id string) (contentstore.DaggerheartWeapon, error) {
	return mapGet(s.weapons, id)
}
func (s *fakeContentStore) ListDaggerheartWeapons(_ context.Context) ([]contentstore.DaggerheartWeapon, error) {
	return mapList(s.weapons)
}
func (s *fakeContentStore) GetDaggerheartArmor(_ context.Context, id string) (contentstore.DaggerheartArmor, error) {
	return mapGet(s.armor, id)
}
func (s *fakeContentStore) ListDaggerheartArmor(_ context.Context) ([]contentstore.DaggerheartArmor, error) {
	return mapList(s.armor)
}
func (s *fakeContentStore) GetDaggerheartItem(_ context.Context, id string) (contentstore.DaggerheartItem, error) {
	return mapGet(s.items, id)
}
func (s *fakeContentStore) ListDaggerheartItems(_ context.Context) ([]contentstore.DaggerheartItem, error) {
	return mapList(s.items)
}
func (s *fakeContentStore) ListDaggerheartContentStrings(_ context.Context, _ string, _ []string, _ string) ([]contentstore.DaggerheartContentString, error) {
	return nil, nil
}

func TestHandlerEndpointsSmoke(t *testing.T) {
	ctx := context.Background()
	handler := NewHandler(newFakeContentStore())

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "GetContentCatalog",
			run: func(t *testing.T) {
				resp, err := handler.GetContentCatalog(ctx, &pb.GetDaggerheartContentCatalogRequest{})
				if err != nil {
					t.Fatalf("GetContentCatalog: %v", err)
				}
				if len(resp.GetCatalog().GetClasses()) != 1 || len(resp.GetCatalog().GetEnvironments()) != 1 {
					t.Fatalf("catalog counts mismatch: %+v", resp.GetCatalog())
				}
			},
		},
		{
			name: "GetAssetMap",
			run: func(t *testing.T) {
				resp, err := handler.GetAssetMap(ctx, &pb.GetDaggerheartAssetMapRequest{})
				if err != nil {
					t.Fatalf("GetAssetMap: %v", err)
				}
				if len(resp.GetAssetMap().GetAssets()) == 0 {
					t.Fatal("expected non-empty asset map")
				}
			},
		},
		{
			name: "GetClass",
			run: func(t *testing.T) {
				resp, err := handler.GetClass(ctx, &pb.GetDaggerheartClassRequest{Id: "class-1"})
				if err != nil || resp.GetClass().GetId() != "class-1" {
					t.Fatalf("GetClass: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListClasses",
			run: func(t *testing.T) {
				resp, err := handler.ListClasses(ctx, &pb.ListDaggerheartClassesRequest{})
				if err != nil || len(resp.GetClasses()) != 1 {
					t.Fatalf("ListClasses: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetSubclass",
			run: func(t *testing.T) {
				resp, err := handler.GetSubclass(ctx, &pb.GetDaggerheartSubclassRequest{Id: "sub-1"})
				if err != nil || resp.GetSubclass().GetId() != "sub-1" {
					t.Fatalf("GetSubclass: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListSubclasses",
			run: func(t *testing.T) {
				resp, err := handler.ListSubclasses(ctx, &pb.ListDaggerheartSubclassesRequest{})
				if err != nil || len(resp.GetSubclasses()) != 1 {
					t.Fatalf("ListSubclasses: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetHeritage",
			run: func(t *testing.T) {
				resp, err := handler.GetHeritage(ctx, &pb.GetDaggerheartHeritageRequest{Id: "her-1"})
				if err != nil || resp.GetHeritage().GetId() != "her-1" {
					t.Fatalf("GetHeritage: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListHeritages",
			run: func(t *testing.T) {
				resp, err := handler.ListHeritages(ctx, &pb.ListDaggerheartHeritagesRequest{})
				if err != nil || len(resp.GetHeritages()) != 1 {
					t.Fatalf("ListHeritages: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetExperience",
			run: func(t *testing.T) {
				resp, err := handler.GetExperience(ctx, &pb.GetDaggerheartExperienceRequest{Id: "exp-1"})
				if err != nil || resp.GetExperience().GetId() != "exp-1" {
					t.Fatalf("GetExperience: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListExperiences",
			run: func(t *testing.T) {
				resp, err := handler.ListExperiences(ctx, &pb.ListDaggerheartExperiencesRequest{})
				if err != nil || len(resp.GetExperiences()) != 1 {
					t.Fatalf("ListExperiences: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetAdversary",
			run: func(t *testing.T) {
				resp, err := handler.GetAdversary(ctx, &pb.GetDaggerheartAdversaryRequest{Id: "adv-1"})
				if err != nil || resp.GetAdversary().GetId() != "adv-1" {
					t.Fatalf("GetAdversary: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListAdversaries",
			run: func(t *testing.T) {
				resp, err := handler.ListAdversaries(ctx, &pb.ListDaggerheartAdversariesRequest{})
				if err != nil || len(resp.GetAdversaries()) != 1 {
					t.Fatalf("ListAdversaries: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetBeastform",
			run: func(t *testing.T) {
				resp, err := handler.GetBeastform(ctx, &pb.GetDaggerheartBeastformRequest{Id: "beast-1"})
				if err != nil || resp.GetBeastform().GetId() != "beast-1" {
					t.Fatalf("GetBeastform: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListBeastforms",
			run: func(t *testing.T) {
				resp, err := handler.ListBeastforms(ctx, &pb.ListDaggerheartBeastformsRequest{})
				if err != nil || len(resp.GetBeastforms()) != 1 {
					t.Fatalf("ListBeastforms: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetCompanionExperience",
			run: func(t *testing.T) {
				resp, err := handler.GetCompanionExperience(ctx, &pb.GetDaggerheartCompanionExperienceRequest{Id: "cexp-1"})
				if err != nil || resp.GetExperience().GetId() != "cexp-1" {
					t.Fatalf("GetCompanionExperience: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListCompanionExperiences",
			run: func(t *testing.T) {
				resp, err := handler.ListCompanionExperiences(ctx, &pb.ListDaggerheartCompanionExperiencesRequest{})
				if err != nil || len(resp.GetExperiences()) != 1 {
					t.Fatalf("ListCompanionExperiences: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetLootEntry",
			run: func(t *testing.T) {
				resp, err := handler.GetLootEntry(ctx, &pb.GetDaggerheartLootEntryRequest{Id: "loot-1"})
				if err != nil || resp.GetEntry().GetId() != "loot-1" {
					t.Fatalf("GetLootEntry: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListLootEntries",
			run: func(t *testing.T) {
				resp, err := handler.ListLootEntries(ctx, &pb.ListDaggerheartLootEntriesRequest{})
				if err != nil || len(resp.GetEntries()) != 1 {
					t.Fatalf("ListLootEntries: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetDamageType",
			run: func(t *testing.T) {
				resp, err := handler.GetDamageType(ctx, &pb.GetDaggerheartDamageTypeRequest{Id: "dt-1"})
				if err != nil || resp.GetDamageType().GetId() != "dt-1" {
					t.Fatalf("GetDamageType: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListDamageTypes",
			run: func(t *testing.T) {
				resp, err := handler.ListDamageTypes(ctx, &pb.ListDaggerheartDamageTypesRequest{})
				if err != nil || len(resp.GetDamageTypes()) != 1 {
					t.Fatalf("ListDamageTypes: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetDomain",
			run: func(t *testing.T) {
				resp, err := handler.GetDomain(ctx, &pb.GetDaggerheartDomainRequest{Id: "dom-1"})
				if err != nil || resp.GetDomain().GetId() != "dom-1" {
					t.Fatalf("GetDomain: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListDomains",
			run: func(t *testing.T) {
				resp, err := handler.ListDomains(ctx, &pb.ListDaggerheartDomainsRequest{})
				if err != nil || len(resp.GetDomains()) != 1 {
					t.Fatalf("ListDomains: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetDomainCard",
			run: func(t *testing.T) {
				resp, err := handler.GetDomainCard(ctx, &pb.GetDaggerheartDomainCardRequest{Id: "card-1"})
				if err != nil || resp.GetDomainCard().GetId() != "card-1" {
					t.Fatalf("GetDomainCard: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListDomainCards",
			run: func(t *testing.T) {
				resp, err := handler.ListDomainCards(ctx, &pb.ListDaggerheartDomainCardsRequest{})
				if err != nil || len(resp.GetDomainCards()) != 1 {
					t.Fatalf("ListDomainCards: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetWeapon",
			run: func(t *testing.T) {
				resp, err := handler.GetWeapon(ctx, &pb.GetDaggerheartWeaponRequest{Id: "weapon-1"})
				if err != nil || resp.GetWeapon().GetId() != "weapon-1" {
					t.Fatalf("GetWeapon: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListWeapons",
			run: func(t *testing.T) {
				resp, err := handler.ListWeapons(ctx, &pb.ListDaggerheartWeaponsRequest{})
				if err != nil || len(resp.GetWeapons()) != 1 {
					t.Fatalf("ListWeapons: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetArmor",
			run: func(t *testing.T) {
				resp, err := handler.GetArmor(ctx, &pb.GetDaggerheartArmorRequest{Id: "armor-1"})
				if err != nil || resp.GetArmor().GetId() != "armor-1" {
					t.Fatalf("GetArmor: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListArmor",
			run: func(t *testing.T) {
				resp, err := handler.ListArmor(ctx, &pb.ListDaggerheartArmorRequest{})
				if err != nil || len(resp.GetArmor()) != 1 {
					t.Fatalf("ListArmor: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetItem",
			run: func(t *testing.T) {
				resp, err := handler.GetItem(ctx, &pb.GetDaggerheartItemRequest{Id: "item-1"})
				if err != nil || resp.GetItem().GetId() != "item-1" {
					t.Fatalf("GetItem: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListItems",
			run: func(t *testing.T) {
				resp, err := handler.ListItems(ctx, &pb.ListDaggerheartItemsRequest{})
				if err != nil || len(resp.GetItems()) != 1 {
					t.Fatalf("ListItems: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetEnvironment",
			run: func(t *testing.T) {
				resp, err := handler.GetEnvironment(ctx, &pb.GetDaggerheartEnvironmentRequest{Id: "env-1"})
				if err != nil || resp.GetEnvironment().GetId() != "env-1" {
					t.Fatalf("GetEnvironment: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListEnvironments",
			run: func(t *testing.T) {
				resp, err := handler.ListEnvironments(ctx, &pb.ListDaggerheartEnvironmentsRequest{})
				if err != nil || len(resp.GetEnvironments()) != 1 {
					t.Fatalf("ListEnvironments: resp=%v err=%v", resp, err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}
