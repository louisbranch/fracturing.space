package contenttransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func TestToProtoDaggerheartWeapon(t *testing.T) {
	proto := toProtoDaggerheartWeapon(contentstore.DaggerheartWeapon{
		ID:           "weapon-1",
		Name:         "Blade",
		Category:     "primary",
		Tier:         2,
		Trait:        "finesse",
		Range:        "melee",
		DamageDice:   []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
		DamageType:   "physical",
		Burden:       1,
		Feature:      "quick",
		DisplayOrder: 12,
		DisplayGroup: contentstore.DaggerheartWeaponDisplayGroupMagic,
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
	if proto.GetDisplayOrder() != 12 || proto.GetDisplayGroup() != pb.DaggerheartWeaponDisplayGroup_DAGGERHEART_WEAPON_DISPLAY_GROUP_MAGIC {
		t.Fatalf("display metadata mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartLootEntry(t *testing.T) {
	proto := toProtoDaggerheartLootEntry(contentstore.DaggerheartLootEntry{
		ID:          "loot-1",
		Name:        "Gold Coins",
		Roll:        3,
		Description: "A pile of gold",
	})

	if proto.GetId() != "loot-1" || proto.GetName() != "Gold Coins" {
		t.Fatalf("loot entry metadata mismatch: %v", proto)
	}
	if proto.GetRoll() != 3 || proto.GetDescription() != "A pile of gold" {
		t.Fatalf("loot entry fields mismatch: roll=%d desc=%s", proto.GetRoll(), proto.GetDescription())
	}
}

func TestToProtoDaggerheartLootEntries(t *testing.T) {
	protos := toProtoDaggerheartLootEntries([]contentstore.DaggerheartLootEntry{
		{ID: "l1", Name: "A"},
		{ID: "l2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "l2" {
		t.Fatalf("loot entries slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartArmor(t *testing.T) {
	proto := toProtoDaggerheartArmor(contentstore.DaggerheartArmor{
		ID:                  "armor-1",
		Name:                "Chain Mail",
		Tier:                2,
		BaseMajorThreshold:  7,
		BaseSevereThreshold: 14,
		ArmorScore:          3,
		Feature:             "Heavy",
	})

	if proto.GetId() != "armor-1" || proto.GetName() != "Chain Mail" {
		t.Fatalf("armor metadata mismatch: %v", proto)
	}
	if proto.GetTier() != 2 || proto.GetArmorScore() != 3 {
		t.Fatalf("armor tier/score mismatch")
	}
	if proto.GetBaseMajorThreshold() != 7 || proto.GetBaseSevereThreshold() != 14 {
		t.Fatalf("armor thresholds mismatch")
	}
	if proto.GetFeature() != "Heavy" {
		t.Fatalf("armor feature mismatch: %v", proto.GetFeature())
	}
}

func TestToProtoDaggerheartArmorList(t *testing.T) {
	protos := toProtoDaggerheartArmorList([]contentstore.DaggerheartArmor{
		{ID: "a1", Name: "A"},
		{ID: "a2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "a2" {
		t.Fatalf("armor list slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartItemAndEnvironment(t *testing.T) {
	item := toProtoDaggerheartItem(contentstore.DaggerheartItem{
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

	env := toProtoDaggerheartEnvironment(contentstore.DaggerheartEnvironment{
		ID:         "env-1",
		Name:       "Grotto",
		Tier:       1,
		Type:       "exploration",
		Difficulty: 2,
		Impulses:   []string{"lure"},
		PotentialAdversaryIDs: []string{
			"adv-1",
		},
		Features: []contentstore.DaggerheartFeature{{ID: "feat-2", Name: "Echo", Description: "Loud", Level: 1}},
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
