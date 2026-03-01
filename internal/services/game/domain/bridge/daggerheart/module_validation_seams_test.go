package daggerheart

import (
	"encoding/json"
	"testing"
)

func TestValidationCommonHelpers(t *testing.T) {
	if err := requireTrimmedValue(" ", "character_id"); err == nil {
		t.Fatal("expected required string validation error")
	}
	if err := requireTrimmedValue("char-1", "character_id"); err != nil {
		t.Fatalf("requireTrimmedValue(valid): %v", err)
	}
	if err := requirePositive(0, "max"); err == nil {
		t.Fatal("expected positive validation error")
	}
	if err := requirePositive(2, "max"); err != nil {
		t.Fatalf("requirePositive(valid): %v", err)
	}
	if err := requireRange(5, 0, 4, "marks"); err == nil {
		t.Fatal("expected range validation error")
	}
	if err := requireRange(3, 0, 4, "marks"); err != nil {
		t.Fatalf("requireRange(valid): %v", err)
	}
}

func TestValidateGMFearSetPayload_Branches(t *testing.T) {
	if err := validateGMFearSetPayload(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected missing after error")
	}
	if err := validateGMFearSetPayload(json.RawMessage(`{"after":100}`)); err == nil {
		t.Fatal("expected out-of-range after error")
	}
	if err := validateGMFearSetPayload(json.RawMessage(`{"after":3}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateLoadoutSwapPayload_Branches(t *testing.T) {
	if err := validateLoadoutSwapPayload(json.RawMessage(`{"character_id":" ","card_id":"c1"}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := validateLoadoutSwapPayload(json.RawMessage(`{"character_id":"char-1","card_id":" "}`)); err == nil {
		t.Fatal("expected missing card_id error")
	}
	if err := validateLoadoutSwapPayload(json.RawMessage(`{"character_id":"char-1","card_id":"card-1","from":"active","to":"vault"}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateRestTakePayload_Branches(t *testing.T) {
	if err := validateRestTakePayload(json.RawMessage(`{"rest_type":" "}`)); err == nil {
		t.Fatal("expected missing rest_type error")
	}
	if err := validateRestTakePayload(json.RawMessage(`{"rest_type":"short","long_term_countdown":{"countdown_id":"","before":1,"after":2}}`)); err == nil {
		t.Fatal("expected invalid long_term_countdown error")
	}
	if err := validateRestTakePayload(json.RawMessage(`{"rest_type":"short","gm_fear_before":1,"gm_fear_after":1,"short_rests_before":1,"short_rests_after":1}`)); err == nil {
		t.Fatal("expected no-mutation error")
	}
	if err := validateRestTakePayload(json.RawMessage(`{"rest_type":"short","gm_fear_before":1,"gm_fear_after":2}`)); err != nil {
		t.Fatalf("expected valid payload, got: %v", err)
	}
}

func TestValidateCharacterTemporaryArmorApplyPayload_Branches(t *testing.T) {
	if err := validateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":" ","source":"ritual","duration":"short_rest","amount":1}`)); err == nil {
		t.Fatal("expected missing character_id error")
	}
	if err := validateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":"char-1","source":" ","duration":"short_rest","amount":1}`)); err == nil {
		t.Fatal("expected missing source error")
	}
	if err := validateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":"char-1","source":"ritual","duration":"minute","amount":1}`)); err == nil {
		t.Fatal("expected invalid duration error")
	}
	if err := validateCharacterTemporaryArmorApplyPayload(json.RawMessage(`{"character_id":"char-1","source":"ritual","duration":"short_rest","amount":0}`)); err == nil {
		t.Fatal("expected amount-positive error")
	}
}

func TestValidateAdversaryCreateUpdatePayload_Branches(t *testing.T) {
	if err := validateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":" ","name":"Goblin"}`)); err == nil {
		t.Fatal("expected missing adversary_id error")
	}
	if err := validateAdversaryCreatePayload(json.RawMessage(`{"adversary_id":"adv-1","name":" "}`)); err == nil {
		t.Fatal("expected missing name error")
	}
	if err := validateAdversaryUpdatePayload(json.RawMessage(`{"adversary_id":"adv-1","name":" "}`)); err == nil {
		t.Fatal("expected update missing name error")
	}
}
