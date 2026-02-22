package daggerheart

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// assertTestSnapshotState extracts a *SnapshotState from an any value returned
// by a fold, accepting both value and pointer types for test convenience.
func assertTestSnapshotState(t *testing.T, v any) SnapshotState {
	t.Helper()
	switch typed := v.(type) {
	case SnapshotState:
		return typed
	case *SnapshotState:
		if typed != nil {
			return *typed
		}
	}
	t.Fatalf("expected SnapshotState or *SnapshotState, got %T", v)
	return SnapshotState{}
}

func TestSnapshotOrDefault_NilReturnsFactoryDefaults(t *testing.T) {
	s, hasState := snapshotOrDefault(nil)
	if hasState {
		t.Fatal("expected hasState=false for nil input")
	}
	if s.GMFear != GMFearDefault {
		t.Fatalf("gm fear = %d, want %d", s.GMFear, GMFearDefault)
	}
	if s.CharacterStates == nil {
		t.Fatal("CharacterStates should be initialized")
	}
	if s.AdversaryStates == nil {
		t.Fatal("AdversaryStates should be initialized")
	}
	if s.CountdownStates == nil {
		t.Fatal("CountdownStates should be initialized")
	}
}

func TestSnapshotOrDefault_ValueReturnsState(t *testing.T) {
	input := SnapshotState{CampaignID: "camp-1", GMFear: 5}
	s, hasState := snapshotOrDefault(input)
	if !hasState {
		t.Fatal("expected hasState=true for value input")
	}
	if s.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want camp-1", s.CampaignID)
	}
	if s.GMFear != 5 {
		t.Fatalf("gm fear = %d, want 5", s.GMFear)
	}
}

func TestSnapshotOrDefault_PointerReturnsState(t *testing.T) {
	input := &SnapshotState{CampaignID: "camp-2", GMFear: 3}
	s, hasState := snapshotOrDefault(input)
	if !hasState {
		t.Fatal("expected hasState=true for pointer input")
	}
	if s.CampaignID != "camp-2" {
		t.Fatalf("campaign id = %s, want camp-2", s.CampaignID)
	}
	if s.GMFear != 3 {
		t.Fatalf("gm fear = %d, want 3", s.GMFear)
	}
}

func TestSnapshotOrDefault_NilPointerReturnsFactoryDefaults(t *testing.T) {
	var input *SnapshotState
	s, hasState := snapshotOrDefault(input)
	if hasState {
		t.Fatal("expected hasState=false for nil pointer input")
	}
	if s.GMFear != GMFearDefault {
		t.Fatalf("gm fear = %d, want %d", s.GMFear, GMFearDefault)
	}
	if s.CharacterStates == nil {
		t.Fatal("CharacterStates should be initialized")
	}
}

func TestFolderApplyGMFearChanged_UpdatesState(t *testing.T) {
	projector := NewFolder()
	state := SnapshotState{CampaignID: "camp-1", GMFear: 2}

	payload, err := json.Marshal(GMFearChangedPayload{Before: 2, After: 5, Reason: "shift"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	updated, err := projector.Fold(state, event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.daggerheart.gm_fear_changed"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot := assertTestSnapshotState(t, updated)
	if snapshot.GMFear != 5 {
		t.Fatalf("gm fear = %d, want %d", snapshot.GMFear, 5)
	}
	if snapshot.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want %s", snapshot.CampaignID, "camp-1")
	}
}

func TestFolderApplyCharacterStatePatched_StoresCharacterState(t *testing.T) {
	projector := NewFolder()
	hpAfter := 6
	hopeAfter := 2
	payload, err := json.Marshal(CharacterStatePatchedPayload{
		CharacterID:    "char-1",
		HPAfter:        &hpAfter,
		HopeAfter:      &hopeAfter,
		HopeMaxAfter:   nil,
		StressAfter:    nil,
		ArmorAfter:     nil,
		LifeStateAfter: nil,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := projector.Fold(SnapshotState{CampaignID: "camp-1"}, event.Event{
		CampaignID:    "camp-1",
		Type:          EventTypeCharacterStatePatched,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot := assertTestSnapshotState(t, updated)
	character, ok := snapshot.CharacterStates["char-1"]
	if !ok {
		t.Fatal("expected character state")
	}
	if character.CampaignID != "camp-1" {
		t.Fatalf("character campaign id = %s, want %s", character.CampaignID, "camp-1")
	}
	if character.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", character.CharacterID, "char-1")
	}
	if character.HP != hpAfter {
		t.Fatalf("hp = %d, want %d", character.HP, hpAfter)
	}
	if character.Hope != hopeAfter {
		t.Fatalf("hope = %d, want %d", character.Hope, hopeAfter)
	}
}

func TestFolderApplyCharacterStatePatched_DoesNotMutateFromBeforeOnly(t *testing.T) {
	projector := NewFolder()
	hpBefore := 7
	payload, err := json.Marshal(CharacterStatePatchedPayload{
		CharacterID: "char-1",
		HPBefore:    &hpBefore,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := projector.Fold(SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[string]CharacterState{
			"char-1": {
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				HP:          0,
			},
		},
	}, event.Event{
		CampaignID:    "camp-1",
		Type:          EventTypeCharacterStatePatched,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot := assertTestSnapshotState(t, updated)
	character, ok := snapshot.CharacterStates["char-1"]
	if !ok {
		t.Fatal("expected character state")
	}
	if character.HP != 0 {
		t.Fatalf("hp = %d, want %d", character.HP, 0)
	}
}

func TestFolderApplyAdversaryUpdated_AppliesZeroAndEmptyValues(t *testing.T) {
	projector := NewFolder()
	state := SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[string]AdversaryState{
			"adv-1": {
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				Name:        "Goblin",
				Kind:        "bruiser",
				SessionID:   "sess-1",
				Notes:       "old notes",
				HP:          6,
				HPMax:       8,
				Stress:      3,
				StressMax:   3,
				Evasion:     2,
				Major:       2,
				Severe:      3,
				Armor:       1,
			},
		},
	}

	payload, err := json.Marshal(AdversaryUpdatedPayload{
		AdversaryID: "adv-1",
		Name:        "Goblin",
		Kind:        "",
		SessionID:   "",
		Notes:       "",
		HP:          0,
		HPMax:       8,
		Stress:      0,
		StressMax:   3,
		Evasion:     0,
		Major:       0,
		Severe:      0,
		Armor:       0,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := projector.Fold(state, event.Event{
		CampaignID:    "camp-1",
		Type:          EventTypeAdversaryUpdated,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot := assertTestSnapshotState(t, updated)
	adversary, ok := snapshot.AdversaryStates["adv-1"]
	if !ok {
		t.Fatal("expected adversary state")
	}
	if adversary.Kind != "" {
		t.Fatalf("kind = %q, want empty", adversary.Kind)
	}
	if adversary.SessionID != "" {
		t.Fatalf("session id = %q, want empty", adversary.SessionID)
	}
	if adversary.Notes != "" {
		t.Fatalf("notes = %q, want empty", adversary.Notes)
	}
	if adversary.HP != 0 {
		t.Fatalf("hp = %d, want 0", adversary.HP)
	}
	if adversary.Stress != 0 {
		t.Fatalf("stress = %d, want 0", adversary.Stress)
	}
	if adversary.Evasion != 0 {
		t.Fatalf("evasion = %d, want 0", adversary.Evasion)
	}
	if adversary.Major != 0 {
		t.Fatalf("major = %d, want 0", adversary.Major)
	}
	if adversary.Severe != 0 {
		t.Fatalf("severe = %d, want 0", adversary.Severe)
	}
	if adversary.Armor != 0 {
		t.Fatalf("armor = %d, want 0", adversary.Armor)
	}
}

func TestFolderApplyHandlesAllRegisteredEvents(t *testing.T) {
	projector := NewFolder()
	for _, def := range daggerheartEventDefinitions {
		t.Run(string(def.Type), func(t *testing.T) {
			payloadJSON := []byte(`{}`)
			if def.Type == EventTypeGMFearChanged {
				payload, err := json.Marshal(GMFearChangedPayload{Before: 1, After: 2})
				if err != nil {
					t.Fatalf("marshal payload: %v", err)
				}
				payloadJSON = payload
			}

			updated, err := projector.Fold(SnapshotState{CampaignID: "camp-1", GMFear: 1}, event.Event{
				CampaignID:    "camp-1",
				Type:          def.Type,
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   payloadJSON,
			})
			if err != nil {
				t.Fatalf("projector apply %s: %v", def.Type, err)
			}
			assertTestSnapshotState(t, updated)
		})
	}
}

func TestFolderApply_RejectsAggregateState(t *testing.T) {
	// System folders should only receive their own state type, not the
	// full aggregate.State. The aggregate folder extracts the system-specific
	// state before calling RouteEvent.
	folder := NewFolder()
	aggState := aggregate.State{
		Systems: map[module.Key]any{
			{ID: SystemID, Version: SystemVersion}: SnapshotState{
				CampaignID: "camp-1",
				GMFear:     3,
			},
		},
	}
	_, err := folder.Fold(aggState, event.Event{
		CampaignID:    "camp-1",
		Type:          EventTypeGMFearChanged,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"before":3,"after":5}`),
	})
	if err == nil {
		t.Fatal("expected error when passing aggregate.State to projector")
	}
}

func TestFolderApplyUnknownEventReturnsError(t *testing.T) {
	projector := NewFolder()
	_, err := projector.Fold(SnapshotState{CampaignID: "camp-1"}, event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.daggerheart.unknown"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}
