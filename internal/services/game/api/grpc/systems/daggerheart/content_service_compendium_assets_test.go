package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

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
