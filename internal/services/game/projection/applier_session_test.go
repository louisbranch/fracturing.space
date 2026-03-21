package projection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection/testevent"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestApplySessionGateAbandoned(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	gateStore.gates["camp-1:sess-1:gate-1"] = storage.SessionGate{
		CampaignID: "camp-1", SessionID: "sess-1", GateID: "gate-1", Status: session.GateStatusOpen,
	}
	applier := Applier{SessionGate: gateStore}

	payload := testevent.SessionGateAbandonedPayload{GateID: "gate-1", Reason: "timeout"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1",
		Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data, Timestamp: stamp,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, err := gateStore.GetSessionGate(ctx, "camp-1", "sess-1", "gate-1")
	if err != nil {
		t.Fatalf("get gate: %v", err)
	}
	if gate.Status != session.GateStatusAbandoned {
		t.Fatalf("gate status = %q, want %q", gate.Status, session.GateStatusAbandoned)
	}
	if gate.ResolvedAt == nil || !gate.ResolvedAt.Equal(stamp) {
		t.Fatalf("gate resolved at mismatch")
	}
}

func TestApplySessionGateAbandoned_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateAbandonedPayload{GateID: "gate-1"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session gate store")
	}
}

func TestApplySessionGateAbandoned_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateAbandonedPayload{GateID: "gate-1"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestApplySessionSpotlightSet(t *testing.T) {
	ctx := context.Background()
	spotlightStore := newFakeSessionSpotlightStore()
	applier := Applier{SessionSpotlight: spotlightStore}

	payload := testevent.SessionSpotlightSetPayload{SpotlightType: "character", CharacterID: "char-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 13, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1",
		Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data, Timestamp: stamp,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	spotlight, err := spotlightStore.GetSessionSpotlight(ctx, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("get spotlight: %v", err)
	}
	if spotlight.CharacterID != "char-1" {
		t.Fatalf("spotlight character = %q, want %q", spotlight.CharacterID, "char-1")
	}
}

func TestApplySessionSpotlightSet_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionSpotlightSetPayload{SpotlightType: "character", CharacterID: "c1"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing spotlight store")
	}
}

func TestApplySessionSpotlightSet_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionSpotlightSetPayload{SpotlightType: "character", CharacterID: "c1"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data}
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestApplySessionSpotlightCleared(t *testing.T) {
	ctx := context.Background()
	spotlightStore := newFakeSessionSpotlightStore()
	applier := Applier{SessionSpotlight: spotlightStore}

	data, _ := json.Marshal(testevent.SessionSpotlightClearedPayload{})
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1",
		Type: testevent.TypeSessionSpotlightCleared, PayloadJSON: data,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(spotlightStore.cleared) != 1 || spotlightStore.cleared[0] != "camp-1:sess-1" {
		t.Fatalf("expected spotlight to be cleared for camp-1:sess-1")
	}
}

func TestApplySessionSpotlightCleared_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionSpotlightClearedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightCleared, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing spotlight store")
	}
}

func TestApplySessionSpotlightCleared_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionSpotlightClearedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionSpotlightCleared, PayloadJSON: data}
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestApplySessionGateOpened(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	applier := Applier{SessionGate: gateStore}

	payload := testevent.SessionGateOpenedPayload{
		GateID:   "gate-1",
		GateType: "choice",
		Reason:   "Player decision needed",
		Metadata: map[string]any{"key": "value"},
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 19, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1",
		Type: testevent.TypeSessionGateOpened, PayloadJSON: data, Timestamp: stamp,
		ActorType: testevent.ActorTypeGM, ActorID: "gm-1",
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, err := gateStore.GetSessionGate(ctx, "camp-1", "sess-1", "gate-1")
	if err != nil {
		t.Fatalf("get gate: %v", err)
	}
	if gate.Status != session.GateStatusOpen {
		t.Fatalf("Status = %q, want open", gate.Status)
	}
	if gate.GateType != "choice" {
		t.Fatalf("GateType = %q, want %q", gate.GateType, "choice")
	}
	if gate.CreatedByActorType != "gm" {
		t.Fatalf("CreatedByActorType = %q, want gm", gate.CreatedByActorType)
	}
}

func TestApplySessionGateOpened_FallbackEntityID(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	applier := Applier{SessionGate: gateStore}

	payload := testevent.SessionGateOpenedPayload{GateID: "", GateType: "choice", Reason: "test"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-fallback",
		Type: testevent.TypeSessionGateOpened, PayloadJSON: data,
		Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := gateStore.GetSessionGate(ctx, "camp-1", "sess-1", "gate-fallback"); err != nil {
		t.Fatalf("expected gate with entity ID fallback, got err: %v", err)
	}
}

func TestApplySessionGateOpened_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateOpenedPayload{GateID: "g", GateType: "choice"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplySessionGateOpened_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateOpenedPayload{GateID: "g", GateType: "choice"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestApplySessionGateOpened_MissingGateID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateOpenedPayload{GateID: "", GateType: "choice"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing gate ID")
	}
}

func TestApplySessionGateResponseRecordedUpdatesProgress(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	metadata := map[string]any{
		"eligible_participant_ids": []string{"p1", "p2"},
	}
	progress, err := session.BuildInitialGateProgressState("decision", metadata)
	if err != nil {
		t.Fatalf("build progress: %v", err)
	}
	gateStore.gates["camp-1:sess-1:gate-1"] = storage.SessionGate{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		GateID:     "gate-1",
		GateType:   "decision",
		Status:     session.GateStatusOpen,
		Metadata:   metadata,
		Progress:   progress,
	}
	applier := Applier{SessionGate: gateStore}

	payload := testevent.SessionGateResponseRecordedPayload{
		GateID:        "gate-1",
		ParticipantID: "p1",
		Decision:      "ready",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 3, 9, 19, 30, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Type:        testevent.TypeSessionGateResponseRecorded,
		PayloadJSON: data,
		Timestamp:   stamp,
		ActorType:   testevent.ActorTypeParticipant,
		ActorID:     "p1",
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, err := gateStore.GetSessionGate(ctx, "camp-1", "sess-1", "gate-1")
	if err != nil {
		t.Fatalf("get gate: %v", err)
	}
	if gate.Progress == nil {
		t.Fatal("expected progress to be updated")
	}
	if gate.Progress.RespondedCount != 1 || gate.Progress.PendingCount != 1 {
		t.Fatalf("progress counts = %#v", gate.Progress)
	}
	if len(gate.Progress.Responses) != 1 || gate.Progress.Responses[0].ParticipantID != "p1" || gate.Progress.Responses[0].Decision != "ready" {
		t.Fatalf("responses = %#v", gate.Progress.Responses)
	}
}

// --- applySessionGateResolved tests ---

func TestApplySessionGateResolved(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	gateStore.gates["camp-1:sess-1:gate-1"] = storage.SessionGate{
		CampaignID: "camp-1", SessionID: "sess-1", GateID: "gate-1",
		Status: session.GateStatusOpen,
	}
	applier := Applier{SessionGate: gateStore}

	payload := testevent.SessionGateResolvedPayload{
		GateID:     "gate-1",
		Decision:   "approved",
		Resolution: map[string]any{"detail": "yes"},
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 19, 30, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1",
		Type: testevent.TypeSessionGateResolved, PayloadJSON: data, Timestamp: stamp,
		ActorType: testevent.ActorTypeGM, ActorID: "gm-1",
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, err := gateStore.GetSessionGate(ctx, "camp-1", "sess-1", "gate-1")
	if err != nil {
		t.Fatalf("get gate: %v", err)
	}
	if gate.Status != session.GateStatusResolved {
		t.Fatalf("Status = %q, want resolved", gate.Status)
	}
	if gate.ResolvedAt == nil || !gate.ResolvedAt.Equal(stamp) {
		t.Fatal("ResolvedAt mismatch")
	}
	if gate.ResolvedByActorType != "gm" {
		t.Fatalf("ResolvedByActorType = %q, want gm", gate.ResolvedByActorType)
	}
}

func TestApplySessionGateResolved_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateResolvedPayload{GateID: "g"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateResolved, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplySessionGateResolved_MissingGateID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateResolvedPayload{GateID: ""})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "", Type: testevent.TypeSessionGateResolved, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing gate ID")
	}
}

// --- applyCampaignCreated tests ---

func TestApplySessionGateAbandoned_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateAbandonedPayload{GateID: "gate-1"})
	evt := testevent.Event{CampaignID: "", SessionID: "sess-1", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplySessionGateAbandoned_MissingGateID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateAbandonedPayload{GateID: ""})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing gate ID")
	}
}

// --- applySessionGateResolved additional tests ---

func TestApplySessionGateResolved_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateResolvedPayload{GateID: "g"})
	evt := testevent.Event{CampaignID: "", SessionID: "sess-1", Type: testevent.TypeSessionGateResolved, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplySessionGateResolved_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateResolvedPayload{GateID: "g"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionGateResolved, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

// --- applySessionGateOpened additional tests ---

func TestApplySessionGateOpened_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateOpenedPayload{GateID: "g", GateType: "choice"})
	evt := testevent.Event{CampaignID: "", SessionID: "sess-1", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

// --- applyParticipantUpdated type assertion errors ---

func TestApplySessionSpotlightSet_InvalidJSON(t *testing.T) {
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplySessionSpotlightSet_InvalidSpotlightType(t *testing.T) {
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	payload := map[string]any{"spotlight_type": "INVALID_TYPE"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid spotlight type")
	}
}

func TestApplySessionSpotlightSet_InvalidTarget(t *testing.T) {
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	payload := map[string]any{"spotlight_type": "CHARACTER", "character_id": ""}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid spotlight target")
	}
}

func TestMarshalResolutionPayload(t *testing.T) {
	// Empty decision and resolution returns nil
	result, err := marshalResolutionPayload("", nil)
	if err != nil || result != nil {
		t.Fatalf("expected nil, got %v, %v", result, err)
	}

	// Decision only
	result, err = marshalResolutionPayload("approve", nil)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for decision-only")
	}

	// Resolution only
	result, err = marshalResolutionPayload("", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for resolution-only")
	}
}

// --- applySessionGateOpened missing branches ---

func TestApplySessionGateOpened_EmptyGateType(t *testing.T) {
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	payload := map[string]any{"gate_id": "gate-1", "gate_type": "  "}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for empty gate type")
	}
}

func TestApplySessionGateOpened_InvalidJSON(t *testing.T) {
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateOpened, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applySessionGateResolved missing branches ---

func TestApplySessionGateResolved_EntityIDFallback(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	gateStore.gates["camp-1:sess-1:gate-1"] = storage.SessionGate{
		CampaignID: "camp-1", SessionID: "sess-1", GateID: "gate-1", Status: session.GateStatusOpen,
	}
	applier := Applier{SessionGate: gateStore}
	payload := testevent.SessionGateResolvedPayload{Decision: "approve"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1", Type: testevent.TypeSessionGateResolved, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate := gateStore.gates["camp-1:sess-1:gate-1"]
	if gate.Status != session.GateStatusResolved {
		t.Fatalf("gate status = %q, want %q", gate.Status, session.GateStatusResolved)
	}
}

func TestApplySessionGateResolved_InvalidJSON(t *testing.T) {
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1", Type: testevent.TypeSessionGateResolved, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applySessionGateAbandoned missing branches ---

func TestApplySessionGateAbandoned_EntityIDFallback(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	gateStore.gates["camp-1:sess-1:gate-1"] = storage.SessionGate{
		CampaignID: "camp-1", SessionID: "sess-1", GateID: "gate-1", Status: session.GateStatusOpen,
	}
	applier := Applier{SessionGate: gateStore}
	payload := testevent.SessionGateAbandonedPayload{Reason: "timeout"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate := gateStore.gates["camp-1:sess-1:gate-1"]
	if gate.Status != session.GateStatusAbandoned {
		t.Fatalf("gate status = %q, want %q", gate.Status, session.GateStatusAbandoned)
	}
}

func TestApplySessionGateAbandoned_InvalidJSON(t *testing.T) {
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyInviteCreated missing branches ---
