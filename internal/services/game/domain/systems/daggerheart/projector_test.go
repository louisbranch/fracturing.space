package daggerheart

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func TestProjectorApplyGMFearChanged_UpdatesState(t *testing.T) {
	projector := Projector{}
	state := SnapshotState{CampaignID: "camp-1", GMFear: 2}

	payload, err := json.Marshal(GMFearChangedPayload{Before: 2, After: 5, Reason: "shift"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	updated, err := projector.Apply(state, event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.daggerheart.gm_fear_changed"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot, ok := updated.(SnapshotState)
	if !ok {
		t.Fatalf("expected SnapshotState, got %T", updated)
	}
	if snapshot.GMFear != 5 {
		t.Fatalf("gm fear = %d, want %d", snapshot.GMFear, 5)
	}
	if snapshot.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want %s", snapshot.CampaignID, "camp-1")
	}
}

func TestProjectorApplyCharacterStatePatched_StoresCharacterState(t *testing.T) {
	projector := Projector{}
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

	updated, err := projector.Apply(SnapshotState{CampaignID: "camp-1"}, event.Event{
		CampaignID:    "camp-1",
		Type:          EventTypeCharacterStatePatched,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot, ok := updated.(SnapshotState)
	if !ok {
		t.Fatalf("expected SnapshotState, got %T", updated)
	}
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

func TestProjectorApplyCharacterStatePatched_DoesNotMutateFromBeforeOnly(t *testing.T) {
	projector := Projector{}
	hpBefore := 7
	payload, err := json.Marshal(CharacterStatePatchedPayload{
		CharacterID: "char-1",
		HPBefore:    &hpBefore,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := projector.Apply(SnapshotState{
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
	snapshot, ok := updated.(SnapshotState)
	if !ok {
		t.Fatalf("expected SnapshotState, got %T", updated)
	}
	character, ok := snapshot.CharacterStates["char-1"]
	if !ok {
		t.Fatal("expected character state")
	}
	if character.HP != 0 {
		t.Fatalf("hp = %d, want %d", character.HP, 0)
	}
}

func TestProjectorApplyAdversaryUpdated_AppliesZeroAndEmptyValues(t *testing.T) {
	projector := Projector{}
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

	updated, err := projector.Apply(state, event.Event{
		CampaignID:    "camp-1",
		Type:          EventTypeAdversaryUpdated,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payload,
	})
	if err != nil {
		t.Fatalf("apply event: %v", err)
	}
	snapshot, ok := updated.(SnapshotState)
	if !ok {
		t.Fatalf("expected SnapshotState, got %T", updated)
	}
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

func TestProjectorApplyHandlesAllRegisteredEvents(t *testing.T) {
	projector := Projector{}
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

			updated, err := projector.Apply(SnapshotState{CampaignID: "camp-1", GMFear: 1}, event.Event{
				CampaignID:    "camp-1",
				Type:          def.Type,
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   payloadJSON,
			})
			if err != nil {
				t.Fatalf("projector apply %s: %v", def.Type, err)
			}
			if _, ok := updated.(SnapshotState); !ok {
				t.Fatalf("expected SnapshotState, got %T", updated)
			}
		})
	}
}

func TestProjectorApply_RejectsAggregateState(t *testing.T) {
	// System projectors should only receive their own state type, not the
	// full aggregate.State. The aggregate applier extracts the system-specific
	// state before calling RouteEvent.
	projector := Projector{}
	aggState := aggregate.State{
		Systems: map[module.Key]any{
			{ID: SystemID, Version: SystemVersion}: SnapshotState{
				CampaignID: "camp-1",
				GMFear:     3,
			},
		},
	}
	_, err := projector.Apply(aggState, event.Event{
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

func TestProjectorApplyUnknownEventReturnsError(t *testing.T) {
	projector := Projector{}
	_, err := projector.Apply(SnapshotState{CampaignID: "camp-1"}, event.Event{
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
