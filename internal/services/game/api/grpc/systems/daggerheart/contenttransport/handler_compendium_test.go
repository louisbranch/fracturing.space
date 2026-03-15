package contenttransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestHandlerCompendiumEndpoints(t *testing.T) {
	ctx := context.Background()
	handler := NewHandler(newFakeContentStore())

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
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
