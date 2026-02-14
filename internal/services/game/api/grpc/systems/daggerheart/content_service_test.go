package daggerheart

import (
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestContentStoreMissing(t *testing.T) {
	var nilService *DaggerheartContentService
	_, err := nilService.contentStore()
	assertStatusCode(t, err, codes.Internal)

	service := &DaggerheartContentService{}
	_, err = service.contentStore()
	assertStatusCode(t, err, codes.Internal)
}

func TestMapContentErr(t *testing.T) {
	err := mapContentErr("get class", storage.ErrNotFound)
	assertStatusCode(t, err, codes.NotFound)

	err = mapContentErr("get class", errors.New("boom"))
	assertStatusCode(t, err, codes.Internal)
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", err)
	}
	if statusErr.Message() != "get class: boom" {
		t.Fatalf("message = %q, want %q", statusErr.Message(), "get class: boom")
	}
}

func TestToProtoDaggerheartClass(t *testing.T) {
	proto := toProtoDaggerheartClass(storage.DaggerheartClass{
		ID:              "class-1",
		Name:            "Guardian",
		StartingEvasion: 12,
		StartingHP:      16,
		StartingItems:   []string{"shield", "blade"},
		Features: []storage.DaggerheartFeature{{
			ID:          "feat-1",
			Name:        "Hold the Line",
			Description: "Stand firm",
			Level:       1,
		}},
		HopeFeature: storage.DaggerheartHopeFeature{
			Name:        "Resolve",
			Description: "Keep fighting",
			HopeCost:    2,
		},
		DomainIDs: []string{"valor", "shield"},
	})

	if proto.GetId() != "class-1" || proto.GetName() != "Guardian" {
		t.Fatalf("class metadata mismatch: %v", proto)
	}
	if proto.GetStartingEvasion() != 12 || proto.GetStartingHp() != 16 {
		t.Fatalf("starting stats mismatch: %v", proto)
	}
	if len(proto.GetStartingItems()) != 2 || proto.GetStartingItems()[0] != "shield" {
		t.Fatalf("starting items mismatch: %v", proto.GetStartingItems())
	}
	if len(proto.GetDomainIds()) != 2 || proto.GetDomainIds()[1] != "shield" {
		t.Fatalf("domain ids mismatch: %v", proto.GetDomainIds())
	}
	if len(proto.GetFeatures()) != 1 || proto.GetFeatures()[0].GetId() != "feat-1" {
		t.Fatalf("features mismatch: %v", proto.GetFeatures())
	}
	if proto.GetHopeFeature().GetHopeCost() != 2 {
		t.Fatalf("hope feature mismatch: %v", proto.GetHopeFeature())
	}
}

func TestToProtoDaggerheartDamageDice(t *testing.T) {
	dice := toProtoDaggerheartDamageDice([]storage.DaggerheartDamageDie{{Sides: 6, Count: 2}})
	if len(dice) != 1 {
		t.Fatalf("expected 1 dice spec, got %d", len(dice))
	}
	if dice[0].GetSides() != 6 || dice[0].GetCount() != 2 {
		t.Fatalf("dice mapping mismatch: %v", dice[0])
	}
}

func TestContentKindMappings(t *testing.T) {
	if heritageKindToProto(" Ancestry ") != pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY {
		t.Fatal("expected ancestry heritage kind")
	}
	if heritageKindToProto("community") != pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY {
		t.Fatal("expected community heritage kind")
	}
	if heritageKindToProto("unknown") != pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_UNSPECIFIED {
		t.Fatal("expected unspecified heritage kind")
	}

	if domainCardTypeToProto("Spell") != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_SPELL {
		t.Fatal("expected spell domain card type")
	}
	if weaponCategoryToProto("secondary") != pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_SECONDARY {
		t.Fatal("expected secondary weapon category")
	}
	if itemRarityToProto("Rare") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_RARE {
		t.Fatal("expected rare item rarity")
	}
	if itemKindToProto("equipment") != pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_EQUIPMENT {
		t.Fatal("expected equipment item kind")
	}
	if environmentTypeToProto(" Social ") != pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_SOCIAL {
		t.Fatal("expected social environment type")
	}
	if damageTypeToProto("Mixed") != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED {
		t.Fatal("expected mixed damage type")
	}
}

func TestToProtoDaggerheartWeapon(t *testing.T) {
	proto := toProtoDaggerheartWeapon(storage.DaggerheartWeapon{
		ID:         "weapon-1",
		Name:       "Blade",
		Category:   "primary",
		Tier:       2,
		Trait:      "finesse",
		Range:      "melee",
		DamageDice: []storage.DaggerheartDamageDie{{Sides: 8, Count: 1}},
		DamageType: "physical",
		Burden:     1,
		Feature:    "quick",
	})

	if proto.GetId() != "weapon-1" || proto.GetCategory() != pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_PRIMARY {
		t.Fatalf("weapon mapping mismatch: %v", proto)
	}
	if proto.GetDamageType() != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL {
		t.Fatalf("damage type mismatch: %v", proto.GetDamageType())
	}
	if len(proto.GetDamageDice()) != 1 || proto.GetDamageDice()[0].GetSides() != 8 {
		t.Fatalf("damage dice mismatch: %v", proto.GetDamageDice())
	}
}

func TestToProtoDaggerheartItemAndEnvironment(t *testing.T) {
	item := toProtoDaggerheartItem(storage.DaggerheartItem{
		ID:          "item-1",
		Name:        "Potion",
		Rarity:      "uncommon",
		Kind:        "consumable",
		StackMax:    3,
		Description: "Heals",
		EffectText:  "Restore",
	})
	if item.GetRarity() != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNCOMMON {
		t.Fatalf("item rarity mismatch: %v", item)
	}
	if item.GetKind() != pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_CONSUMABLE {
		t.Fatalf("item kind mismatch: %v", item)
	}

	env := toProtoDaggerheartEnvironment(storage.DaggerheartEnvironment{
		ID:         "env-1",
		Name:       "Grotto",
		Tier:       1,
		Type:       "exploration",
		Difficulty: 2,
		Impulses:   []string{"lure"},
		PotentialAdversaryIDs: []string{
			"adv-1",
		},
		Features: []storage.DaggerheartFeature{{ID: "feat-2", Name: "Echo", Description: "Loud", Level: 1}},
		Prompts:  []string{"What echoes?"},
	})
	if env.GetType() != pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_EXPLORATION {
		t.Fatalf("environment type mismatch: %v", env)
	}
	if len(env.GetPotentialAdversaryIds()) != 1 || env.GetPotentialAdversaryIds()[0] != "adv-1" {
		t.Fatalf("environment adversary ids mismatch: %v", env.GetPotentialAdversaryIds())
	}
	if len(env.GetFeatures()) != 1 || env.GetFeatures()[0].GetId() != "feat-2" {
		t.Fatalf("environment features mismatch: %v", env.GetFeatures())
	}
}
