package daggerheart

import (
	"testing"

	daggerheartadapter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/adapter"
	daggerheartvalidator "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/validator"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestNormalizedActiveBeastformPtr_ClampedFields(t *testing.T) {
	// Negative trait/evasion/damage bonuses clamped to 0.
	active := &daggerheartstate.CharacterActiveBeastformState{
		BeastformID:  "bf-1",
		TraitBonus:   -1,
		EvasionBonus: -1,
		DamageBonus:  -1,
		DamageDice:   []daggerheartstate.CharacterDamageDie{{Count: 2, Sides: 6}},
	}
	got := daggerheartstate.NormalizedActiveBeastformPtr(active)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.TraitBonus != 0 {
		t.Fatalf("trait bonus = %d, want 0", got.TraitBonus)
	}
	if got.EvasionBonus != 0 {
		t.Fatalf("evasion bonus = %d, want 0", got.EvasionBonus)
	}
	if got.DamageBonus != 0 {
		t.Fatalf("damage bonus = %d, want 0", got.DamageBonus)
	}
}

func TestCompanionStateFromProjection(t *testing.T) {
	// nil returns nil.
	if got := daggerheartadapter.CompanionStateFromProjection(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil converts.
	value := &projectionstore.DaggerheartCompanionState{
		Status:             daggerheartstate.CompanionStatusAway,
		ActiveExperienceID: "exp-1",
	}
	got := daggerheartadapter.CompanionStateFromProjection(value)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Status != daggerheartstate.CompanionStatusAway || got.ActiveExperienceID != "exp-1" {
		t.Fatalf("got = %+v, want away/exp-1", got)
	}
}

func TestNormalizedClassStatePtr(t *testing.T) {
	// nil returns nil.
	if got := daggerheartdecider.NormalizedClassStatePtr(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil returns normalized copy.
	state := &daggerheartstate.CharacterClassState{AttackBonusUntilRest: -1, FocusTargetID: " char-1 "}
	got := daggerheartdecider.NormalizedClassStatePtr(state)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.AttackBonusUntilRest != 0 {
		t.Fatalf("attack bonus = %d, want 0", got.AttackBonusUntilRest)
	}
	if got.FocusTargetID != "char-1" {
		t.Fatalf("focus target = %q, want char-1", got.FocusTargetID)
	}
}

func TestHasClassStateFieldChange(t *testing.T) {
	// Both nil.
	if daggerheartvalidator.HasClassStateFieldChange(nil, nil) {
		t.Fatal("nil/nil should be no change")
	}
	// One nil.
	s := &daggerheartstate.CharacterClassState{}
	if !daggerheartvalidator.HasClassStateFieldChange(nil, s) {
		t.Fatal("nil/non-nil should be change")
	}
	if !daggerheartvalidator.HasClassStateFieldChange(s, nil) {
		t.Fatal("non-nil/nil should be change")
	}
	// Same values.
	a := &daggerheartstate.CharacterClassState{AttackBonusUntilRest: 2}
	b := &daggerheartstate.CharacterClassState{AttackBonusUntilRest: 2}
	if daggerheartvalidator.HasClassStateFieldChange(a, b) {
		t.Fatal("same values should be no change")
	}
	// Different values.
	c := &daggerheartstate.CharacterClassState{AttackBonusUntilRest: 3}
	if !daggerheartvalidator.HasClassStateFieldChange(a, c) {
		t.Fatal("different values should be change")
	}
}

func TestHasRestTakeMutation_AllBranches(t *testing.T) {
	// Fear change.
	if !daggerheartvalidator.HasRestTakeMutation(daggerheartpayload.RestTakePayload{GMFearBefore: 1, GMFearAfter: 2}) {
		t.Fatal("fear change should be mutation")
	}
	// Short rests change.
	if !daggerheartvalidator.HasRestTakeMutation(daggerheartpayload.RestTakePayload{ShortRestsBefore: 0, ShortRestsAfter: 1}) {
		t.Fatal("short rests change should be mutation")
	}
	// RefreshRest.
	if !daggerheartvalidator.HasRestTakeMutation(daggerheartpayload.RestTakePayload{RefreshRest: true}) {
		t.Fatal("refresh rest should be mutation")
	}
	// RefreshLongRest.
	if !daggerheartvalidator.HasRestTakeMutation(daggerheartpayload.RestTakePayload{RefreshLongRest: true}) {
		t.Fatal("refresh long rest should be mutation")
	}
	// Interrupted.
	if !daggerheartvalidator.HasRestTakeMutation(daggerheartpayload.RestTakePayload{Interrupted: true}) {
		t.Fatal("interrupted should be mutation")
	}
	// CampaignCountdownAdvances.
	if !daggerheartvalidator.HasRestTakeMutation(daggerheartpayload.RestTakePayload{CampaignCountdownAdvances: []daggerheartpayload.CampaignCountdownAdvancePayload{{}}}) {
		t.Fatal("campaign countdown advances should be mutation")
	}
	// DowntimeMoves.
	if !daggerheartvalidator.HasRestTakeMutation(daggerheartpayload.RestTakePayload{DowntimeMoves: []daggerheartpayload.DowntimeMoveAppliedPayload{{}}}) {
		t.Fatal("downtime moves should be mutation")
	}
	// Participants only.
	if !daggerheartvalidator.HasRestTakeMutation(daggerheartpayload.RestTakePayload{Participants: []ids.CharacterID{"char-1"}}) {
		t.Fatal("participants should be mutation")
	}
	// No mutation (empty payload with zero values matching).
	if daggerheartvalidator.HasRestTakeMutation(daggerheartpayload.RestTakePayload{}) {
		t.Fatal("empty payload should be no mutation")
	}
}
