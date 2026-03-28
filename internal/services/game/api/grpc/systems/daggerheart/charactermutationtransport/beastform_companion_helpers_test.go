package charactermutationtransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartcontent "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestResolvedBeastformStateUsesEvolutionTraitAndFragileFlag(t *testing.T) {
	t.Parallel()

	state := resolvedBeastformState(daggerheartcontent.DaggerheartBeastformEntry{
		ID:           "wolf",
		Trait:        "agility",
		TraitBonus:   2,
		EvasionBonus: 1,
		Attack: daggerheartcontent.DaggerheartBeastformAttack{
			Range:      "Melee",
			DamageType: "physical",
			DamageDice: []daggerheartcontent.DaggerheartDamageDie{{Count: 2, Sides: 8}},
		},
		Features: []daggerheartcontent.DaggerheartBeastformFeature{{ID: "fragile"}},
	}, "instinct")

	if state.AttackTrait != "instinct" || state.BaseTrait != "agility" {
		t.Fatalf("state traits = %+v", state)
	}
	if !state.DropOnAnyHPMark {
		t.Fatalf("DropOnAnyHPMark = false, want true")
	}
	if len(state.DamageDice) != 1 || state.DamageDice[0].Count != 2 || state.DamageDice[0].Sides != 8 {
		t.Fatalf("damage dice = %+v", state.DamageDice)
	}
}

func TestResolvedBeastformStateFallsBackToBaseTraitAndNameFragile(t *testing.T) {
	t.Parallel()

	state := resolvedBeastformState(daggerheartcontent.DaggerheartBeastformEntry{
		ID:    "bear",
		Trait: "strength",
		Attack: daggerheartcontent.DaggerheartBeastformAttack{
			DamageBonus: 3,
		},
		Features: []daggerheartcontent.DaggerheartBeastformFeature{{Name: "Fragile"}},
	}, "")

	if state.AttackTrait != "strength" {
		t.Fatalf("AttackTrait = %q, want strength", state.AttackTrait)
	}
	if !beastformHasFragile([]daggerheartcontent.DaggerheartBeastformFeature{{Name: "Fragile"}}) {
		t.Fatal("beastformHasFragile() = false, want true")
	}
}

func TestCompanionProfileHelpers(t *testing.T) {
	t.Parallel()

	profile := projectionstore.DaggerheartCharacterProfile{
		CompanionSheet: &projectionstore.DaggerheartCompanionSheet{
			Experiences: []projectionstore.DaggerheartCompanionExperience{{ExperienceID: "exp-1"}},
		},
	}
	if !profileHasCompanionExperience(profile, "exp-1") {
		t.Fatal("profileHasCompanionExperience() = false, want true")
	}
	if profileHasCompanionExperience(profile, "missing") {
		t.Fatal("profileHasCompanionExperience() = true, want false")
	}
}

func TestCompanionStateForCharacterDefaultsAndNormalizes(t *testing.T) {
	t.Parallel()

	withSheet := projectionstore.DaggerheartCharacterProfile{CompanionSheet: &projectionstore.DaggerheartCompanionSheet{}}
	if got := companionStateForCharacter(withSheet, projectionstore.DaggerheartCharacterState{}); got.Status != daggerheartstate.CompanionStatusPresent {
		t.Fatalf("default companion state = %+v, want present", got)
	}

	got := companionStateForCharacter(withSheet, projectionstore.DaggerheartCharacterState{
		CompanionState: &projectionstore.DaggerheartCompanionState{
			Status:             daggerheartstate.CompanionStatusAway,
			ActiveExperienceID: "exp-2",
		},
	})
	if got.Status != daggerheartstate.CompanionStatusAway || got.ActiveExperienceID != "exp-2" {
		t.Fatalf("companion state = %+v", got)
	}
}

func TestCompanionReturnResolutionLabelAndPtr(t *testing.T) {
	t.Parallel()

	if got := companionReturnResolutionLabel(pb.DaggerheartCompanionReturnResolution_DAGGERHEART_COMPANION_RETURN_RESOLUTION_EXPERIENCE_COMPLETED); got != "experience_completed" {
		t.Fatalf("completed label = %q", got)
	}
	if got := companionReturnResolutionLabel(pb.DaggerheartCompanionReturnResolution_DAGGERHEART_COMPANION_RETURN_RESOLUTION_EARLY_RETURN); got != "early_return" {
		t.Fatalf("early label = %q", got)
	}
	if got := companionReturnResolutionLabel(pb.DaggerheartCompanionReturnResolution_DAGGERHEART_COMPANION_RETURN_RESOLUTION_UNSPECIFIED); got != "" {
		t.Fatalf("unspecified label = %q, want empty", got)
	}

	ptr := companionStatePtr(daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusAway, ActiveExperienceID: "exp-3"})
	if ptr == nil || ptr.Status != daggerheartstate.CompanionStatusAway || ptr.ActiveExperienceID != "exp-3" {
		t.Fatalf("companionStatePtr() = %+v", ptr)
	}
}
