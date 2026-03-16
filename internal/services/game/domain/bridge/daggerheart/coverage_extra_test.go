package daggerheart

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestArmorCanMitigate_Branches(t *testing.T) {
	// Physical-only mode: only physical damage mitigated.
	if !armorCanMitigate(ArmorDamageRules{MitigationMode: string(contentstore.DaggerheartArmorMitigationModePhysicalOnly)}, DamageTypes{Physical: true}) {
		t.Fatal("physical_only should mitigate physical")
	}
	if armorCanMitigate(ArmorDamageRules{MitigationMode: string(contentstore.DaggerheartArmorMitigationModePhysicalOnly)}, DamageTypes{Magic: true}) {
		t.Fatal("physical_only should not mitigate magic")
	}
	if armorCanMitigate(ArmorDamageRules{MitigationMode: string(contentstore.DaggerheartArmorMitigationModePhysicalOnly)}, DamageTypes{Physical: true, Magic: true}) {
		t.Fatal("physical_only should not mitigate mixed")
	}

	// Magic-only mode: only magic damage mitigated.
	if !armorCanMitigate(ArmorDamageRules{MitigationMode: string(contentstore.DaggerheartArmorMitigationModeMagicOnly)}, DamageTypes{Magic: true}) {
		t.Fatal("magic_only should mitigate magic")
	}
	if armorCanMitigate(ArmorDamageRules{MitigationMode: string(contentstore.DaggerheartArmorMitigationModeMagicOnly)}, DamageTypes{Physical: true}) {
		t.Fatal("magic_only should not mitigate physical")
	}
	if armorCanMitigate(ArmorDamageRules{MitigationMode: string(contentstore.DaggerheartArmorMitigationModeMagicOnly)}, DamageTypes{Physical: true, Magic: true}) {
		t.Fatal("magic_only should not mitigate mixed")
	}

	// Default (any) mode: always mitigates.
	if !armorCanMitigate(ArmorDamageRules{}, DamageTypes{Physical: true, Magic: true}) {
		t.Fatal("default mode should mitigate any")
	}
}

func TestCompanionProjectionStateFromProfile(t *testing.T) {
	// nil companion sheet returns nil.
	if got := companionProjectionStateFromProfile(CharacterProfile{}); got != nil {
		t.Fatal("expected nil for nil companion sheet")
	}
	// Non-nil companion sheet returns present status.
	got := companionProjectionStateFromProfile(CharacterProfile{
		CompanionSheet: &CharacterCompanionSheet{Name: "Scout"},
	})
	if got == nil {
		t.Fatal("expected non-nil for companion sheet")
	}
	if got.Status != CompanionStatusPresent {
		t.Fatalf("status = %q, want present", got.Status)
	}
}

func TestIsCharacterStatePatchNoMutation_ExtraBranches(t *testing.T) {
	hp5 := 5
	hope3 := 3
	hopeMax5 := 5
	stress0 := 0
	armor2 := 2
	alive := LifeStateAlive

	snapshot := SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]CharacterState{
			"char-1": {HP: 5, Hope: 3, HopeMax: 5, Stress: 0, Armor: 2, LifeState: LifeStateAlive},
		},
		CharacterClassStates:    map[ids.CharacterID]CharacterClassState{},
		CharacterSubclassStates: map[ids.CharacterID]CharacterSubclassState{},
	}

	// All fields match → no mutation.
	if !isCharacterStatePatchNoMutation(snapshot, CharacterStatePatchPayload{
		CharacterID:    "char-1",
		HPAfter:        &hp5,
		HopeAfter:      &hope3,
		HopeMaxAfter:   &hopeMax5,
		StressAfter:    &stress0,
		ArmorAfter:     &armor2,
		LifeStateAfter: &alive,
	}) {
		t.Fatal("identical state should be no mutation")
	}

	// HP mismatch → mutation.
	hp10 := 10
	if isCharacterStatePatchNoMutation(snapshot, CharacterStatePatchPayload{
		CharacterID: "char-1",
		HPAfter:     &hp10,
	}) {
		t.Fatal("different HP should be mutation")
	}

	// HopeMax mismatch → mutation.
	hopeMax3 := 3
	if isCharacterStatePatchNoMutation(snapshot, CharacterStatePatchPayload{
		CharacterID:  "char-1",
		HopeMaxAfter: &hopeMax3,
	}) {
		t.Fatal("different hope max should be mutation")
	}

	// LifeState mismatch → mutation.
	dead := "dead"
	if isCharacterStatePatchNoMutation(snapshot, CharacterStatePatchPayload{
		CharacterID:    "char-1",
		LifeStateAfter: &dead,
	}) {
		t.Fatal("different life state should be mutation")
	}

	// ClassState match → no mutation.
	classState := &CharacterClassState{}
	if !isCharacterStatePatchNoMutation(snapshot, CharacterStatePatchPayload{
		CharacterID:     "char-1",
		ClassStateAfter: classState,
	}) {
		t.Fatal("matching class state should be no mutation")
	}

	// SubclassState match → no mutation.
	subclassState := &CharacterSubclassState{}
	if !isCharacterStatePatchNoMutation(snapshot, CharacterStatePatchPayload{
		CharacterID:        "char-1",
		SubclassStateAfter: subclassState,
	}) {
		t.Fatal("matching subclass state should be no mutation")
	}
}

func TestResolveDamageApplication_WardedMagicReduction(t *testing.T) {
	target := DamageTarget{
		HP:              10,
		Stress:          0,
		Armor:           3,
		MajorThreshold:  5,
		SevereThreshold: 9,
		ArmorRules: ArmorDamageRules{
			WardedMagicReduction:  true,
			WardedReductionAmount: 2,
		},
	}
	input := DamageApplyInput{
		Amount: 4,
		Types:  DamageTypes{Magic: true},
	}
	app, mitigated, err := ResolveDamageApplication(target, input)
	if err != nil {
		t.Fatalf("ResolveDamageApplication: %v", err)
	}
	if !mitigated {
		t.Fatal("warded magic reduction should mitigate")
	}
	_ = app
}

func TestValidateConditionSetPayload_ExtraBranches(t *testing.T) {
	hidden := ConditionState{ID: "hidden", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden"}
	vuln := ConditionState{ID: "vulnerable", Class: ConditionClassStandard, Standard: "vulnerable", Code: "vulnerable", Label: "Vulnerable"}

	// Removed without before → error.
	if err := validateConditionSetPayload(nil, []ConditionState{hidden}, nil, []ConditionState{vuln}); err == nil {
		t.Fatal("expected error for removed without before")
	}

	// Added mismatch with before → error.
	if err := validateConditionSetPayload([]ConditionState{hidden}, []ConditionState{hidden, vuln}, []ConditionState{hidden}, nil); err == nil {
		t.Fatal("expected error for added mismatch with before")
	}

	// Added mismatch without before → error.
	if err := validateConditionSetPayload(nil, []ConditionState{hidden}, []ConditionState{vuln}, nil); err == nil {
		t.Fatal("expected error for added mismatch without before")
	}

	// Removed mismatch with before → error.
	if err := validateConditionSetPayload([]ConditionState{hidden, vuln}, []ConditionState{hidden}, nil, []ConditionState{hidden}); err == nil {
		t.Fatal("expected error for removed mismatch with before")
	}

	// No change with before → error.
	if err := validateConditionSetPayload([]ConditionState{hidden}, []ConditionState{hidden}, nil, nil); err == nil {
		t.Fatal("expected error for no change with before")
	}

	// No change without before → error.
	if err := validateConditionSetPayload(nil, []ConditionState{}, []ConditionState{}, nil); err == nil {
		t.Fatal("expected error for empty change without before")
	}
}

func TestValidateBeastformTransformedPayload(t *testing.T) {
	// Missing character_id.
	data := `{"beastform_id":"bf-1","active_beastform":{"beastform_id":"bf-1","damage_dice":[{"count":1,"sides":6}]}}`
	if err := validateBeastformTransformedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing character_id")
	}
	// Missing beastform_id.
	data = `{"character_id":"char-1","active_beastform":{"beastform_id":"bf-1","damage_dice":[{"count":1,"sides":6}]}}`
	if err := validateBeastformTransformedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing beastform_id")
	}
	// Missing active_beastform.
	data = `{"character_id":"char-1","beastform_id":"bf-1"}`
	if err := validateBeastformTransformedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing active_beastform")
	}
	// Valid.
	data = `{"character_id":"char-1","beastform_id":"bf-1","active_beastform":{"beastform_id":"bf-1","damage_dice":[{"count":1,"sides":6}]}}`
	if err := validateBeastformTransformedPayload([]byte(data)); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
}

func TestValidateBeastformDroppedPayload(t *testing.T) {
	// Missing character_id.
	data := `{"beastform_id":"bf-1"}`
	if err := validateBeastformDroppedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing character_id")
	}
	// Missing beastform_id.
	data = `{"character_id":"char-1"}`
	if err := validateBeastformDroppedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing beastform_id")
	}
	// Valid.
	data = `{"character_id":"char-1","beastform_id":"bf-1"}`
	if err := validateBeastformDroppedPayload([]byte(data)); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
}

func TestValidateCompanionExperienceBegunPayload(t *testing.T) {
	// Missing character_id.
	data := `{"experience_id":"exp-1","companion_state":{"status":"away","active_experience_id":"exp-1"}}`
	if err := validateCompanionExperienceBegunPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing character_id")
	}
	// Missing experience_id.
	data = `{"character_id":"char-1","companion_state":{"status":"away","active_experience_id":"exp-1"}}`
	if err := validateCompanionExperienceBegunPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing experience_id")
	}
	// Missing companion_state.
	data = `{"character_id":"char-1","experience_id":"exp-1"}`
	if err := validateCompanionExperienceBegunPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing companion_state")
	}
	// Valid.
	data = `{"character_id":"char-1","experience_id":"exp-1","companion_state":{"status":"away","active_experience_id":"exp-1"}}`
	if err := validateCompanionExperienceBegunPayload([]byte(data)); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
}

func TestValidateCompanionReturnedPayload(t *testing.T) {
	// Missing character_id.
	data := `{"resolution":"success","companion_state":{"status":"present"}}`
	if err := validateCompanionReturnedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing character_id")
	}
	// Missing resolution.
	data = `{"character_id":"char-1","companion_state":{"status":"present"}}`
	if err := validateCompanionReturnedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing resolution")
	}
	// Missing companion_state.
	data = `{"character_id":"char-1","resolution":"success"}`
	if err := validateCompanionReturnedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing companion_state")
	}
	// Valid.
	data = `{"character_id":"char-1","resolution":"success","companion_state":{"status":"present"}}`
	if err := validateCompanionReturnedPayload([]byte(data)); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
}

func TestResolveDamageApplication_ArmorCannotMitigate(t *testing.T) {
	target := DamageTarget{
		HP:              10,
		Stress:          2,
		Armor:           3,
		MajorThreshold:  5,
		SevereThreshold: 9,
		ArmorRules: ArmorDamageRules{
			MitigationMode: string(contentstore.DaggerheartArmorMitigationModePhysicalOnly),
		},
	}
	input := DamageApplyInput{
		Amount: 4,
		Types:  DamageTypes{Magic: true},
	}
	app, _, err := ResolveDamageApplication(target, input)
	if err != nil {
		t.Fatalf("ResolveDamageApplication: %v", err)
	}
	// When armor can't mitigate, stress stays unchanged.
	if app.StressBefore != 2 || app.StressAfter != 2 {
		t.Fatalf("stress should stay at 2, got before=%d after=%d", app.StressBefore, app.StressAfter)
	}
}
