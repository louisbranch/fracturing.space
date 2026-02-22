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
	svc, err := NewDaggerheartContentService(Stores{DaggerheartContent: cs})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewDaggerheartContentServiceRejectsMissingStore(t *testing.T) {
	svc, err := NewDaggerheartContentService(Stores{})
	if err == nil {
		t.Fatal("expected constructor error for missing content store")
	}
	if svc != nil {
		t.Fatal("expected nil service on constructor error")
	}
}
