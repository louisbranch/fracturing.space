package contenttransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func TestToProtoDaggerheartAdversaryEntry(t *testing.T) {
	proto := toProtoDaggerheartAdversaryEntry(contentstore.DaggerheartAdversaryEntry{
		ID:              "adv-1",
		Name:            "Goblin",
		Tier:            1,
		Role:            "bruiser",
		Description:     "A goblin",
		Motives:         "Greed",
		Difficulty:      2,
		MajorThreshold:  8,
		SevereThreshold: 12,
		HP:              6,
		Stress:          3,
		Armor:           1,
		AttackModifier:  2,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Slash",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 6, Count: 1}},
			DamageBonus: 2,
			DamageType:  "physical",
		},
		Experiences: []contentstore.DaggerheartAdversaryExperience{
			{Name: "Stealth", Modifier: 3},
		},
		Features: []contentstore.DaggerheartAdversaryFeature{
			{ID: "feat-1", Name: "Sneak", Kind: "passive", Description: "Steals", CostType: "action", Cost: 1},
		},
	})

	if proto.GetId() != "adv-1" || proto.GetName() != "Goblin" {
		t.Fatalf("adversary entry metadata mismatch: %v", proto)
	}
	if proto.GetTier() != 1 || proto.GetRole() != "bruiser" {
		t.Fatalf("adversary entry tier/role mismatch")
	}
	if proto.GetHp() != 6 || proto.GetStress() != 3 || proto.GetArmor() != 1 {
		t.Fatalf("adversary entry stats mismatch")
	}
	if proto.GetMajorThreshold() != 8 || proto.GetSevereThreshold() != 12 {
		t.Fatalf("adversary entry thresholds mismatch")
	}
	attack := proto.GetStandardAttack()
	if attack.GetName() != "Slash" || attack.GetDamageBonus() != 2 {
		t.Fatalf("adversary attack mismatch: %v", attack)
	}
	if attack.GetDamageType() != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL {
		t.Fatalf("adversary attack damage type mismatch: %v", attack.GetDamageType())
	}
	if len(attack.GetDamageDice()) != 1 || attack.GetDamageDice()[0].GetSides() != 6 {
		t.Fatalf("adversary attack dice mismatch: %v", attack.GetDamageDice())
	}
	if len(proto.GetExperiences()) != 1 || proto.GetExperiences()[0].GetName() != "Stealth" {
		t.Fatalf("adversary experiences mismatch: %v", proto.GetExperiences())
	}
	if proto.GetExperiences()[0].GetModifier() != 3 {
		t.Fatalf("adversary experience modifier mismatch")
	}
	if len(proto.GetFeatures()) != 1 || proto.GetFeatures()[0].GetId() != "feat-1" {
		t.Fatalf("adversary features mismatch: %v", proto.GetFeatures())
	}
	feat := proto.GetFeatures()[0]
	if feat.GetKind() != "passive" || feat.GetCostType() != "action" || feat.GetCost() != 1 {
		t.Fatalf("adversary feature fields mismatch: %v", feat)
	}
}

func TestToProtoDaggerheartAdversaryEntries(t *testing.T) {
	protos := toProtoDaggerheartAdversaryEntries([]contentstore.DaggerheartAdversaryEntry{
		{ID: "a1", Name: "A"},
		{ID: "a2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "a1" {
		t.Fatalf("adversary entries slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartBeastform(t *testing.T) {
	proto := toProtoDaggerheartBeastform(contentstore.DaggerheartBeastformEntry{
		ID:           "beast-1",
		Name:         "Wolf",
		Tier:         2,
		Examples:     "Dire wolf",
		Trait:        "agility",
		TraitBonus:   3,
		EvasionBonus: 2,
		Attack: contentstore.DaggerheartBeastformAttack{
			Range:       "melee",
			Trait:       "agility",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageBonus: 1,
			DamageType:  "physical",
		},
		Advantages: []string{"pack tactics"},
		Features: []contentstore.DaggerheartBeastformFeature{
			{ID: "bf-1", Name: "Keen Smell", Description: "Tracks by scent"},
		},
	})

	if proto.GetId() != "beast-1" || proto.GetName() != "Wolf" {
		t.Fatalf("beastform metadata mismatch: %v", proto)
	}
	if proto.GetTier() != 2 || proto.GetTrait() != "agility" {
		t.Fatalf("beastform tier/trait mismatch")
	}
	if proto.GetTraitBonus() != 3 || proto.GetEvasionBonus() != 2 {
		t.Fatalf("beastform bonuses mismatch")
	}
	if proto.GetExamples() != "Dire wolf" {
		t.Fatalf("beastform examples mismatch: %v", proto.GetExamples())
	}
	attack := proto.GetAttack()
	if attack.GetRange() != "melee" || attack.GetTrait() != "agility" {
		t.Fatalf("beastform attack mismatch: %v", attack)
	}
	if attack.GetDamageType() != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL {
		t.Fatalf("beastform attack damage type mismatch")
	}
	if len(proto.GetAdvantages()) != 1 || proto.GetAdvantages()[0] != "pack tactics" {
		t.Fatalf("beastform advantages mismatch: %v", proto.GetAdvantages())
	}
	if len(proto.GetFeatures()) != 1 || proto.GetFeatures()[0].GetId() != "bf-1" {
		t.Fatalf("beastform features mismatch: %v", proto.GetFeatures())
	}
}

func TestToProtoDaggerheartBeastforms(t *testing.T) {
	protos := toProtoDaggerheartBeastforms([]contentstore.DaggerheartBeastformEntry{
		{ID: "b1", Name: "A"},
		{ID: "b2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "b2" {
		t.Fatalf("beastforms slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartCompanionExperience(t *testing.T) {
	proto := toProtoDaggerheartCompanionExperience(contentstore.DaggerheartCompanionExperienceEntry{
		ID:          "cexp-1",
		Name:        "Guard Training",
		Description: "Trained as a guard",
	})

	if proto.GetId() != "cexp-1" || proto.GetName() != "Guard Training" || proto.GetDescription() != "Trained as a guard" {
		t.Fatalf("companion experience mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartCompanionExperiences(t *testing.T) {
	protos := toProtoDaggerheartCompanionExperiences([]contentstore.DaggerheartCompanionExperienceEntry{
		{ID: "c1", Name: "A"},
		{ID: "c2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "c1" {
		t.Fatalf("companion experiences slice mismatch: got %d items", len(protos))
	}
}
