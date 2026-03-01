package daggerheart

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyProfile_ResetDeleteError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.deleteCharacterProfileErr = errText("delete profile failed")
	adapter := NewAdapter(store)

	err := adapter.ApplyProfile(context.Background(), "camp-1", "char-1", json.RawMessage(`{"reset":true}`))
	if err == nil || !strings.Contains(err.Error(), "delete daggerheart profile: delete profile failed") {
		t.Fatalf("ApplyProfile(reset) error = %v, want wrapped delete error", err)
	}
}

func TestApplyProfile_InvalidProfileValidationError(t *testing.T) {
	store := newFaultDaggerheartStore()
	adapter := NewAdapter(store)

	// level/hp_max values violate profile.Validate invariants.
	err := adapter.ApplyProfile(context.Background(), "camp-1", "char-1", json.RawMessage(`{
		"level": 0,
		"hp_max": -1,
		"stress_max": 0,
		"evasion": 0,
		"major_threshold": 0,
		"severe_threshold": 0,
		"proficiency": 0,
		"armor_score": 0,
		"armor_max": 0
	}`))
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "validate daggerheart profile payload") {
		t.Fatalf("ApplyProfile() error = %v, want validation prefix", err)
	}
}

type errText string

func (e errText) Error() string { return string(e) }
