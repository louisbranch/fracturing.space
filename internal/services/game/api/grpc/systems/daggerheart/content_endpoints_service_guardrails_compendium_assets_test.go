package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

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

func TestListCompendiumAssetEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListLootEntries", func() error { _, err := svc.ListLootEntries(ctx, nil); return err }},
		{"ListWeapons", func() error { _, err := svc.ListWeapons(ctx, nil); return err }},
		{"ListArmor", func() error { _, err := svc.ListArmor(ctx, nil); return err }},
		{"ListItems", func() error { _, err := svc.ListItems(ctx, nil); return err }},
		{"ListEnvironments", func() error { _, err := svc.ListEnvironments(ctx, nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestListCompendiumAssetEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListLootEntries", func() error { _, err := svc.ListLootEntries(ctx, &pb.ListDaggerheartLootEntriesRequest{}); return err }},
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
			grpcassert.StatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

func TestGetCompendiumAssetEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetLootEntry", func() error { _, err := svc.GetLootEntry(ctx, nil); return err }},
		{"GetWeapon", func() error { _, err := svc.GetWeapon(ctx, nil); return err }},
		{"GetArmor", func() error { _, err := svc.GetArmor(ctx, nil); return err }},
		{"GetItem", func() error { _, err := svc.GetItem(ctx, nil); return err }},
		{"GetEnvironment", func() error { _, err := svc.GetEnvironment(ctx, nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestGetCompendiumAssetEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetLootEntry", func() error { _, err := svc.GetLootEntry(ctx, &pb.GetDaggerheartLootEntryRequest{Id: "x"}); return err }},
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
			grpcassert.StatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

func TestGetCompendiumAssetEndpoints_EmptyID(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetLootEntry", func() error { _, err := svc.GetLootEntry(ctx, &pb.GetDaggerheartLootEntryRequest{Id: ""}); return err }},
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
			grpcassert.StatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestGetCompendiumAssetEndpoints_NotFound(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetLootEntry", func() error {
			_, err := svc.GetLootEntry(ctx, &pb.GetDaggerheartLootEntryRequest{Id: "missing"})
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
			grpcassert.StatusCode(t, tc.fn(), codes.NotFound)
		})
	}
}
