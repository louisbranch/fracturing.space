package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

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
