package scene

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestFold_Created_SetsActiveAndMetadata(t *testing.T) {
	evt := event.Event{
		Type:        EventTypeCreated,
		PayloadJSON: []byte(`{"scene_id":"s1","name":"Cavern","description":"Dark","character_ids":["c1"]}`),
	}
	state, err := Fold(State{}, evt)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Open {
		t.Fatal("expected active = true")
	}
	if state.SceneID != "s1" {
		t.Fatalf("scene id = %q, want %q", state.SceneID, "s1")
	}
	if state.Name != "Cavern" {
		t.Fatalf("name = %q, want %q", state.Name, "Cavern")
	}
	if state.Description != "Dark" {
		t.Fatalf("description = %q, want %q", state.Description, "Dark")
	}
	if state.Characters == nil {
		t.Fatal("expected characters map to be initialized")
	}
}

func TestFold_Updated_UpdatesNameAndDescription(t *testing.T) {
	state := State{SceneID: "s1", Name: "Old", Description: "Old desc", Open: true}
	evt := event.Event{
		Type:        EventTypeUpdated,
		PayloadJSON: []byte(`{"scene_id":"s1","name":"New","description":"New desc"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if state.Name != "New" {
		t.Fatalf("name = %q, want %q", state.Name, "New")
	}
	if state.Description != "New desc" {
		t.Fatalf("description = %q, want %q", state.Description, "New desc")
	}
}

func TestFold_Updated_PartialUpdate_KeepsExistingFields(t *testing.T) {
	state := State{SceneID: "s1", Name: "Keep", Description: "Keep desc", Open: true}
	evt := event.Event{
		Type:        EventTypeUpdated,
		PayloadJSON: []byte(`{"scene_id":"s1","name":"Changed"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if state.Name != "Changed" {
		t.Fatalf("name = %q, want %q", state.Name, "Changed")
	}
	if state.Description != "Keep desc" {
		t.Fatalf("description = %q, want %q (should be preserved)", state.Description, "Keep desc")
	}
}

func TestFold_Ended_ClearsActiveAndTransientState(t *testing.T) {
	state := State{
		SceneID:              "s1",
		Open:                 true,
		GateOpen:             true,
		GateID:               "g1",
		SpotlightType:        "gm",
		SpotlightCharacterID: "c1",
	}
	evt := event.Event{
		Type:        EventTypeEnded,
		PayloadJSON: []byte(`{"scene_id":"s1"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if state.Open {
		t.Fatal("expected active = false")
	}
	if state.GateOpen {
		t.Fatal("expected gate_open = false after end")
	}
	if state.GateID != "" {
		t.Fatalf("gate_id = %q, want empty", state.GateID)
	}
	if state.SpotlightType != "" {
		t.Fatalf("spotlight_type = %q, want empty", state.SpotlightType)
	}
}

func TestFold_CharacterAdded_AddsToMap(t *testing.T) {
	state := State{Characters: map[ids.CharacterID]bool{"c1": true}}
	evt := event.Event{
		Type:        EventTypeCharacterAdded,
		PayloadJSON: []byte(`{"scene_id":"s1","character_id":"c2"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Characters["c2"] {
		t.Fatal("expected c2 in characters")
	}
	if !state.Characters["c1"] {
		t.Fatal("expected c1 still in characters")
	}
}

func TestFold_CharacterRemoved_RemovesFromMap(t *testing.T) {
	state := State{Characters: map[ids.CharacterID]bool{"c1": true, "c2": true}}
	evt := event.Event{
		Type:        EventTypeCharacterRemoved,
		PayloadJSON: []byte(`{"scene_id":"s1","character_id":"c1"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if state.Characters["c1"] {
		t.Fatal("expected c1 removed from characters")
	}
	if !state.Characters["c2"] {
		t.Fatal("expected c2 still in characters")
	}
}

func TestFold_GateOpened_SetsGateState(t *testing.T) {
	state := State{Open: true}
	evt := event.Event{
		Type:        EventTypeGateOpened,
		PayloadJSON: []byte(`{"scene_id":"s1","gate_id":"g1","gate_type":"decision"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if !state.GateOpen {
		t.Fatal("expected gate_open = true")
	}
	if state.GateID != "g1" {
		t.Fatalf("gate_id = %q, want %q", state.GateID, "g1")
	}
}

func TestFold_GateResolved_ClearsGateState(t *testing.T) {
	state := State{GateOpen: true, GateID: "g1"}
	evt := event.Event{
		Type:        EventTypeGateResolved,
		PayloadJSON: []byte(`{"scene_id":"s1","gate_id":"g1"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if state.GateOpen {
		t.Fatal("expected gate_open = false")
	}
	if state.GateID != "" {
		t.Fatalf("gate_id = %q, want empty", state.GateID)
	}
}

func TestFold_GateAbandoned_ClearsGateState(t *testing.T) {
	state := State{GateOpen: true, GateID: "g1"}
	evt := event.Event{
		Type:        EventTypeGateAbandoned,
		PayloadJSON: []byte(`{"scene_id":"s1","gate_id":"g1"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if state.GateOpen {
		t.Fatal("expected gate_open = false")
	}
}

func TestFold_SpotlightSet_SetsSpotlightState(t *testing.T) {
	state := State{Open: true}
	evt := event.Event{
		Type:        EventTypeSpotlightSet,
		PayloadJSON: []byte(`{"scene_id":"s1","spotlight_type":"character","character_id":"c1"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if state.SpotlightType != "character" {
		t.Fatalf("spotlight_type = %q, want %q", state.SpotlightType, "character")
	}
	if state.SpotlightCharacterID != "c1" {
		t.Fatalf("spotlight_character_id = %q, want %q", state.SpotlightCharacterID, "c1")
	}
}

func TestFold_SpotlightCleared_ClearsSpotlightState(t *testing.T) {
	state := State{SpotlightType: "gm", SpotlightCharacterID: "c1"}
	evt := event.Event{
		Type:        EventTypeSpotlightCleared,
		PayloadJSON: []byte(`{"scene_id":"s1"}`),
	}
	state, err := Fold(state, evt)
	if err != nil {
		t.Fatal(err)
	}
	if state.SpotlightType != "" {
		t.Fatalf("spotlight_type = %q, want empty", state.SpotlightType)
	}
	if state.SpotlightCharacterID != "" {
		t.Fatalf("spotlight_character_id = %q, want empty", state.SpotlightCharacterID)
	}
}

func TestFold_UnknownEventType_ReturnsError(t *testing.T) {
	original := State{
		SceneID: "s1",
		Open:    true,
		Name:    "Test",
	}
	_, err := Fold(original, event.Event{
		Type:        event.Type("scene.unknown"),
		PayloadJSON: []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected error for unknown event type, got nil")
	}
}

func TestFold_CorruptPayload_ReturnsError(t *testing.T) {
	corruptPayload := []byte(`{`)
	typesWithPayloads := []event.Type{
		EventTypeCreated,
		EventTypeUpdated,
		EventTypeCharacterAdded,
		EventTypeCharacterRemoved,
		EventTypeGateOpened,
		EventTypeSpotlightSet,
	}
	for _, evtType := range typesWithPayloads {
		_, err := Fold(State{}, event.Event{Type: evtType, PayloadJSON: corruptPayload})
		if err == nil {
			t.Fatalf("expected error for corrupt payload on %s", evtType)
		}
	}
}

func TestFold_PlayerPhasePostedClearsYieldedAndReviewState(t *testing.T) {
	t.Parallel()

	state := State{
		PlayerPhaseID: "phase-1",
		PlayerPhaseSlots: map[ids.ParticipantID]PlayerPhaseSlot{
			"p1": {
				ParticipantID:      "p1",
				Yielded:            true,
				ReviewStatus:       PlayerPhaseSlotReviewStatusChangesRequested,
				ReviewReason:       "Fix this.",
				ReviewCharacterIDs: []ids.CharacterID{"c1"},
			},
		},
	}
	payloadJSON, err := json.Marshal(PlayerPhasePostedPayload{
		ParticipantID: "p1",
		CharacterIDs:  []ids.CharacterID{"c1"},
		SummaryText:   "Corrected action.",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	state, err = Fold(state, event.Event{Type: EventTypePlayerPhasePosted, PayloadJSON: payloadJSON})
	if err != nil {
		t.Fatalf("fold: %v", err)
	}

	slot := state.PlayerPhaseSlots["p1"]
	if slot.Yielded {
		t.Fatal("slot yielded = true, want false after repost")
	}
	if slot.ReviewStatus != PlayerPhaseSlotReviewStatusOpen {
		t.Fatalf("review status = %q, want %q", slot.ReviewStatus, PlayerPhaseSlotReviewStatusOpen)
	}
	if slot.ReviewReason != "" {
		t.Fatalf("review reason = %q, want empty", slot.ReviewReason)
	}
	if len(slot.ReviewCharacterIDs) != 0 {
		t.Fatalf("review character ids = %#v, want empty", slot.ReviewCharacterIDs)
	}
}
