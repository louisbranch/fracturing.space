package charactertransport

import (
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
)

func TestCharacterToProto(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	got := CharacterToProto(storage.CharacterRecord{
		ID:            "char-1",
		CampaignID:    "camp-1",
		ParticipantID: "part-1",
		Name:          "Rook",
		Kind:          character.KindPC,
		Notes:         "note",
		AvatarSetID:   "set-1",
		AvatarAssetID: "asset-1",
		Pronouns:      sharedpronouns.PronounSheHer,
		Aliases:       []string{"Alias"},
		CreatedAt:     now,
		UpdatedAt:     now,
	})

	if got.GetId() != "char-1" || got.GetCampaignId() != "camp-1" || got.GetName() != "Rook" {
		t.Fatalf("character identity mismatch: %+v", got)
	}
	if got.GetKind() != campaignv1.CharacterKind_PC {
		t.Fatalf("kind = %v", got.GetKind())
	}
	if got.GetParticipantId().GetValue() != "part-1" {
		t.Fatalf("participant id = %q", got.GetParticipantId().GetValue())
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
	if len(conditions) != 2 || conditions[0] != daggerheart.ConditionHidden || conditions[1] != daggerheart.ConditionVulnerable {
		t.Fatalf("conditions = %#v", conditions)
	}
	if _, err := DaggerheartConditionsFromProto([]daggerheartv1.DaggerheartCondition{daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED}); err == nil {
		t.Fatal("expected unspecified condition error")
	}
	if got := DaggerheartConditionsToProto([]string{daggerheart.ConditionRestrained, daggerheart.ConditionHidden}); len(got) != 2 {
		t.Fatalf("conditions to proto = %#v", got)
	}
	if DaggerheartExperiencesToProto(nil) != nil {
		t.Fatal("nil experiences should stay nil")
	}

	state, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS)
	if err != nil || state != daggerheart.LifeStateUnconscious {
		t.Fatalf("life state from proto = %q err=%v", state, err)
	}
	if alive, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE); err != nil || alive != daggerheart.LifeStateAlive {
		t.Fatalf("alive state = %q err=%v", alive, err)
	}
	if blaze, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY); err != nil || blaze != daggerheart.LifeStateBlazeOfGlory {
		t.Fatalf("blaze state = %q err=%v", blaze, err)
	}
	if dead, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD); err != nil || dead != daggerheart.LifeStateDead {
		t.Fatalf("dead state = %q err=%v", dead, err)
	}
	if _, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED); err == nil {
		t.Fatal("expected unspecified life state error")
	}
	if _, err := DaggerheartLifeStateFromProto(daggerheartv1.DaggerheartLifeState(99)); err == nil {
		t.Fatal("expected invalid life state error")
	}
	if DaggerheartLifeStateToProto(daggerheart.LifeStateDead) != daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD {
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
			ID:       daggerheart.ConditionHidden,
			Class:    "standard",
			Standard: daggerheart.ConditionHidden,
			Code:     daggerheart.ConditionHidden,
			Label:    daggerheart.ConditionHidden,
		}},
		LifeState: daggerheart.LifeStateAlive,
	})
	if state.GetDaggerheart().GetHp() != 10 || len(state.GetDaggerheart().GetConditionStates()) != 1 {
		t.Fatalf("state = %+v", state.GetDaggerheart())
	}
}
