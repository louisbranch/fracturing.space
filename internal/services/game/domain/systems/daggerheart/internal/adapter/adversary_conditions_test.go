package adapter

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestAdversaryHandlersPersistProjectionState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	a := NewAdapter(store, nil)

	createdAt := time.Date(2026, time.March, 20, 10, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	if err := a.HandleAdversaryCreated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
		Timestamp:  createdAt,
	}, payload.AdversaryCreatedPayload{
		AdversaryID:      ids.AdversaryID(" adv-1 "),
		AdversaryEntryID: " adversary.goblin ",
		Name:             " Goblin Cutter ",
		Kind:             " minion ",
		SessionID:        ids.SessionID(" sess-1 "),
		SceneID:          ids.SceneID(" scene-1 "),
		Notes:            " first wave ",
		HP:               6,
		HPMax:            6,
		Stress:           1,
		StressMax:        2,
		Evasion:          10,
		Major:            4,
		Severe:           8,
		Armor:            1,
		FeatureStates: []rules.AdversaryFeatureState{{
			FeatureID:       " feature.focus ",
			Status:          " active ",
			FocusedTargetID: " char-1 ",
		}},
		PendingExperience: &rules.AdversaryPendingExperience{Name: " Grudge ", Modifier: 2},
		SpotlightGateID:   ids.GateID(" gate-1 "),
		SpotlightCount:    3,
	}); err != nil {
		t.Fatalf("HandleAdversaryCreated() returned error: %v", err)
	}

	got := store.adversaries[profileKey("camp-1", "adv-1")]
	if got.AdversaryEntryID != "adversary.goblin" || got.Name != "Goblin Cutter" || got.Kind != "minion" {
		t.Fatalf("created adversary = %+v, want trimmed identity fields", got)
	}
	if got.SessionID != "sess-1" || got.SceneID != "scene-1" || got.Notes != "first wave" {
		t.Fatalf("created adversary = %+v, want trimmed scene fields", got)
	}
	if got.PendingExperience == nil || got.PendingExperience.Name != "Grudge" || got.PendingExperience.Modifier != 2 {
		t.Fatalf("created pending experience = %#v, want trimmed pending experience", got.PendingExperience)
	}
	wantFeatureStates := []projectionstore.DaggerheartAdversaryFeatureState{{
		FeatureID:       "feature.focus",
		Status:          "active",
		FocusedTargetID: "char-1",
	}}
	if !reflect.DeepEqual(got.FeatureStates, wantFeatureStates) {
		t.Fatalf("created feature states = %#v, want %#v", got.FeatureStates, wantFeatureStates)
	}
	if got.CreatedAt != createdAt.UTC() || got.UpdatedAt != createdAt.UTC() {
		t.Fatalf("created timestamps = (%s, %s), want both %s", got.CreatedAt, got.UpdatedAt, createdAt.UTC())
	}

	got.Conditions = []projectionstore.DaggerheartConditionState{{
		ID:    "hidden",
		Class: string(rules.ConditionClassStandard),
		Code:  rules.ConditionHidden,
	}}
	store.adversaries[profileKey("camp-1", "adv-1")] = got

	updatedAt := createdAt.Add(2 * time.Hour)
	if err := a.HandleAdversaryUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
		Timestamp:  updatedAt,
	}, payload.AdversaryUpdatedPayload{
		AdversaryID:      ids.AdversaryID(" adv-1 "),
		AdversaryEntryID: " adversary.hobgoblin ",
		Name:             " Hobgoblin Captain ",
		Kind:             " elite ",
		SessionID:        ids.SessionID(" sess-2 "),
		SceneID:          ids.SceneID(" scene-2 "),
		Notes:            " reinforced ",
		HP:               8,
		HPMax:            8,
		Stress:           2,
		StressMax:        3,
		Evasion:          12,
		Major:            5,
		Severe:           9,
		Armor:            2,
		FeatureStates: []rules.AdversaryFeatureState{{
			FeatureID: " feature.command ",
			Status:    " spent ",
		}},
		SpotlightGateID: ids.GateID(" gate-2 "),
		SpotlightCount:  1,
	}); err != nil {
		t.Fatalf("HandleAdversaryUpdated() returned error: %v", err)
	}

	got = store.adversaries[profileKey("camp-1", "adv-1")]
	if got.CreatedAt != createdAt.UTC() || got.UpdatedAt != updatedAt.UTC() {
		t.Fatalf("updated timestamps = (%s, %s), want created preserved and updated refreshed", got.CreatedAt, got.UpdatedAt)
	}
	if len(got.Conditions) != 1 || got.Conditions[0].Code != rules.ConditionHidden {
		t.Fatalf("updated conditions = %#v, want preserved hidden condition", got.Conditions)
	}
	if got.PendingExperience != nil {
		t.Fatalf("updated pending experience = %#v, want nil", got.PendingExperience)
	}
	if got.FeatureStates[0].FeatureID != "feature.command" || got.FeatureStates[0].Status != "spent" {
		t.Fatalf("updated feature states = %#v, want trimmed replacement state", got.FeatureStates)
	}

	hpAfter := 4
	armorAfter := 0
	damageAt := updatedAt.Add(30 * time.Minute)
	if err := a.HandleAdversaryDamageApplied(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
		Timestamp:  damageAt,
	}, payload.AdversaryDamageAppliedPayload{
		AdversaryID: ids.AdversaryID(" adv-1 "),
		Hp:          &hpAfter,
		Armor:       &armorAfter,
	}); err != nil {
		t.Fatalf("HandleAdversaryDamageApplied() returned error: %v", err)
	}
	got = store.adversaries[profileKey("camp-1", "adv-1")]
	if got.HP != 4 || got.Armor != 0 || got.UpdatedAt != damageAt.UTC() {
		t.Fatalf("damaged adversary = %+v, want hp=4 armor=0 updated_at refreshed", got)
	}

	if err := a.HandleAdversaryDeleted(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.AdversaryDeletedPayload{
		AdversaryID: ids.AdversaryID(" adv-1 "),
	}); err != nil {
		t.Fatalf("HandleAdversaryDeleted() returned error: %v", err)
	}
	if _, ok := store.adversaries[profileKey("camp-1", "adv-1")]; ok {
		t.Fatal("adversary still present after delete")
	}
	if got := ToProjectionAdversaryPendingExperience(nil); got != nil {
		t.Fatalf("ToProjectionAdversaryPendingExperience(nil) = %#v, want nil", got)
	}
}

func TestAdversaryHandlersValidateAndWrapStoreErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	a := NewAdapter(store, nil)

	if err := a.HandleAdversaryCreated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.AdversaryCreatedPayload{
		AdversaryID: ids.AdversaryID("adv-1"),
		HP:          1,
		HPMax:       0,
	}); err == nil || !strings.Contains(err.Error(), "hp_max must be positive") {
		t.Fatalf("HandleAdversaryCreated() error = %v, want hp_max validation", err)
	}

	if err := a.HandleAdversaryUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.AdversaryUpdatedPayload{
		AdversaryID: ids.AdversaryID("missing"),
		HP:          1,
		HPMax:       1,
	}); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("HandleAdversaryUpdated() error = %v, want storage.ErrNotFound", err)
	}

	store.adversaries[profileKey("camp-1", "adv-1")] = projectionstore.DaggerheartAdversary{
		CampaignID:  "camp-1",
		AdversaryID: "adv-1",
		HP:          4,
		HPMax:       4,
		StressMax:   1,
		Evasion:     10,
		Major:       2,
		Severe:      4,
	}
	hpTooHigh := 5
	if err := a.HandleAdversaryDamageApplied(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.AdversaryDamageAppliedPayload{
		AdversaryID: ids.AdversaryID("adv-1"),
		Hp:          &hpTooHigh,
	}); err == nil || !strings.Contains(err.Error(), "hp must be in range 0..4") {
		t.Fatalf("HandleAdversaryDamageApplied() error = %v, want hp bounds error", err)
	}

	store.getAdversaryErr = errors.New("read failed")
	if err := a.ApplyAdversaryConditionPatch(ctx, "camp-1", "adv-1", nil); err == nil || !strings.Contains(err.Error(), "get daggerheart adversary: read failed") {
		t.Fatalf("ApplyAdversaryConditionPatch() get error = %v, want wrapped get error", err)
	}
	store.getAdversaryErr = nil

	store.putAdversaryErr = errors.New("write failed")
	if err := a.ApplyAdversaryConditionPatch(ctx, "camp-1", "adv-1", []rules.ConditionState{mustStandardCondition(t, rules.ConditionHidden)}); err == nil || !strings.Contains(err.Error(), "put daggerheart adversary: write failed") {
		t.Fatalf("ApplyAdversaryConditionPatch() put error = %v, want wrapped put error", err)
	}
	store.putAdversaryErr = nil

	store.deleteAdversaryErr = errors.New("delete failed")
	if err := a.HandleAdversaryDeleted(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.AdversaryDeletedPayload{
		AdversaryID: ids.AdversaryID("adv-1"),
	}); err == nil || !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("HandleAdversaryDeleted() error = %v, want delete error", err)
	}
}

func TestConditionHandlersNormalizeAndPersistConditions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	store.profiles[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		ArmorMax:    4,
	}
	store.states[profileKey("camp-1", "char-1")] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       4,
	}
	store.adversaries[profileKey("camp-1", "adv-1")] = projectionstore.DaggerheartAdversary{
		CampaignID:  "camp-1",
		AdversaryID: "adv-1",
	}
	a := NewAdapter(store, nil)

	hidden := mustStandardCondition(t, rules.ConditionHidden)
	vulnerable := mustStandardCondition(t, rules.ConditionVulnerable)
	if err := a.HandleConditionChanged(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.ConditionChangedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Conditions:  []rules.ConditionState{vulnerable, hidden, hidden},
		RollSeq:     uint64Ptr(2),
	}); err != nil {
		t.Fatalf("HandleConditionChanged() returned error: %v", err)
	}
	if got := store.states[profileKey("camp-1", "char-1")].Conditions; len(got) != 2 || got[0].Code != rules.ConditionHidden || got[1].Code != rules.ConditionVulnerable {
		t.Fatalf("character conditions = %#v, want hidden then vulnerable", got)
	}

	if err := a.HandleAdversaryConditionChanged(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.AdversaryConditionChangedPayload{
		AdversaryID: ids.AdversaryID("adv-1"),
		Conditions:  []rules.ConditionState{vulnerable, hidden, hidden},
		RollSeq:     uint64Ptr(3),
	}); err != nil {
		t.Fatalf("HandleAdversaryConditionChanged() returned error: %v", err)
	}
	if got := store.adversaries[profileKey("camp-1", "adv-1")].Conditions; len(got) != 2 || got[0].Code != rules.ConditionHidden || got[1].Code != rules.ConditionVulnerable {
		t.Fatalf("adversary conditions = %#v, want hidden then vulnerable", got)
	}

	if err := a.HandleConditionChanged(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.ConditionChangedPayload{
		CharacterID: ids.CharacterID("char-1"),
		RollSeq:     uint64Ptr(0),
	}); err == nil || !strings.Contains(err.Error(), "condition_changed roll_seq must be positive") {
		t.Fatalf("HandleConditionChanged() zero roll_seq error = %v, want validation error", err)
	}

	invalid := rules.ConditionState{ID: "bad", Class: rules.ConditionClassStandard, Standard: "unknown"}
	if err := a.HandleConditionChanged(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.ConditionChangedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Conditions:  []rules.ConditionState{invalid},
	}); err == nil || !strings.Contains(err.Error(), "condition_changed conditions_after") {
		t.Fatalf("HandleConditionChanged() invalid conditions error = %v, want wrapped normalization error", err)
	}

	if err := a.HandleAdversaryConditionChanged(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.AdversaryConditionChangedPayload{
		AdversaryID: ids.AdversaryID("adv-1"),
		RollSeq:     uint64Ptr(0),
	}); err == nil || !strings.Contains(err.Error(), "adversary_condition_changed roll_seq must be positive") {
		t.Fatalf("HandleAdversaryConditionChanged() zero roll_seq error = %v, want validation error", err)
	}

	if err := a.HandleAdversaryConditionChanged(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.AdversaryConditionChangedPayload{
		AdversaryID: ids.AdversaryID("adv-1"),
		Conditions:  []rules.ConditionState{invalid},
	}); err == nil || !strings.Contains(err.Error(), "adversary_condition_changed conditions_after") {
		t.Fatalf("HandleAdversaryConditionChanged() invalid conditions error = %v, want wrapped normalization error", err)
	}
}

func mustStandardCondition(t *testing.T, code string) rules.ConditionState {
	t.Helper()

	state, err := rules.StandardConditionState(code)
	if err != nil {
		t.Fatalf("StandardConditionState(%q) returned error: %v", code, err)
	}
	return state
}

func uint64Ptr(v uint64) *uint64 { return &v }
