package daggerheart

import (
	"testing"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestCharacterSubclassState_IsZero(t *testing.T) {
	if !(daggerheartstate.CharacterSubclassState{}).IsZero() {
		t.Fatal("zero-value subclass state should be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{BattleRitualUsedThisLongRest: true}).IsZero() {
		t.Fatal("state with battle ritual used should not be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{GiftedPerformerRelaxingSongUses: 1}).IsZero() {
		t.Fatal("state with song uses should not be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{TranscendenceActive: true}).IsZero() {
		t.Fatal("state with transcendence active should not be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{ElementalChannel: daggerheartstate.ElementalChannelFire}).IsZero() {
		t.Fatal("state with elemental channel should not be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{NemesisTargetID: "adv-1"}).IsZero() {
		t.Fatal("state with nemesis target should not be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{RousingSpeechUsedThisLongRest: true}).IsZero() {
		t.Fatal("state with rousing speech used should not be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{WardensProtectionUsedThisLongRest: true}).IsZero() {
		t.Fatal("state with wardens protection used should not be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{ContactsEverywhereUsesThisSession: 2}).IsZero() {
		t.Fatal("state with contacts everywhere should not be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{SparingTouchUsesThisLongRest: 1}).IsZero() {
		t.Fatal("state with sparing touch should not be IsZero")
	}
	if (daggerheartstate.CharacterSubclassState{ClarityOfNatureUsedThisLongRest: true}).IsZero() {
		t.Fatal("state with clarity of nature should not be IsZero")
	}
}

func TestCharacterSubclassState_Normalized_ClampsNegatives(t *testing.T) {
	state := daggerheartstate.CharacterSubclassState{
		GiftedPerformerRelaxingSongUses:        -1,
		GiftedPerformerEpicSongUses:            -5,
		GiftedPerformerHeartbreakingSongUses:   -3,
		ContactsEverywhereUsesThisSession:      -1,
		ContactsEverywhereActionDieBonus:       -2,
		ContactsEverywhereDamageDiceBonusCount: -1,
		SparingTouchUsesThisLongRest:           -1,
		ElementalistActionBonus:                -1,
		ElementalistDamageBonus:                -1,
		TranscendenceTraitBonusValue:           -1,
		TranscendenceProficiencyBonus:          -1,
		TranscendenceEvasionBonus:              -1,
		TranscendenceSevereThresholdBonus:      -1,
	}
	got := state.Normalized()
	if got.GiftedPerformerRelaxingSongUses != 0 {
		t.Fatalf("relaxing song uses = %d, want 0", got.GiftedPerformerRelaxingSongUses)
	}
	if got.GiftedPerformerEpicSongUses != 0 {
		t.Fatalf("epic song uses = %d, want 0", got.GiftedPerformerEpicSongUses)
	}
	if got.ContactsEverywhereUsesThisSession != 0 {
		t.Fatalf("contacts everywhere = %d, want 0", got.ContactsEverywhereUsesThisSession)
	}
	if got.SparingTouchUsesThisLongRest != 0 {
		t.Fatalf("sparing touch = %d, want 0", got.SparingTouchUsesThisLongRest)
	}
	if got.ElementalistActionBonus != 0 {
		t.Fatalf("elementalist action bonus = %d, want 0", got.ElementalistActionBonus)
	}
}

func TestCharacterSubclassState_Normalized_TranscendenceInactive(t *testing.T) {
	state := daggerheartstate.CharacterSubclassState{
		TranscendenceActive:               false,
		TranscendenceTraitBonusTarget:     "agility",
		TranscendenceTraitBonusValue:      2,
		TranscendenceProficiencyBonus:     1,
		TranscendenceEvasionBonus:         1,
		TranscendenceSevereThresholdBonus: 3,
	}
	got := state.Normalized()
	if got.TranscendenceTraitBonusTarget != "" {
		t.Fatalf("inactive transcendence trait target = %q, want empty", got.TranscendenceTraitBonusTarget)
	}
	if got.TranscendenceTraitBonusValue != 0 {
		t.Fatalf("inactive transcendence trait value = %d, want 0", got.TranscendenceTraitBonusValue)
	}
	if got.TranscendenceProficiencyBonus != 0 {
		t.Fatalf("inactive transcendence proficiency = %d, want 0", got.TranscendenceProficiencyBonus)
	}
}

func TestCharacterSubclassState_Normalized_ElementalChannel(t *testing.T) {
	// Valid channel preserved.
	got := daggerheartstate.CharacterSubclassState{ElementalChannel: " Fire "}.Normalized()
	if got.ElementalChannel != daggerheartstate.ElementalChannelFire {
		t.Fatalf("elemental channel = %q, want %q", got.ElementalChannel, daggerheartstate.ElementalChannelFire)
	}
	// Invalid channel cleared.
	got = daggerheartstate.CharacterSubclassState{ElementalChannel: "lightning"}.Normalized()
	if got.ElementalChannel != "" {
		t.Fatalf("invalid elemental channel = %q, want empty", got.ElementalChannel)
	}
}
