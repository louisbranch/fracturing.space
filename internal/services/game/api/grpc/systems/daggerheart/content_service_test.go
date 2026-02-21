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

func TestToProtoDaggerheartSubclass(t *testing.T) {
	proto := toProtoDaggerheartSubclass(storage.DaggerheartSubclass{
		ID:             "sub-1",
		Name:           "Bladeweaver",
		SpellcastTrait: "agility",
		FoundationFeatures: []storage.DaggerheartFeature{
			{ID: "f1", Name: "Foundation", Description: "Base", Level: 1},
		},
		SpecializationFeatures: []storage.DaggerheartFeature{
			{ID: "f2", Name: "Specialization", Description: "Mid", Level: 5},
		},
		MasteryFeatures: []storage.DaggerheartFeature{
			{ID: "f3", Name: "Mastery", Description: "Top", Level: 10},
		},
	})

	if proto.GetId() != "sub-1" || proto.GetName() != "Bladeweaver" {
		t.Fatalf("subclass metadata mismatch: %v", proto)
	}
	if proto.GetSpellcastTrait() != "agility" {
		t.Fatalf("spellcast trait mismatch: %v", proto.GetSpellcastTrait())
	}
	if len(proto.GetFoundationFeatures()) != 1 || proto.GetFoundationFeatures()[0].GetId() != "f1" {
		t.Fatalf("foundation features mismatch: %v", proto.GetFoundationFeatures())
	}
	if len(proto.GetSpecializationFeatures()) != 1 || proto.GetSpecializationFeatures()[0].GetId() != "f2" {
		t.Fatalf("specialization features mismatch: %v", proto.GetSpecializationFeatures())
	}
	if len(proto.GetMasteryFeatures()) != 1 || proto.GetMasteryFeatures()[0].GetId() != "f3" {
		t.Fatalf("mastery features mismatch: %v", proto.GetMasteryFeatures())
	}
}

func TestToProtoDaggerheartSubclasses(t *testing.T) {
	protos := toProtoDaggerheartSubclasses([]storage.DaggerheartSubclass{
		{ID: "s1", Name: "A"},
		{ID: "s2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "s1" || protos[1].GetId() != "s2" {
		t.Fatalf("subclasses slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartHeritage(t *testing.T) {
	proto := toProtoDaggerheartHeritage(storage.DaggerheartHeritage{
		ID:   "her-1",
		Name: "Elf",
		Kind: "ancestry",
		Features: []storage.DaggerheartFeature{
			{ID: "f1", Name: "Keen Eyes", Description: "See far", Level: 1},
		},
	})

	if proto.GetId() != "her-1" || proto.GetName() != "Elf" {
		t.Fatalf("heritage metadata mismatch: %v", proto)
	}
	if proto.GetKind() != pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY {
		t.Fatalf("heritage kind mismatch: %v", proto.GetKind())
	}
	if len(proto.GetFeatures()) != 1 || proto.GetFeatures()[0].GetId() != "f1" {
		t.Fatalf("heritage features mismatch: %v", proto.GetFeatures())
	}
}

func TestToProtoDaggerheartHeritages(t *testing.T) {
	protos := toProtoDaggerheartHeritages([]storage.DaggerheartHeritage{
		{ID: "h1", Name: "A", Kind: "ancestry"},
		{ID: "h2", Name: "B", Kind: "community"},
	})
	if len(protos) != 2 || protos[0].GetId() != "h1" {
		t.Fatalf("heritages slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartExperience(t *testing.T) {
	proto := toProtoDaggerheartExperience(storage.DaggerheartExperienceEntry{
		ID:          "exp-1",
		Name:        "Wanderer",
		Description: "Traveled far",
	})

	if proto.GetId() != "exp-1" || proto.GetName() != "Wanderer" || proto.GetDescription() != "Traveled far" {
		t.Fatalf("experience mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartExperiences(t *testing.T) {
	protos := toProtoDaggerheartExperiences([]storage.DaggerheartExperienceEntry{
		{ID: "e1", Name: "A"},
		{ID: "e2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "e2" {
		t.Fatalf("experiences slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartAdversaryEntry(t *testing.T) {
	proto := toProtoDaggerheartAdversaryEntry(storage.DaggerheartAdversaryEntry{
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
		StandardAttack: storage.DaggerheartAdversaryAttack{
			Name:        "Slash",
			Range:       "melee",
			DamageDice:  []storage.DaggerheartDamageDie{{Sides: 6, Count: 1}},
			DamageBonus: 2,
			DamageType:  "physical",
		},
		Experiences: []storage.DaggerheartAdversaryExperience{
			{Name: "Stealth", Modifier: 3},
		},
		Features: []storage.DaggerheartAdversaryFeature{
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
	protos := toProtoDaggerheartAdversaryEntries([]storage.DaggerheartAdversaryEntry{
		{ID: "a1", Name: "A"},
		{ID: "a2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "a1" {
		t.Fatalf("adversary entries slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartBeastform(t *testing.T) {
	proto := toProtoDaggerheartBeastform(storage.DaggerheartBeastformEntry{
		ID:           "beast-1",
		Name:         "Wolf",
		Tier:         2,
		Examples:     "Dire wolf",
		Trait:        "agility",
		TraitBonus:   3,
		EvasionBonus: 2,
		Attack: storage.DaggerheartBeastformAttack{
			Range:       "melee",
			Trait:       "agility",
			DamageDice:  []storage.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageBonus: 1,
			DamageType:  "physical",
		},
		Advantages: []string{"pack tactics"},
		Features: []storage.DaggerheartBeastformFeature{
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
	protos := toProtoDaggerheartBeastforms([]storage.DaggerheartBeastformEntry{
		{ID: "b1", Name: "A"},
		{ID: "b2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "b2" {
		t.Fatalf("beastforms slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartCompanionExperience(t *testing.T) {
	proto := toProtoDaggerheartCompanionExperience(storage.DaggerheartCompanionExperienceEntry{
		ID:          "cexp-1",
		Name:        "Guard Training",
		Description: "Trained as a guard",
	})

	if proto.GetId() != "cexp-1" || proto.GetName() != "Guard Training" || proto.GetDescription() != "Trained as a guard" {
		t.Fatalf("companion experience mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartCompanionExperiences(t *testing.T) {
	protos := toProtoDaggerheartCompanionExperiences([]storage.DaggerheartCompanionExperienceEntry{
		{ID: "c1", Name: "A"},
		{ID: "c2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "c1" {
		t.Fatalf("companion experiences slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartLootEntry(t *testing.T) {
	proto := toProtoDaggerheartLootEntry(storage.DaggerheartLootEntry{
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
	protos := toProtoDaggerheartLootEntries([]storage.DaggerheartLootEntry{
		{ID: "l1", Name: "A"},
		{ID: "l2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "l2" {
		t.Fatalf("loot entries slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartDamageTypeEntry(t *testing.T) {
	proto := toProtoDaggerheartDamageType(storage.DaggerheartDamageTypeEntry{
		ID:          "dt-1",
		Name:        "Fire",
		Description: "Burns things",
	})

	if proto.GetId() != "dt-1" || proto.GetName() != "Fire" || proto.GetDescription() != "Burns things" {
		t.Fatalf("damage type entry mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartDamageTypes(t *testing.T) {
	protos := toProtoDaggerheartDamageTypes([]storage.DaggerheartDamageTypeEntry{
		{ID: "d1", Name: "A"},
		{ID: "d2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "d1" {
		t.Fatalf("damage types slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartDomain(t *testing.T) {
	proto := toProtoDaggerheartDomain(storage.DaggerheartDomain{
		ID:          "dom-1",
		Name:        "Valor",
		Description: "Bravery domain",
	})

	if proto.GetId() != "dom-1" || proto.GetName() != "Valor" || proto.GetDescription() != "Bravery domain" {
		t.Fatalf("domain mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartDomains(t *testing.T) {
	protos := toProtoDaggerheartDomains([]storage.DaggerheartDomain{
		{ID: "d1", Name: "A"},
		{ID: "d2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "d2" {
		t.Fatalf("domains slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartDomainCard(t *testing.T) {
	proto := toProtoDaggerheartDomainCard(storage.DaggerheartDomainCard{
		ID:          "card-1",
		Name:        "Fireball",
		DomainID:    "dom-1",
		Level:       3,
		Type:        "spell",
		RecallCost:  2,
		UsageLimit:  "once",
		FeatureText: "Deals fire damage",
	})

	if proto.GetId() != "card-1" || proto.GetName() != "Fireball" {
		t.Fatalf("domain card metadata mismatch: %v", proto)
	}
	if proto.GetDomainId() != "dom-1" || proto.GetLevel() != 3 {
		t.Fatalf("domain card domain/level mismatch")
	}
	if proto.GetType() != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_SPELL {
		t.Fatalf("domain card type mismatch: %v", proto.GetType())
	}
	if proto.GetRecallCost() != 2 || proto.GetUsageLimit() != "once" || proto.GetFeatureText() != "Deals fire damage" {
		t.Fatalf("domain card fields mismatch")
	}
}

func TestToProtoDaggerheartDomainCards(t *testing.T) {
	protos := toProtoDaggerheartDomainCards([]storage.DaggerheartDomainCard{
		{ID: "c1", Name: "A"},
		{ID: "c2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "c1" {
		t.Fatalf("domain cards slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartArmor(t *testing.T) {
	proto := toProtoDaggerheartArmor(storage.DaggerheartArmor{
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
	protos := toProtoDaggerheartArmorList([]storage.DaggerheartArmor{
		{ID: "a1", Name: "A"},
		{ID: "a2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "a2" {
		t.Fatalf("armor list slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartClasses(t *testing.T) {
	protos := toProtoDaggerheartClasses([]storage.DaggerheartClass{
		{ID: "c1", Name: "A"},
		{ID: "c2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "c1" || protos[1].GetId() != "c2" {
		t.Fatalf("classes slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartFeatures(t *testing.T) {
	protos := toProtoDaggerheartFeatures([]storage.DaggerheartFeature{
		{ID: "f1", Name: "Shield Bash", Description: "Bash with shield", Level: 1},
		{ID: "f2", Name: "Rally", Description: "Rally allies", Level: 5},
	})

	if len(protos) != 2 {
		t.Fatalf("expected 2 features, got %d", len(protos))
	}
	if protos[0].GetId() != "f1" || protos[0].GetName() != "Shield Bash" {
		t.Fatalf("feature 0 mismatch: %v", protos[0])
	}
	if protos[1].GetLevel() != 5 || protos[1].GetDescription() != "Rally allies" {
		t.Fatalf("feature 1 mismatch: %v", protos[1])
	}
}

func TestToProtoDaggerheartHopeFeature(t *testing.T) {
	proto := toProtoDaggerheartHopeFeature(storage.DaggerheartHopeFeature{
		Name:        "Inspiring",
		Description: "Inspire allies",
		HopeCost:    3,
	})

	if proto.GetName() != "Inspiring" || proto.GetDescription() != "Inspire allies" || proto.GetHopeCost() != 3 {
		t.Fatalf("hope feature mismatch: %v", proto)
	}
}

func TestContentKindMappingsExtended(t *testing.T) {
	// domainCardType: ability, grimoire
	if domainCardTypeToProto("ability") != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_ABILITY {
		t.Fatal("expected ability domain card type")
	}
	if domainCardTypeToProto("grimoire") != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_GRIMOIRE {
		t.Fatal("expected grimoire domain card type")
	}
	if domainCardTypeToProto("unknown") != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_UNSPECIFIED {
		t.Fatal("expected unspecified domain card type")
	}

	// weaponCategory: unspecified
	if weaponCategoryToProto("unknown") != pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_UNSPECIFIED {
		t.Fatal("expected unspecified weapon category")
	}

	// itemRarity: common, unique
	if itemRarityToProto("common") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_COMMON {
		t.Fatal("expected common item rarity")
	}
	if itemRarityToProto("unique") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNIQUE {
		t.Fatal("expected unique item rarity")
	}
	if itemRarityToProto("legendary") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_LEGENDARY {
		t.Fatal("expected legendary item rarity")
	}
	if itemRarityToProto("unknown") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNSPECIFIED {
		t.Fatal("expected unspecified item rarity")
	}

	// itemKind: treasure
	if itemKindToProto("treasure") != pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_TREASURE {
		t.Fatal("expected treasure item kind")
	}
	if itemKindToProto("unknown") != pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_UNSPECIFIED {
		t.Fatal("expected unspecified item kind")
	}

	// environmentType: traversal, event
	if environmentTypeToProto("traversal") != pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_TRAVERSAL {
		t.Fatal("expected traversal environment type")
	}
	if environmentTypeToProto("event") != pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_EVENT {
		t.Fatal("expected event environment type")
	}
	if environmentTypeToProto("unknown") != pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_UNSPECIFIED {
		t.Fatal("expected unspecified environment type")
	}

	// damageType: magic
	if damageTypeToProto("magic") != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC {
		t.Fatal("expected magic damage type")
	}
	if damageTypeToProto("unknown") != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		t.Fatal("expected unspecified damage type")
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
