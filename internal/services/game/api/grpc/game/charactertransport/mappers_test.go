package charactertransport

import (
	"context"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
)

func TestCharacterToProto(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	got := CharacterToProto(storage.CharacterRecord{
		ID:                 "char-1",
		CampaignID:         "camp-1",
		OwnerParticipantID: "part-1",
		Name:               "Rook",
		Kind:               character.KindPC,
		Notes:              "note",
		AvatarSetID:        "set-1",
		AvatarAssetID:      "asset-1",
		Pronouns:           sharedpronouns.PronounSheHer,
		Aliases:            []string{"Alias"},
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	if got.GetId() != "char-1" || got.GetCampaignId() != "camp-1" || got.GetName() != "Rook" {
		t.Fatalf("character identity mismatch: %+v", got)
	}
	if got.GetKind() != campaignv1.CharacterKind_PC {
		t.Fatalf("kind = %v", got.GetKind())
	}
	if got.GetOwnerParticipantId().GetValue() != "part-1" {
		t.Fatalf("owner participant id = %q", got.GetOwnerParticipantId().GetValue())
	}
	if len(got.GetAliases()) != 1 || got.GetAliases()[0] != "Alias" {
		t.Fatalf("aliases = %#v", got.GetAliases())
	}
}

func TestCharacterEnumAndDaggerheartConversions(t *testing.T) {
	if KindFromProto(campaignv1.CharacterKind_NPC) != character.KindNPC {
		t.Fatal("kind from proto mismatch")
	}
	if KindFromProto(campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED) != character.KindUnspecified {
		t.Fatal("unspecified kind from proto mismatch")
	}
	if KindToProto(character.KindPC) != campaignv1.CharacterKind_PC {
		t.Fatal("kind to proto mismatch")
	}
	if KindToProto(character.Kind("")) != campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
		t.Fatal("unspecified kind mismatch")
	}

	conditions, err := DaggerheartConditionsFromProto([]daggerheartv1.DaggerheartCondition{
		daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN,
		daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE,
	})
	if err != nil {
		t.Fatalf("conditions from proto: %v", err)
	}
	if len(conditions) != 2 || conditions[0] != rules.ConditionHidden || conditions[1] != rules.ConditionVulnerable {
		t.Fatalf("conditions = %#v", conditions)
	}
	if _, err := DaggerheartConditionsFromProto([]daggerheartv1.DaggerheartCondition{daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED}); err == nil {
		t.Fatal("expected unspecified condition error")
	}
	if got := DaggerheartConditionsToProto([]string{rules.ConditionRestrained, rules.ConditionHidden}); len(got) != 2 {
		t.Fatalf("conditions to proto = %#v", got)
	}
	if DaggerheartExperiencesToProto(nil) != nil {
		t.Fatal("nil experiences should stay nil")
	}

	state, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS)
	if err != nil || state != mechanics.LifeStateUnconscious {
		t.Fatalf("life state from proto = %q err=%v", state, err)
	}
	if alive, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE); err != nil || alive != daggerheartstate.LifeStateAlive {
		t.Fatalf("alive state = %q err=%v", alive, err)
	}
	if blaze, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY); err != nil || blaze != mechanics.LifeStateBlazeOfGlory {
		t.Fatalf("blaze state = %q err=%v", blaze, err)
	}
	if dead, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD); err != nil || dead != mechanics.LifeStateDead {
		t.Fatalf("dead state = %q err=%v", dead, err)
	}
	if _, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED); err == nil {
		t.Fatal("expected unspecified life state error")
	}
	if _, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState(99)); err == nil {
		t.Fatal("expected invalid life state error")
	}
	if DaggerheartLifeStateToProto(mechanics.LifeStateDead) != daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD {
		t.Fatal("life state to proto mismatch")
	}
	if DaggerheartLifeStateToProto("") != daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED {
		t.Fatal("unspecified life state to proto mismatch")
	}
}

func TestDaggerheartProfileAndStateToProto(t *testing.T) {
	profile := DaggerheartProfileToProto("camp-1", "char-1", projectionstore.DaggerheartCharacterProfile{
		Level:                        2,
		HpMax:                        14,
		StressMax:                    6,
		Evasion:                      11,
		MajorThreshold:               18,
		SevereThreshold:              24,
		Proficiency:                  3,
		ArmorScore:                   1,
		ArmorMax:                     2,
		Agility:                      1,
		Strength:                     2,
		Finesse:                      3,
		Instinct:                     4,
		Presence:                     5,
		Knowledge:                    6,
		Experiences:                  []projectionstore.DaggerheartExperience{{Name: "Scout", Modifier: 2}},
		ClassID:                      "class-1",
		SubclassID:                   "sub-1",
		SubclassCreationRequirements: []projectionstore.DaggerheartSubclassCreationRequirement{projectionstore.DaggerheartSubclassCreationRequirementCompanionSheet},
		Heritage: projectionstore.DaggerheartHeritageSelection{
			AncestryLabel:           "Half-Clank",
			FirstFeatureAncestryID:  "anc-1",
			FirstFeatureID:          "anc-1.feature-1",
			SecondFeatureAncestryID: "anc-2",
			SecondFeatureID:         "anc-2.feature-2",
			CommunityID:             "comm-1",
		},
		CompanionSheet: &projectionstore.DaggerheartCompanionSheet{
			AnimalKind:        "Wolf",
			Name:              "Ash",
			Evasion:           10,
			Experiences:       []projectionstore.DaggerheartCompanionExperience{{ExperienceID: "companion-experience.tracking", Name: "Tracking", Modifier: 2}, {ExperienceID: "companion-experience.guarding", Name: "Guarding", Modifier: 2}},
			AttackDescription: "Bites at close range",
			AttackRange:       "melee",
			DamageDieSides:    6,
			DamageType:        "physical",
		},
		TraitsAssigned:       true,
		DetailsRecorded:      true,
		StartingWeaponIDs:    []string{"weapon-1"},
		StartingArmorID:      "armor-1",
		StartingPotionItemID: "potion-1",
		Background:           "bg",
		Description:          "desc",
		DomainCardIDs:        []string{"domain-1"},
		Connections:          "conn",
	}, nil)
	if profile.GetDaggerheart().GetLevel() != 2 || len(profile.GetDaggerheart().GetExperiences()) != 1 {
		t.Fatalf("profile = %+v", profile.GetDaggerheart())
	}
	if got := profile.GetDaggerheart().GetHeritage().GetAncestryLabel(); got != "Half-Clank" {
		t.Fatalf("heritage ancestry label = %q, want %q", got, "Half-Clank")
	}
	if got := profile.GetDaggerheart().GetCompanionSheet().GetName(); got != "Ash" {
		t.Fatalf("companion name = %q, want %q", got, "Ash")
	}

	state := DaggerheartStateToProto("camp-1", "char-1", projectionstore.DaggerheartCharacterState{
		Hp:      10,
		Hope:    2,
		HopeMax: 6,
		Stress:  1,
		Armor:   0,
		Conditions: []projectionstore.DaggerheartConditionState{{
			ID:       rules.ConditionHidden,
			Class:    "standard",
			Standard: rules.ConditionHidden,
			Code:     rules.ConditionHidden,
			Label:    rules.ConditionHidden,
		}},
		LifeState: daggerheartstate.LifeStateAlive,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{{
			Source:   "spell",
			SourceID: "spell.arcane-ward",
			Duration: "short_rest",
			Amount:   2,
		}},
		StatModifiers: []projectionstore.DaggerheartStatModifier{{
			ID:            "mod-evasion-wall",
			Target:        "evasion",
			Delta:         3,
			Label:         "Wall",
			Source:        "domain_card",
			ClearTriggers: []string{"SHORT_REST"},
		}},
	})
	if state.GetDaggerheart().GetHp() != 10 || len(state.GetDaggerheart().GetConditionStates()) != 1 {
		t.Fatalf("state = %+v", state.GetDaggerheart())
	}
	if len(state.GetDaggerheart().GetTemporaryArmorBuckets()) != 1 {
		t.Fatalf("temporary armor buckets = %+v", state.GetDaggerheart().GetTemporaryArmorBuckets())
	}
	if len(state.GetDaggerheart().GetStatModifiers()) != 1 {
		t.Fatalf("stat modifiers = %+v", state.GetDaggerheart().GetStatModifiers())
	}
}

func TestDaggerheartSheetProfileToProtoAddsEquipmentSummaries(t *testing.T) {
	profile := projectionstore.DaggerheartCharacterProfile{
		StartingWeaponIDs: []string{"weapon.primary-blade", "weapon.side-knife"},
		StartingArmorID:   "armor.leather",
		EquippedArmorID:   "armor.scale",
	}
	content := workflowContentStore{
		weapons: map[string]contentstore.DaggerheartWeapon{
			"weapon.primary-blade": {
				ID:         "weapon.primary-blade",
				Name:       "Primary Blade",
				Category:   "primary",
				Trait:      "Agility",
				Range:      "melee",
				DamageDice: []contentstore.DaggerheartDamageDie{{Count: 1, Sides: 8}, {Count: 1, Sides: 4}},
				DamageType: "physical",
				Feature:    "Reliable",
			},
			"weapon.side-knife": {
				ID:         "weapon.side-knife",
				Name:       "Side Knife",
				Category:   "secondary",
				Trait:      "Finesse",
				Range:      "very close",
				DamageDice: []contentstore.DaggerheartDamageDie{{Count: 1, Sides: 6}},
				DamageType: "physical",
			},
		},
		armors: map[string]contentstore.DaggerheartArmor{
			"armor.scale":   {ID: "armor.scale", Name: "Scale", ArmorScore: 3, Feature: "Bulky"},
			"armor.leather": {ID: "armor.leather", Name: "Leather", ArmorScore: 1, Feature: "Quiet"},
		},
	}

	got := DaggerheartSheetProfileToProto(context.Background(), "camp-1", "char-1", profile, content).GetDaggerheart()
	if got == nil {
		t.Fatal("sheet profile = nil")
	}
	if got.GetPrimaryWeapon().GetName() != "Primary Blade" {
		t.Fatalf("primary weapon = %#v", got.GetPrimaryWeapon())
	}
	if got.GetPrimaryWeapon().GetDamageDice() != "1d8 + 1d4" {
		t.Fatalf("primary damage dice = %q, want %q", got.GetPrimaryWeapon().GetDamageDice(), "1d8 + 1d4")
	}
	if got.GetSecondaryWeapon().GetName() != "Side Knife" {
		t.Fatalf("secondary weapon = %#v", got.GetSecondaryWeapon())
	}
	if got.GetActiveArmor().GetName() != "Scale" {
		t.Fatalf("active armor = %#v", got.GetActiveArmor())
	}
	if got.GetActiveArmor().GetBaseScore() != 3 {
		t.Fatalf("active armor base score = %d, want 3", got.GetActiveArmor().GetBaseScore())
	}
}

func TestDaggerheartSheetProfileToProtoAddsHeritageDisplayNames(t *testing.T) {
	profile := projectionstore.DaggerheartCharacterProfile{
		Heritage: projectionstore.DaggerheartHeritageSelection{
			FirstFeatureAncestryID:  "heritage.ancestry.clank",
			SecondFeatureAncestryID: "heritage.ancestry.orc",
			CommunityID:             "heritage.community.farmer",
		},
	}
	content := workflowContentStore{
		heritages: map[string]contentstore.DaggerheartHeritage{
			"heritage.ancestry.clank":   {ID: "heritage.ancestry.clank", Kind: "ancestry", Name: "Clank"},
			"heritage.ancestry.orc":     {ID: "heritage.ancestry.orc", Kind: "ancestry", Name: "Orc"},
			"heritage.community.farmer": {ID: "heritage.community.farmer", Kind: "community", Name: "Farmer"},
		},
	}

	got := DaggerheartSheetProfileToProto(context.Background(), "camp-1", "char-1", profile, content).GetDaggerheart()
	if got == nil || got.GetHeritage() == nil {
		t.Fatalf("sheet profile heritage = %#v", got)
	}
	if got.GetHeritage().GetAncestryName() != "Clank / Orc" {
		t.Fatalf("heritage ancestry name = %q, want %q", got.GetHeritage().GetAncestryName(), "Clank / Orc")
	}
	if got.GetHeritage().GetCommunityName() != "Farmer" {
		t.Fatalf("heritage community name = %q, want %q", got.GetHeritage().GetCommunityName(), "Farmer")
	}
}

func TestDaggerheartSheetProfileToProtoPrefersAncestryLabelAndSkipsMissingHeritageContent(t *testing.T) {
	profile := projectionstore.DaggerheartCharacterProfile{
		Heritage: projectionstore.DaggerheartHeritageSelection{
			AncestryLabel:           "Half-Clank",
			FirstFeatureAncestryID:  "heritage.ancestry.clank",
			SecondFeatureAncestryID: "heritage.ancestry.orc",
			CommunityID:             "heritage.community.unknown",
		},
	}
	content := workflowContentStore{
		heritages: map[string]contentstore.DaggerheartHeritage{
			"heritage.ancestry.clank": {ID: "heritage.ancestry.clank", Kind: "ancestry", Name: "Clank"},
			"heritage.ancestry.orc":   {ID: "heritage.ancestry.orc", Kind: "ancestry", Name: "Orc"},
		},
	}

	got := DaggerheartSheetProfileToProto(context.Background(), "camp-1", "char-1", profile, content).GetDaggerheart()
	if got == nil || got.GetHeritage() == nil {
		t.Fatalf("sheet profile heritage = %#v", got)
	}
	if got.GetHeritage().GetAncestryName() != "Half-Clank" {
		t.Fatalf("heritage ancestry name = %q, want %q", got.GetHeritage().GetAncestryName(), "Half-Clank")
	}
	if got.GetHeritage().GetCommunityName() != "" {
		t.Fatalf("heritage community name = %q, want empty", got.GetHeritage().GetCommunityName())
	}
}

func TestDaggerheartSheetProfileToProtoFallsBackToStartingArmorAndSkipsMissingContent(t *testing.T) {
	profile := projectionstore.DaggerheartCharacterProfile{
		StartingWeaponIDs: []string{"weapon.unknown", "weapon.side-knife"},
		StartingArmorID:   "armor.leather",
	}
	content := workflowContentStore{
		weapons: map[string]contentstore.DaggerheartWeapon{
			"weapon.side-knife": {
				ID:         "weapon.side-knife",
				Name:       "Side Knife",
				Category:   "secondary",
				DamageDice: []contentstore.DaggerheartDamageDie{{Count: 1, Sides: 6}},
			},
		},
		armors: map[string]contentstore.DaggerheartArmor{
			"armor.leather": {ID: "armor.leather", Name: "Leather", ArmorScore: 1},
		},
	}

	got := DaggerheartSheetProfileToProto(context.Background(), "camp-1", "char-1", profile, content).GetDaggerheart()
	if got == nil {
		t.Fatal("sheet profile = nil")
	}
	if got.GetPrimaryWeapon() != nil {
		t.Fatalf("primary weapon = %#v, want nil when lookup is missing", got.GetPrimaryWeapon())
	}
	if got.GetSecondaryWeapon().GetName() != "Side Knife" {
		t.Fatalf("secondary weapon = %#v", got.GetSecondaryWeapon())
	}
	if got.GetActiveArmor().GetName() != "Leather" {
		t.Fatalf("active armor = %#v, want starting armor fallback", got.GetActiveArmor())
	}
}
