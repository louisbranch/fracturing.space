package scene

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var fixedNow = time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)

func nowFunc() time.Time { return fixedNow }

func activeScene(id string, characters ...string) State {
	chars := make(map[string]bool, len(characters))
	for _, c := range characters {
		chars[c] = true
	}
	return State{
		SceneID:    id,
		Active:     true,
		Characters: chars,
	}
}

func scenesMap(scenes ...State) map[string]State {
	m := make(map[string]State, len(scenes))
	for _, s := range scenes {
		m[s.SceneID] = s
	}
	return m
}

func cmd(t command.Type, payloadJSON string) command.Command {
	return command.Command{
		CampaignID:  "camp-1",
		Type:        t,
		ActorType:   command.ActorTypeGM,
		ActorID:     "gm-1",
		SessionID:   "sess-1",
		PayloadJSON: []byte(payloadJSON),
	}
}

func requireAccepted(t *testing.T, d command.Decision, eventCount int) {
	t.Helper()
	if len(d.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d: %v", len(d.Rejections), d.Rejections)
	}
	if len(d.Events) != eventCount {
		t.Fatalf("expected %d events, got %d", eventCount, len(d.Events))
	}
}

func requireRejected(t *testing.T, d command.Decision, code string) {
	t.Helper()
	if len(d.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(d.Events))
	}
	if len(d.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(d.Rejections))
	}
	if d.Rejections[0].Code != code {
		t.Fatalf("rejection code = %q, want %q", d.Rejections[0].Code, code)
	}
}

// --- Create ---

func TestDecideCreate_EmitsCreatedAndCharacterAddedEvents(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCreate, `{
		"scene_id": "scene-1",
		"name": " The Dark Cavern ",
		"description": " A damp cave ",
		"character_ids": ["char-1", "char-2"]
	}`), nowFunc)

	// 1 created + 2 character_added
	requireAccepted(t, d, 3)

	if d.Events[0].Type != EventTypeCreated {
		t.Fatalf("event[0] type = %s, want %s", d.Events[0].Type, EventTypeCreated)
	}
	if d.Events[0].EntityType != "scene" {
		t.Fatalf("entity type = %s, want scene", d.Events[0].EntityType)
	}
	if d.Events[0].EntityID != "scene-1" {
		t.Fatalf("entity id = %s, want scene-1", d.Events[0].EntityID)
	}

	var created CreatePayload
	if err := json.Unmarshal(d.Events[0].PayloadJSON, &created); err != nil {
		t.Fatal(err)
	}
	if created.Name != "The Dark Cavern" {
		t.Fatalf("name = %q, want %q", created.Name, "The Dark Cavern")
	}
	if created.Description != "A damp cave" {
		t.Fatalf("description = %q, want %q", created.Description, "A damp cave")
	}

	for i, evtIdx := range []int{1, 2} {
		if d.Events[evtIdx].Type != EventTypeCharacterAdded {
			t.Fatalf("event[%d] type = %s, want %s", evtIdx, d.Events[evtIdx].Type, EventTypeCharacterAdded)
		}
		var added CharacterAddedPayload
		if err := json.Unmarshal(d.Events[evtIdx].PayloadJSON, &added); err != nil {
			t.Fatal(err)
		}
		wantCharID := []string{"char-1", "char-2"}[i]
		if added.CharacterID != wantCharID {
			t.Fatalf("event[%d] character_id = %q, want %q", evtIdx, added.CharacterID, wantCharID)
		}
	}
}

func TestDecideCreate_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCreate, `{"scene_id":"","name":"x","character_ids":["c1"]}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

func TestDecideCreate_MissingName_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCreate, `{"scene_id":"s1","name":"  ","character_ids":["c1"]}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneNameRequired)
}

func TestDecideCreate_NoCharacters_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCreate, `{"scene_id":"s1","name":"x","character_ids":[]}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneCharactersRequired)
}

func TestDecideCreate_DeduplicatesCharacters(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCreate, `{"scene_id":"s1","name":"x","character_ids":["c1","c1","c2"]}`), nowFunc)
	// 1 created + 2 unique character_added (not 3)
	requireAccepted(t, d, 3)
}

// --- Update ---

func TestDecideUpdate_EmitsUpdatedEvent(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeUpdate, `{"scene_id":"s1","name":"New Name"}`), nowFunc)
	requireAccepted(t, d, 1)
	if d.Events[0].Type != EventTypeUpdated {
		t.Fatalf("event type = %s, want %s", d.Events[0].Type, EventTypeUpdated)
	}
}

func TestDecideUpdate_SceneNotActive_Rejects(t *testing.T) {
	scenes := scenesMap(State{SceneID: "s1", Active: false})
	d := Decide(scenes, cmd(CommandTypeUpdate, `{"scene_id":"s1","name":"x"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneNotActive)
}

func TestDecideUpdate_SceneNotFound_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeUpdate, `{"scene_id":"missing","name":"x"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneNotFound)
}

// --- End ---

func TestDecideEnd_EmitsEndedEvent(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeEnd, `{"scene_id":"s1","reason":"done"}`), nowFunc)
	requireAccepted(t, d, 1)
	if d.Events[0].Type != EventTypeEnded {
		t.Fatalf("event type = %s, want %s", d.Events[0].Type, EventTypeEnded)
	}
}

func TestDecideEnd_SceneNotActive_Rejects(t *testing.T) {
	scenes := scenesMap(State{SceneID: "s1", Active: false})
	d := Decide(scenes, cmd(CommandTypeEnd, `{"scene_id":"s1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneNotActive)
}

// --- Character Add ---

func TestDecideCharacterAdd_EmitsCharacterAddedEvent(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	d := Decide(scenes, cmd(CommandTypeCharacterAdd, `{"scene_id":"s1","character_id":"c2"}`), nowFunc)
	requireAccepted(t, d, 1)
	if d.Events[0].Type != EventTypeCharacterAdded {
		t.Fatalf("event type = %s, want %s", d.Events[0].Type, EventTypeCharacterAdded)
	}
}

func TestDecideCharacterAdd_AlreadyPresent_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	d := Decide(scenes, cmd(CommandTypeCharacterAdd, `{"scene_id":"s1","character_id":"c1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeCharacterAlreadyInScene)
}

func TestDecideCharacterAdd_MissingCharacterID_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeCharacterAdd, `{"scene_id":"s1","character_id":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeCharacterIDRequired)
}

// --- Character Remove ---

func TestDecideCharacterRemove_EmitsCharacterRemovedEvent(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1", "c2"))
	d := Decide(scenes, cmd(CommandTypeCharacterRemove, `{"scene_id":"s1","character_id":"c1"}`), nowFunc)
	requireAccepted(t, d, 1)
	if d.Events[0].Type != EventTypeCharacterRemoved {
		t.Fatalf("event type = %s, want %s", d.Events[0].Type, EventTypeCharacterRemoved)
	}
}

func TestDecideCharacterRemove_NotInScene_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	d := Decide(scenes, cmd(CommandTypeCharacterRemove, `{"scene_id":"s1","character_id":"c2"}`), nowFunc)
	requireRejected(t, d, rejectionCodeCharacterNotInScene)
}

// --- Character Transfer ---

func TestDecideCharacterTransfer_EmitsRemoveAndAddEvents(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"), activeScene("s2", "c2"))
	d := Decide(scenes, cmd(CommandTypeCharacterTransfer, `{
		"source_scene_id": "s1",
		"target_scene_id": "s2",
		"character_id": "c1"
	}`), nowFunc)
	requireAccepted(t, d, 2)
	if d.Events[0].Type != EventTypeCharacterRemoved {
		t.Fatalf("event[0] type = %s, want %s", d.Events[0].Type, EventTypeCharacterRemoved)
	}
	if d.Events[1].Type != EventTypeCharacterAdded {
		t.Fatalf("event[1] type = %s, want %s", d.Events[1].Type, EventTypeCharacterAdded)
	}
}

func TestDecideCharacterTransfer_SourceNotActive_Rejects(t *testing.T) {
	scenes := scenesMap(State{SceneID: "s1"}, activeScene("s2"))
	d := Decide(scenes, cmd(CommandTypeCharacterTransfer, `{"source_scene_id":"s1","target_scene_id":"s2","character_id":"c1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneNotActive)
}

func TestDecideCharacterTransfer_TargetNotActive_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"), State{SceneID: "s2"})
	d := Decide(scenes, cmd(CommandTypeCharacterTransfer, `{"source_scene_id":"s1","target_scene_id":"s2","character_id":"c1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneNotActive)
}

func TestDecideCharacterTransfer_CharacterNotInSource_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"), activeScene("s2"))
	d := Decide(scenes, cmd(CommandTypeCharacterTransfer, `{"source_scene_id":"s1","target_scene_id":"s2","character_id":"missing"}`), nowFunc)
	requireRejected(t, d, rejectionCodeCharacterNotInScene)
}

// --- Transition ---

func TestDecideTransition_EmitsCreatedCharacterAddedAndEndedEvents(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1", "c2"))
	d := Decide(scenes, cmd(CommandTypeTransition, `{
		"source_scene_id": "s1",
		"new_scene_id": "s2",
		"name": "The Next Room",
		"description": "A bright hallway"
	}`), nowFunc)

	// 1 created + 2 character_added + 1 ended
	requireAccepted(t, d, 4)

	if d.Events[0].Type != EventTypeCreated {
		t.Fatalf("event[0] type = %s, want %s", d.Events[0].Type, EventTypeCreated)
	}
	// Characters are sorted, so c1 then c2.
	if d.Events[1].Type != EventTypeCharacterAdded {
		t.Fatalf("event[1] type = %s, want %s", d.Events[1].Type, EventTypeCharacterAdded)
	}
	if d.Events[2].Type != EventTypeCharacterAdded {
		t.Fatalf("event[2] type = %s, want %s", d.Events[2].Type, EventTypeCharacterAdded)
	}
	lastEvt := d.Events[len(d.Events)-1]
	if lastEvt.Type != EventTypeEnded {
		t.Fatalf("last event type = %s, want %s", lastEvt.Type, EventTypeEnded)
	}
	if lastEvt.EntityID != "s1" {
		t.Fatalf("ended event entity id = %s, want s1", lastEvt.EntityID)
	}
}

func TestDecideTransition_MissingNewSceneID_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	d := Decide(scenes, cmd(CommandTypeTransition, `{"source_scene_id":"s1","new_scene_id":"","name":"x"}`), nowFunc)
	requireRejected(t, d, rejectionCodeNewSceneIDRequired)
}

func TestDecideTransition_SourceNotActive_Rejects(t *testing.T) {
	scenes := scenesMap(State{SceneID: "s1"})
	d := Decide(scenes, cmd(CommandTypeTransition, `{"source_scene_id":"s1","new_scene_id":"s2","name":"x"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneNotActive)
}

// --- Gate Open ---

func TestDecideGateOpen_EmitsGateOpenedEvent(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeGateOpen, `{
		"scene_id": "s1",
		"gate_id": "gate-1",
		"gate_type": "decision",
		"reason": "Choose your path"
	}`), nowFunc)
	requireAccepted(t, d, 1)
	if d.Events[0].Type != EventTypeGateOpened {
		t.Fatalf("event type = %s, want %s", d.Events[0].Type, EventTypeGateOpened)
	}
	if d.Events[0].EntityType != "scene_gate" {
		t.Fatalf("entity type = %s, want scene_gate", d.Events[0].EntityType)
	}
}

func TestDecideGateOpen_AlreadyOpen_Rejects(t *testing.T) {
	s := activeScene("s1")
	s.GateOpen = true
	s.GateID = "existing"
	scenes := scenesMap(s)
	d := Decide(scenes, cmd(CommandTypeGateOpen, `{"scene_id":"s1","gate_id":"gate-2","gate_type":"decision"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneGateAlreadyOpen)
}

func TestDecideGateOpen_NormalizesGateType(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeGateOpen, `{
		"scene_id": "s1",
		"gate_id": "gate-1",
		"gate_type": " Decision "
	}`), nowFunc)
	requireAccepted(t, d, 1)
	var payload GateOpenedPayload
	if err := json.Unmarshal(d.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.GateType != "decision" {
		t.Fatalf("gate_type = %q, want %q", payload.GateType, "decision")
	}
}

func TestDecideGateOpen_MissingGateID_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeGateOpen, `{"scene_id":"s1","gate_id":"","gate_type":"decision"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneGateIDRequired)
}

// --- Gate Resolve ---

func TestDecideGateResolve_EmitsGateResolvedEvent(t *testing.T) {
	s := activeScene("s1")
	s.GateOpen = true
	s.GateID = "gate-1"
	scenes := scenesMap(s)
	d := Decide(scenes, cmd(CommandTypeGateResolve, `{"scene_id":"s1","gate_id":"gate-1","decision":"proceed"}`), nowFunc)
	requireAccepted(t, d, 1)
	if d.Events[0].Type != EventTypeGateResolved {
		t.Fatalf("event type = %s, want %s", d.Events[0].Type, EventTypeGateResolved)
	}
}

func TestDecideGateResolve_GateNotOpen_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeGateResolve, `{"scene_id":"s1","gate_id":"gate-1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneGateNotOpen)
}

// --- Gate Abandon ---

func TestDecideGateAbandon_EmitsGateAbandonedEvent(t *testing.T) {
	s := activeScene("s1")
	s.GateOpen = true
	s.GateID = "gate-1"
	scenes := scenesMap(s)
	d := Decide(scenes, cmd(CommandTypeGateAbandon, `{"scene_id":"s1","gate_id":"gate-1","reason":"timeout"}`), nowFunc)
	requireAccepted(t, d, 1)
	if d.Events[0].Type != EventTypeGateAbandoned {
		t.Fatalf("event type = %s, want %s", d.Events[0].Type, EventTypeGateAbandoned)
	}
}

func TestDecideGateAbandon_GateNotOpen_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeGateAbandon, `{"scene_id":"s1","gate_id":"gate-1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneGateNotOpen)
}

// --- Spotlight Set ---

func TestDecideSpotlightSet_EmitsSpotlightSetEvent(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	d := Decide(scenes, cmd(CommandTypeSpotlightSet, `{"scene_id":"s1","spotlight_type":"character","character_id":"c1"}`), nowFunc)
	requireAccepted(t, d, 1)
	if d.Events[0].Type != EventTypeSpotlightSet {
		t.Fatalf("event type = %s, want %s", d.Events[0].Type, EventTypeSpotlightSet)
	}
}

func TestDecideSpotlightSet_CharacterNotInScene_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	d := Decide(scenes, cmd(CommandTypeSpotlightSet, `{"scene_id":"s1","spotlight_type":"character","character_id":"c2"}`), nowFunc)
	requireRejected(t, d, rejectionCodeCharacterNotInScene)
}

func TestDecideSpotlightSet_CharacterTypeWithoutCharacterID_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	d := Decide(scenes, cmd(CommandTypeSpotlightSet, `{"scene_id":"s1","spotlight_type":"character","character_id":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeCharacterIDRequired)
}

func TestDecideSpotlightSet_GMType_Accepted(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeSpotlightSet, `{"scene_id":"s1","spotlight_type":"gm"}`), nowFunc)
	requireAccepted(t, d, 1)
}

func TestDecideSpotlightSet_MissingType_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeSpotlightSet, `{"scene_id":"s1","spotlight_type":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeSpotlightTypeRequired)
}

// --- Spotlight Clear ---

func TestDecideSpotlightClear_EmitsSpotlightClearedEvent(t *testing.T) {
	s := activeScene("s1")
	s.SpotlightType = "gm"
	scenes := scenesMap(s)
	d := Decide(scenes, cmd(CommandTypeSpotlightClear, `{"scene_id":"s1"}`), nowFunc)
	requireAccepted(t, d, 1)
	if d.Events[0].Type != EventTypeSpotlightCleared {
		t.Fatalf("event type = %s, want %s", d.Events[0].Type, EventTypeSpotlightCleared)
	}
}

func TestDecideSpotlightClear_NotSet_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeSpotlightClear, `{"scene_id":"s1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSpotlightNotSet)
}

// --- Unsupported Command ---

func TestDecide_UnsupportedCommandType_Rejects(t *testing.T) {
	d := Decide(nil, cmd("scene.unknown", `{}`), nowFunc)
	requireRejected(t, d, command.RejectionCodeCommandTypeUnsupported)
}

// --- Payload decode errors (all 12 command types) ---

func TestDecide_PayloadDecodeErrors(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	invalid := `{bad json`

	types := []command.Type{
		CommandTypeCreate,
		CommandTypeUpdate,
		CommandTypeEnd,
		CommandTypeCharacterAdd,
		CommandTypeCharacterRemove,
		CommandTypeCharacterTransfer,
		CommandTypeTransition,
		CommandTypeGateOpen,
		CommandTypeGateResolve,
		CommandTypeGateAbandon,
		CommandTypeSpotlightSet,
		CommandTypeSpotlightClear,
	}
	for _, ct := range types {
		t.Run(string(ct), func(t *testing.T) {
			d := Decide(scenes, cmd(ct, invalid), nowFunc)
			requireRejected(t, d, command.RejectionCodePayloadDecodeFailed)
		})
	}
}

// --- Missing field validation branches ---

func TestDecide_NilNowFuncDefaultsToTimeNow(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCreate, `{"scene_id":"s1","name":"x","character_ids":["c1"]}`), nil)
	requireAccepted(t, d, 2)
}

func TestDecideUpdate_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeUpdate, `{"scene_id":"","name":"x"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

func TestDecideCharacterAdd_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCharacterAdd, `{"scene_id":"","character_id":"c1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

func TestDecideTransition_MissingSourceSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeTransition, `{"source_scene_id":"","new_scene_id":"s2","name":"x"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSourceSceneIDRequired)
}

func TestDecideTransition_MissingName_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	d := Decide(scenes, cmd(CommandTypeTransition, `{"source_scene_id":"s1","new_scene_id":"s2","name":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneNameRequired)
}

func TestDecideEnd_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeEnd, `{"scene_id":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

func TestDecideCharacterRemove_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCharacterRemove, `{"scene_id":"","character_id":"c1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

func TestDecideCharacterRemove_MissingCharacterID_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1", "c1"))
	d := Decide(scenes, cmd(CommandTypeCharacterRemove, `{"scene_id":"s1","character_id":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeCharacterIDRequired)
}

func TestDecideCharacterTransfer_MissingSourceSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCharacterTransfer, `{"source_scene_id":"","target_scene_id":"s2","character_id":"c1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSourceSceneIDRequired)
}

func TestDecideCharacterTransfer_MissingTargetSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCharacterTransfer, `{"source_scene_id":"s1","target_scene_id":"","character_id":"c1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeTargetSceneIDRequired)
}

func TestDecideCharacterTransfer_MissingCharacterID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeCharacterTransfer, `{"source_scene_id":"s1","target_scene_id":"s2","character_id":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeCharacterIDRequired)
}

func TestDecideGateOpen_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeGateOpen, `{"scene_id":"","gate_id":"g1","gate_type":"decision"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

func TestDecideGateOpen_MissingGateType_Rejects(t *testing.T) {
	scenes := scenesMap(activeScene("s1"))
	d := Decide(scenes, cmd(CommandTypeGateOpen, `{"scene_id":"s1","gate_id":"g1","gate_type":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneGateTypeRequired)
}

func TestDecideGateResolve_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeGateResolve, `{"scene_id":"","gate_id":"g1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

func TestDecideGateResolve_MissingGateID_Rejects(t *testing.T) {
	s := activeScene("s1")
	s.GateOpen = true
	scenes := scenesMap(s)
	d := Decide(scenes, cmd(CommandTypeGateResolve, `{"scene_id":"s1","gate_id":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneGateIDRequired)
}

func TestDecideGateAbandon_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeGateAbandon, `{"scene_id":"","gate_id":"g1"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

func TestDecideGateAbandon_MissingGateID_Rejects(t *testing.T) {
	s := activeScene("s1")
	s.GateOpen = true
	scenes := scenesMap(s)
	d := Decide(scenes, cmd(CommandTypeGateAbandon, `{"scene_id":"s1","gate_id":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneGateIDRequired)
}

func TestDecideSpotlightSet_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeSpotlightSet, `{"scene_id":"","spotlight_type":"gm"}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

func TestDecideSpotlightClear_MissingSceneID_Rejects(t *testing.T) {
	d := Decide(nil, cmd(CommandTypeSpotlightClear, `{"scene_id":""}`), nowFunc)
	requireRejected(t, d, rejectionCodeSceneIDRequired)
}

// --- Envelope fields ---

func TestDecideCreate_CopiesEnvelopeFields(t *testing.T) {
	c := command.Command{
		CampaignID:  "camp-1",
		Type:        CommandTypeCreate,
		ActorType:   command.ActorTypeGM,
		ActorID:     "gm-1",
		SessionID:   "sess-1",
		RequestID:   "req-1",
		PayloadJSON: []byte(`{"scene_id":"s1","name":"Test","character_ids":["c1"]}`),
	}
	d := Decide(nil, c, nowFunc)
	requireAccepted(t, d, 2)

	evt := d.Events[0]
	if evt.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want camp-1", evt.CampaignID)
	}
	if evt.ActorType != event.ActorTypeGM {
		t.Fatalf("actor type = %s, want gm", evt.ActorType)
	}
	if evt.SessionID != "sess-1" {
		t.Fatalf("session id = %s, want sess-1", evt.SessionID)
	}
	if !evt.Timestamp.Equal(fixedNow) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, fixedNow)
	}
}
