package daggerheart

import (
	"testing"

	daggerheartadapter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/adapter"
	daggerheartvalidator "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/validator"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestCompanionProjectionStateFromProfile(t *testing.T) {
	// nil companion sheet returns nil.
	if got := daggerheartadapter.CompanionProjectionStateFromProfile(daggerheartstate.CharacterProfile{}); got != nil {
		t.Fatal("expected nil for nil companion sheet")
	}
	// Non-nil companion sheet returns present status.
	got := daggerheartadapter.CompanionProjectionStateFromProfile(daggerheartstate.CharacterProfile{
		CompanionSheet: &daggerheartstate.CharacterCompanionSheet{Name: "Scout"},
	})
	if got == nil {
		t.Fatal("expected non-nil for companion sheet")
	}
	if got.Status != daggerheartstate.CompanionStatusPresent {
		t.Fatalf("status = %q, want present", got.Status)
	}
}

func TestIsCharacterStatePatchNoMutation_ExtraBranches(t *testing.T) {
	hp5 := 5
	hope3 := 3
	hopeMax5 := 5
	stress0 := 0
	armor2 := 2
	alive := daggerheartstate.LifeStateAlive

	snapshot := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {HP: 5, Hope: 3, HopeMax: 5, Stress: 0, Armor: 2, LifeState: daggerheartstate.LifeStateAlive},
		},
		CharacterClassStates:    map[ids.CharacterID]daggerheartstate.CharacterClassState{},
		CharacterSubclassStates: map[ids.CharacterID]daggerheartstate.CharacterSubclassState{},
	}

	// All fields match → no mutation.
	if !daggerheartdecider.IsCharacterStatePatchNoMutation(snapshot, daggerheartpayload.CharacterStatePatchPayload{
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
	if daggerheartdecider.IsCharacterStatePatchNoMutation(snapshot, daggerheartpayload.CharacterStatePatchPayload{
		CharacterID: "char-1",
		HPAfter:     &hp10,
	}) {
		t.Fatal("different HP should be mutation")
	}

	// HopeMax mismatch → mutation.
	hopeMax3 := 3
	if daggerheartdecider.IsCharacterStatePatchNoMutation(snapshot, daggerheartpayload.CharacterStatePatchPayload{
		CharacterID:  "char-1",
		HopeMaxAfter: &hopeMax3,
	}) {
		t.Fatal("different hope max should be mutation")
	}

	// LifeState mismatch → mutation.
	dead := "dead"
	if daggerheartdecider.IsCharacterStatePatchNoMutation(snapshot, daggerheartpayload.CharacterStatePatchPayload{
		CharacterID:    "char-1",
		LifeStateAfter: &dead,
	}) {
		t.Fatal("different life state should be mutation")
	}

	// ClassState match → no mutation.
	classState := &daggerheartstate.CharacterClassState{}
	if !daggerheartdecider.IsCharacterStatePatchNoMutation(snapshot, daggerheartpayload.CharacterStatePatchPayload{
		CharacterID:     "char-1",
		ClassStateAfter: classState,
	}) {
		t.Fatal("matching class state should be no mutation")
	}

	// SubclassState match → no mutation.
	subclassState := &daggerheartstate.CharacterSubclassState{}
	if !daggerheartdecider.IsCharacterStatePatchNoMutation(snapshot, daggerheartpayload.CharacterStatePatchPayload{
		CharacterID:        "char-1",
		SubclassStateAfter: subclassState,
	}) {
		t.Fatal("matching subclass state should be no mutation")
	}
}

func TestValidateConditionSetPayload_ExtraBranches(t *testing.T) {
	hidden := rules.ConditionState{ID: "hidden", Class: rules.ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden"}
	vuln := rules.ConditionState{ID: "vulnerable", Class: rules.ConditionClassStandard, Standard: "vulnerable", Code: "vulnerable", Label: "Vulnerable"}

	// Removed without before → error.
	if err := daggerheartvalidator.ValidateConditionSetPayload(nil, []rules.ConditionState{hidden}, nil, []rules.ConditionState{vuln}); err == nil {
		t.Fatal("expected error for removed without before")
	}

	// Added mismatch with before → error.
	if err := daggerheartvalidator.ValidateConditionSetPayload([]rules.ConditionState{hidden}, []rules.ConditionState{hidden, vuln}, []rules.ConditionState{hidden}, nil); err == nil {
		t.Fatal("expected error for added mismatch with before")
	}

	// Added mismatch without before → error.
	if err := daggerheartvalidator.ValidateConditionSetPayload(nil, []rules.ConditionState{hidden}, []rules.ConditionState{vuln}, nil); err == nil {
		t.Fatal("expected error for added mismatch without before")
	}

	// Removed mismatch with before → error.
	if err := daggerheartvalidator.ValidateConditionSetPayload([]rules.ConditionState{hidden, vuln}, []rules.ConditionState{hidden}, nil, []rules.ConditionState{hidden}); err == nil {
		t.Fatal("expected error for removed mismatch with before")
	}

	// No change with before → error.
	if err := daggerheartvalidator.ValidateConditionSetPayload([]rules.ConditionState{hidden}, []rules.ConditionState{hidden}, nil, nil); err == nil {
		t.Fatal("expected error for no change with before")
	}

	// No change without before → error.
	if err := daggerheartvalidator.ValidateConditionSetPayload(nil, []rules.ConditionState{}, []rules.ConditionState{}, nil); err == nil {
		t.Fatal("expected error for empty change without before")
	}
}

func TestValidateBeastformTransformedPayload(t *testing.T) {
	// Missing character_id.
	data := `{"beastform_id":"bf-1","active_beastform":{"beastform_id":"bf-1","damage_dice":[{"count":1,"sides":6}]}}`
	if err := daggerheartvalidator.ValidateBeastformTransformedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing character_id")
	}
	// Missing beastform_id.
	data = `{"character_id":"char-1","active_beastform":{"beastform_id":"bf-1","damage_dice":[{"count":1,"sides":6}]}}`
	if err := daggerheartvalidator.ValidateBeastformTransformedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing beastform_id")
	}
	// Missing active_beastform.
	data = `{"character_id":"char-1","beastform_id":"bf-1"}`
	if err := daggerheartvalidator.ValidateBeastformTransformedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing active_beastform")
	}
	// Valid.
	data = `{"character_id":"char-1","beastform_id":"bf-1","active_beastform":{"beastform_id":"bf-1","damage_dice":[{"count":1,"sides":6}]}}`
	if err := daggerheartvalidator.ValidateBeastformTransformedPayload([]byte(data)); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
}

func TestValidateBeastformDroppedPayload(t *testing.T) {
	// Missing character_id.
	data := `{"beastform_id":"bf-1"}`
	if err := daggerheartvalidator.ValidateBeastformDroppedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing character_id")
	}
	// Missing beastform_id.
	data = `{"character_id":"char-1"}`
	if err := daggerheartvalidator.ValidateBeastformDroppedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing beastform_id")
	}
	// Valid.
	data = `{"character_id":"char-1","beastform_id":"bf-1"}`
	if err := daggerheartvalidator.ValidateBeastformDroppedPayload([]byte(data)); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
}

func TestValidateCompanionExperienceBegunPayload(t *testing.T) {
	// Missing character_id.
	data := `{"experience_id":"exp-1","companion_state":{"status":"away","active_experience_id":"exp-1"}}`
	if err := daggerheartvalidator.ValidateCompanionExperienceBegunPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing character_id")
	}
	// Missing experience_id.
	data = `{"character_id":"char-1","companion_state":{"status":"away","active_experience_id":"exp-1"}}`
	if err := daggerheartvalidator.ValidateCompanionExperienceBegunPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing experience_id")
	}
	// Missing companion_state.
	data = `{"character_id":"char-1","experience_id":"exp-1"}`
	if err := daggerheartvalidator.ValidateCompanionExperienceBegunPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing companion_state")
	}
	// Valid.
	data = `{"character_id":"char-1","experience_id":"exp-1","companion_state":{"status":"away","active_experience_id":"exp-1"}}`
	if err := daggerheartvalidator.ValidateCompanionExperienceBegunPayload([]byte(data)); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
}

func TestValidateCompanionReturnedPayload(t *testing.T) {
	// Missing character_id.
	data := `{"resolution":"success","companion_state":{"status":"present"}}`
	if err := daggerheartvalidator.ValidateCompanionReturnedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing character_id")
	}
	// Missing resolution.
	data = `{"character_id":"char-1","companion_state":{"status":"present"}}`
	if err := daggerheartvalidator.ValidateCompanionReturnedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing resolution")
	}
	// Missing companion_state.
	data = `{"character_id":"char-1","resolution":"success"}`
	if err := daggerheartvalidator.ValidateCompanionReturnedPayload([]byte(data)); err == nil {
		t.Fatal("expected error for missing companion_state")
	}
	// Valid.
	data = `{"character_id":"char-1","resolution":"success","companion_state":{"status":"present"}}`
	if err := daggerheartvalidator.ValidateCompanionReturnedPayload([]byte(data)); err != nil {
		t.Fatalf("valid payload: %v", err)
	}
}
