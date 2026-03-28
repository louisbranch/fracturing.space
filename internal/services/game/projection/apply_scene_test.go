package projection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func sceneEvent(typ event.Type, campaignID, sessionID, sceneID string, payloadJSON []byte, ts time.Time) event.Event {
	return event.Event{
		CampaignID:  ids.CampaignID(campaignID),
		Type:        typ,
		SessionID:   ids.SessionID(sessionID),
		SceneID:     ids.SceneID(sceneID),
		Timestamp:   ts,
		ActorType:   event.ActorTypeSystem,
		PayloadJSON: payloadJSON,
	}
}

var sceneStamp = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// --- scene.created ---

func TestApplySceneCreated(t *testing.T) {
	ctx := context.Background()
	sceneStore := newFakeSceneStore()
	charStore := newFakeSceneCharacterStore()
	applier := Applier{Scene: sceneStore, SceneCharacter: charStore, SceneInteraction: newFakeSceneInteractionStore()}

	payload := scene.CreatePayload{SceneID: "sc-1", Name: "Battle", Description: "Fierce", CharacterIDs: []ids.CharacterID{"char-1", "char-2"}}
	data, _ := json.Marshal(payload)
	evt := sceneEvent(scene.EventTypeCreated, "camp-1", "sess-1", "sc-1", data, sceneStamp)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	rec, err := sceneStore.GetScene(ctx, "camp-1", "sc-1")
	if err != nil {
		t.Fatalf("get scene: %v", err)
	}
	if rec.Name != "Battle" {
		t.Errorf("name = %q, want %q", rec.Name, "Battle")
	}
	if rec.Description != "Fierce" {
		t.Errorf("description = %q, want %q", rec.Description, "Fierce")
	}
	if !rec.Open {
		t.Error("expected active")
	}
	if rec.SessionID != "sess-1" {
		t.Errorf("session_id = %q, want %q", rec.SessionID, "sess-1")
	}
	chars, err := charStore.ListSceneCharacters(ctx, "camp-1", "sc-1")
	if err != nil {
		t.Fatalf("list chars: %v", err)
	}
	if len(chars) != 2 {
		t.Fatalf("char count = %d, want 2", len(chars))
	}
}

func TestApplySceneCreated_MissingStore(t *testing.T) {
	data, _ := json.Marshal(scene.CreatePayload{SceneID: "sc-1", Name: "X"})
	evt := sceneEvent(scene.EventTypeCreated, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := (Applier{}).Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing scene store")
	}
}

func TestApplySceneCreated_MissingSessionID(t *testing.T) {
	sceneStore := newFakeSceneStore()
	charStore := newFakeSceneCharacterStore()
	applier := Applier{Scene: sceneStore, SceneCharacter: charStore, SceneInteraction: newFakeSceneInteractionStore()}

	data, _ := json.Marshal(scene.CreatePayload{SceneID: "sc-1", Name: "X"})
	evt := sceneEvent(scene.EventTypeCreated, "camp-1", "", "sc-1", data, sceneStamp)

	if err := applier.Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestApplySceneCreated_SkipsEmptyCharacterIDs(t *testing.T) {
	ctx := context.Background()
	sceneStore := newFakeSceneStore()
	charStore := newFakeSceneCharacterStore()
	applier := Applier{Scene: sceneStore, SceneCharacter: charStore, SceneInteraction: newFakeSceneInteractionStore()}

	payload := scene.CreatePayload{SceneID: "sc-1", Name: "X", CharacterIDs: []ids.CharacterID{"char-1", "", " "}}
	data, _ := json.Marshal(payload)
	evt := sceneEvent(scene.EventTypeCreated, "camp-1", "sess-1", "sc-1", data, sceneStamp)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	chars, _ := charStore.ListSceneCharacters(ctx, "camp-1", "sc-1")
	if len(chars) != 1 {
		t.Fatalf("char count = %d, want 1 (empty IDs skipped)", len(chars))
	}
}

// --- scene.updated ---

func TestApplySceneUpdated(t *testing.T) {
	ctx := context.Background()
	sceneStore := newFakeSceneStore()
	sceneStore.scenes["camp-1:sc-1"] = storage.SceneRecord{
		CampaignID: "camp-1", SceneID: "sc-1", SessionID: "sess-1",
		Name: "Old", Description: "Old desc", Open: true,
		CreatedAt: sceneStamp, UpdatedAt: sceneStamp,
	}
	applier := Applier{Scene: sceneStore}

	payload := scene.UpdatePayload{SceneID: "sc-1", Name: "New", Description: "New desc"}
	data, _ := json.Marshal(payload)
	later := sceneStamp.Add(time.Hour)
	evt := sceneEvent(scene.EventTypeUpdated, "camp-1", "sess-1", "sc-1", data, later)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	rec, _ := sceneStore.GetScene(ctx, "camp-1", "sc-1")
	if rec.Name != "New" {
		t.Errorf("name = %q, want %q", rec.Name, "New")
	}
	if rec.Description != "New desc" {
		t.Errorf("description = %q, want %q", rec.Description, "New desc")
	}
	if !rec.UpdatedAt.Equal(later) {
		t.Errorf("updated_at = %v, want %v", rec.UpdatedAt, later)
	}
}

func TestApplySceneUpdated_MissingStore(t *testing.T) {
	data, _ := json.Marshal(scene.UpdatePayload{SceneID: "sc-1", Name: "X"})
	evt := sceneEvent(scene.EventTypeUpdated, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := (Applier{}).Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing scene store")
	}
}

// --- scene.ended ---

func TestApplySceneEnded(t *testing.T) {
	ctx := context.Background()
	sceneStore := newFakeSceneStore()
	sceneStore.scenes["camp-1:sc-1"] = storage.SceneRecord{
		CampaignID: "camp-1", SceneID: "sc-1", Open: true,
		CreatedAt: sceneStamp, UpdatedAt: sceneStamp,
	}
	spotlightStore := newFakeSceneSpotlightStore()
	sceneInteractionStore := newFakeSceneInteractionStore()
	sceneInteractionStore.interactions["camp-1:sc-1"] = storage.SceneInteraction{
		CampaignID:           "camp-1",
		SceneID:              "sc-1",
		SessionID:            "sess-1",
		PhaseOpen:            true,
		PhaseID:              "phase-1",
		PhaseStatus:          scene.PlayerPhaseStatusPlayers,
		ActingCharacterIDs:   []string{"char-1"},
		ActingParticipantIDs: []string{"p-1"},
		Slots: []storage.ScenePlayerSlot{{
			ParticipantID: "p-1",
			CharacterIDs:  []string{"char-1"},
		}},
	}
	applier := Applier{Scene: sceneStore, SceneSpotlight: spotlightStore, SceneInteraction: sceneInteractionStore}

	data, _ := json.Marshal(scene.EndPayload{SceneID: "sc-1"})
	later := sceneStamp.Add(time.Hour)
	evt := sceneEvent(scene.EventTypeEnded, "camp-1", "sess-1", "sc-1", data, later)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	rec, _ := sceneStore.GetScene(ctx, "camp-1", "sc-1")
	if rec.Open {
		t.Error("expected inactive")
	}
	if rec.EndedAt == nil {
		t.Fatal("expected ended_at")
	}
	gotInteraction, err := sceneInteractionStore.GetSceneInteraction(ctx, "camp-1", "sc-1")
	if err != nil {
		t.Fatalf("get scene interaction: %v", err)
	}
	if gotInteraction.PhaseOpen || gotInteraction.PhaseID != "" || gotInteraction.PhaseStatus != "" {
		t.Fatalf("interaction phase state = %#v, want cleared", gotInteraction)
	}
	if len(gotInteraction.ActingCharacterIDs) != 0 || len(gotInteraction.ActingParticipantIDs) != 0 || len(gotInteraction.Slots) != 0 {
		t.Fatalf("interaction actors/slots = %#v, want cleared", gotInteraction)
	}
}

func TestApplySceneEnded_MissingStore(t *testing.T) {
	data, _ := json.Marshal(scene.EndPayload{SceneID: "sc-1"})
	evt := sceneEvent(scene.EventTypeEnded, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := (Applier{}).Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing scene store")
	}
}

func TestApplySceneEnded_MissingSceneInteractionFails(t *testing.T) {
	ctx := context.Background()
	sceneStore := newFakeSceneStore()
	sceneStore.scenes["camp-1:sc-1"] = storage.SceneRecord{
		CampaignID: "camp-1", SceneID: "sc-1", Open: true,
		CreatedAt: sceneStamp, UpdatedAt: sceneStamp,
	}
	applier := Applier{
		Scene:            sceneStore,
		SceneSpotlight:   newFakeSceneSpotlightStore(),
		SceneInteraction: newFakeSceneInteractionStore(),
	}

	data, _ := json.Marshal(scene.EndPayload{SceneID: "sc-1"})
	evt := sceneEvent(scene.EventTypeEnded, "camp-1", "sess-1", "sc-1", data, sceneStamp.Add(time.Hour))

	err := applier.Apply(ctx, evt)
	if err == nil {
		t.Fatal("expected error for missing scene interaction state")
	}
	if got := err.Error(); got != "get scene interaction on end: record not found" {
		t.Fatalf("error = %q, want missing interaction failure", got)
	}
}

// --- scene.character_added ---

func TestApplySceneCharacterAdded(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeSceneCharacterStore()
	applier := Applier{SceneCharacter: charStore}

	data, _ := json.Marshal(scene.CharacterAddedPayload{SceneID: "sc-1", CharacterID: "char-1"})
	evt := sceneEvent(scene.EventTypeCharacterAdded, "camp-1", "sess-1", "sc-1", data, sceneStamp)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	chars, _ := charStore.ListSceneCharacters(ctx, "camp-1", "sc-1")
	if len(chars) != 1 || chars[0].CharacterID != "char-1" {
		t.Fatalf("chars = %v, want [{char-1}]", chars)
	}
}

func TestApplySceneCharacterAdded_MissingCharacterID(t *testing.T) {
	charStore := newFakeSceneCharacterStore()
	applier := Applier{SceneCharacter: charStore}

	data, _ := json.Marshal(scene.CharacterAddedPayload{SceneID: "sc-1", CharacterID: ""})
	evt := sceneEvent(scene.EventTypeCharacterAdded, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := applier.Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing character ID")
	}
}

func TestApplySceneCharacterAdded_MissingStore(t *testing.T) {
	data, _ := json.Marshal(scene.CharacterAddedPayload{SceneID: "sc-1", CharacterID: "char-1"})
	evt := sceneEvent(scene.EventTypeCharacterAdded, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := (Applier{}).Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing scene character store")
	}
}

// --- scene.character_removed ---

func TestApplySceneCharacterRemoved(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeSceneCharacterStore()
	charStore.characters["camp-1:sc-1"] = []storage.SceneCharacterRecord{
		{CampaignID: "camp-1", SceneID: "sc-1", CharacterID: "char-1"},
	}
	applier := Applier{SceneCharacter: charStore}

	data, _ := json.Marshal(scene.CharacterRemovedPayload{SceneID: "sc-1", CharacterID: "char-1"})
	evt := sceneEvent(scene.EventTypeCharacterRemoved, "camp-1", "sess-1", "sc-1", data, sceneStamp)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	chars, _ := charStore.ListSceneCharacters(ctx, "camp-1", "sc-1")
	if len(chars) != 0 {
		t.Fatalf("chars = %v, want empty", chars)
	}
}

func TestApplySceneCharacterRemoved_MissingCharacterID(t *testing.T) {
	charStore := newFakeSceneCharacterStore()
	applier := Applier{SceneCharacter: charStore}

	data, _ := json.Marshal(scene.CharacterRemovedPayload{SceneID: "sc-1", CharacterID: ""})
	evt := sceneEvent(scene.EventTypeCharacterRemoved, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := applier.Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing character ID")
	}
}

// --- scene.gate_opened ---

func TestApplySceneGateOpened(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSceneGateStore()
	applier := Applier{SceneGate: gateStore}

	payload := scene.GateOpenedPayload{SceneID: "sc-1", GateID: "gate-1", GateType: "gm_consequence", Reason: "test"}
	data, _ := json.Marshal(payload)
	evt := sceneEvent(scene.EventTypeGateOpened, "camp-1", "sess-1", "sc-1", data, sceneStamp)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, err := gateStore.GetSceneGate(ctx, "camp-1", "sc-1", "gate-1")
	if err != nil {
		t.Fatalf("get gate: %v", err)
	}
	if gate.Status != session.GateStatusOpen {
		t.Errorf("status = %q, want %q", gate.Status, session.GateStatusOpen)
	}
	if gate.GateType != "gm_consequence" {
		t.Errorf("gate_type = %q, want %q", gate.GateType, "gm_consequence")
	}
	if gate.Reason != "test" {
		t.Errorf("reason = %q, want %q", gate.Reason, "test")
	}
}

func TestApplySceneGateOpened_MissingGateID(t *testing.T) {
	gateStore := newFakeSceneGateStore()
	applier := Applier{SceneGate: gateStore}

	data, _ := json.Marshal(scene.GateOpenedPayload{SceneID: "sc-1", GateID: "", GateType: "gm_consequence"})
	evt := sceneEvent(scene.EventTypeGateOpened, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := applier.Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing gate ID")
	}
}

func TestApplySceneGateOpened_MissingStore(t *testing.T) {
	data, _ := json.Marshal(scene.GateOpenedPayload{SceneID: "sc-1", GateID: "gate-1", GateType: "gm_consequence"})
	evt := sceneEvent(scene.EventTypeGateOpened, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := (Applier{}).Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing scene gate store")
	}
}

// --- scene.gate_resolved ---

func TestApplySceneGateResolved(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSceneGateStore()
	gateStore.gates["camp-1:sc-1:gate-1"] = storage.SceneGate{
		CampaignID: "camp-1", SceneID: "sc-1", GateID: "gate-1", Status: session.GateStatusOpen,
	}
	applier := Applier{SceneGate: gateStore}

	payload := scene.GateResolvedPayload{SceneID: "sc-1", GateID: "gate-1", Decision: "proceed"}
	data, _ := json.Marshal(payload)
	evt := sceneEvent(scene.EventTypeGateResolved, "camp-1", "sess-1", "sc-1", data, sceneStamp)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, _ := gateStore.GetSceneGate(ctx, "camp-1", "sc-1", "gate-1")
	if gate.Status != session.GateStatusResolved {
		t.Errorf("status = %q, want %q", gate.Status, session.GateStatusResolved)
	}
	if gate.ResolvedAt == nil {
		t.Error("expected resolved_at")
	}
}

func TestApplySceneGateResolved_MissingGateID(t *testing.T) {
	gateStore := newFakeSceneGateStore()
	applier := Applier{SceneGate: gateStore}

	data, _ := json.Marshal(scene.GateResolvedPayload{SceneID: "sc-1", GateID: ""})
	evt := sceneEvent(scene.EventTypeGateResolved, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := applier.Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing gate ID")
	}
}

func TestApplySceneGateResolved_MissingStore(t *testing.T) {
	data, _ := json.Marshal(scene.GateResolvedPayload{SceneID: "sc-1", GateID: "gate-1"})
	evt := sceneEvent(scene.EventTypeGateResolved, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := (Applier{}).Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing scene gate store")
	}
}

// --- scene.gate_abandoned ---

func TestApplySceneGateAbandoned(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSceneGateStore()
	gateStore.gates["camp-1:sc-1:gate-1"] = storage.SceneGate{
		CampaignID: "camp-1", SceneID: "sc-1", GateID: "gate-1", Status: session.GateStatusOpen,
	}
	applier := Applier{SceneGate: gateStore}

	payload := scene.GateAbandonedPayload{SceneID: "sc-1", GateID: "gate-1", Reason: "timeout"}
	data, _ := json.Marshal(payload)
	evt := sceneEvent(scene.EventTypeGateAbandoned, "camp-1", "sess-1", "sc-1", data, sceneStamp)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, _ := gateStore.GetSceneGate(ctx, "camp-1", "sc-1", "gate-1")
	if gate.Status != session.GateStatusAbandoned {
		t.Errorf("status = %q, want %q", gate.Status, session.GateStatusAbandoned)
	}
	if gate.ResolvedAt == nil {
		t.Error("expected resolved_at")
	}
}

func TestApplySceneGateAbandoned_MissingGateID(t *testing.T) {
	gateStore := newFakeSceneGateStore()
	applier := Applier{SceneGate: gateStore}

	data, _ := json.Marshal(scene.GateAbandonedPayload{SceneID: "sc-1", GateID: ""})
	evt := sceneEvent(scene.EventTypeGateAbandoned, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := applier.Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing gate ID")
	}
}

func TestApplySceneGateAbandoned_MissingStore(t *testing.T) {
	data, _ := json.Marshal(scene.GateAbandonedPayload{SceneID: "sc-1", GateID: "gate-1"})
	evt := sceneEvent(scene.EventTypeGateAbandoned, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := (Applier{}).Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing scene gate store")
	}
}

// --- scene.spotlight_set ---

func TestApplySceneSpotlightSet(t *testing.T) {
	ctx := context.Background()
	spotlightStore := newFakeSceneSpotlightStore()
	applier := Applier{SceneSpotlight: spotlightStore}

	payload := scene.SpotlightSetPayload{SceneID: "sc-1", SpotlightType: "character", CharacterID: "char-1"}
	data, _ := json.Marshal(payload)
	evt := sceneEvent(scene.EventTypeSpotlightSet, "camp-1", "sess-1", "sc-1", data, sceneStamp)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	spotlight, err := spotlightStore.GetSceneSpotlight(ctx, "camp-1", "sc-1")
	if err != nil {
		t.Fatalf("get spotlight: %v", err)
	}
	if spotlight.CharacterID != "char-1" {
		t.Errorf("character_id = %q, want %q", spotlight.CharacterID, "char-1")
	}
	if string(spotlight.SpotlightType) != "character" {
		t.Errorf("spotlight_type = %q, want %q", spotlight.SpotlightType, "character")
	}
}

func TestApplySceneSpotlightSet_MissingStore(t *testing.T) {
	data, _ := json.Marshal(scene.SpotlightSetPayload{SceneID: "sc-1", SpotlightType: "character"})
	evt := sceneEvent(scene.EventTypeSpotlightSet, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := (Applier{}).Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing scene spotlight store")
	}
}

// --- scene.spotlight_cleared ---

func TestApplySceneSpotlightCleared(t *testing.T) {
	ctx := context.Background()
	spotlightStore := newFakeSceneSpotlightStore()
	spotlightStore.spotlights["camp-1:sc-1"] = storage.SceneSpotlight{
		CampaignID: "camp-1", SceneID: "sc-1",
	}
	applier := Applier{SceneSpotlight: spotlightStore}

	data, _ := json.Marshal(scene.SpotlightClearedPayload{SceneID: "sc-1"})
	evt := sceneEvent(scene.EventTypeSpotlightCleared, "camp-1", "sess-1", "sc-1", data, sceneStamp)

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(spotlightStore.cleared) != 1 {
		t.Fatalf("cleared count = %d, want 1", len(spotlightStore.cleared))
	}
}

func TestApplySceneSpotlightCleared_MissingStore(t *testing.T) {
	data, _ := json.Marshal(scene.SpotlightClearedPayload{SceneID: "sc-1"})
	evt := sceneEvent(scene.EventTypeSpotlightCleared, "camp-1", "sess-1", "sc-1", data, sceneStamp)
	if err := (Applier{}).Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing scene spotlight store")
	}
}

// --- resolveSceneID ---

func TestResolveSceneID_PrefersPayload(t *testing.T) {
	id, err := resolveSceneID("sc-payload", "sc-envelope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sc-payload" {
		t.Errorf("id = %q, want %q", id, "sc-payload")
	}
}

func TestResolveSceneID_FallsBackToEnvelope(t *testing.T) {
	id, err := resolveSceneID("", "sc-envelope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sc-envelope" {
		t.Errorf("id = %q, want %q", id, "sc-envelope")
	}
}

func TestResolveSceneID_ErrorWhenBothEmpty(t *testing.T) {
	_, err := resolveSceneID("", "")
	if err == nil {
		t.Fatal("expected error for empty scene IDs")
	}
}
