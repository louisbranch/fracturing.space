package daggerheart

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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
		Type:          eventTypeCharacterStatePatched,
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

func TestProjectorApplyHandlesAllRegisteredEvents(t *testing.T) {
	projector := Projector{}
	for _, def := range daggerheartEventDefinitions {
		t.Run(string(def.Type), func(t *testing.T) {
			payloadJSON := []byte(`{}`)
			if def.Type == eventTypeGMFearChanged {
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
