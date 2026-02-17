package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type mockStore struct {
	events []Event
}

func (m *mockStore) AppendEvent(ctx context.Context, evt Event) (Event, error) {
	evt.Seq = uint64(len(m.events) + 1)
	evt.Hash = "testhash"
	m.events = append(m.events, evt)
	return evt, nil
}

func TestEmitter_Emit(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	// Override time for deterministic tests
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	emitter.now = func() time.Time { return fixedTime }

	evt, err := emitter.Emit(context.Background(), EmitInput{
		CampaignID: "camp-1",
		Type:       TypeCharacterCreated,
		ActorType:  ActorTypeSystem,
		EntityType: "character",
		EntityID:   "char-1",
		Payload:    CharacterCreatedPayload{CharacterID: "char-1", Name: "Test", Kind: "PC"},
	})
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	if evt.Seq != 1 {
		t.Errorf("expected seq 1, got %d", evt.Seq)
	}
	if evt.CampaignID != "camp-1" {
		t.Errorf("expected campaign ID camp-1, got %s", evt.CampaignID)
	}
	if evt.Type != TypeCharacterCreated {
		t.Errorf("expected type %s, got %s", TypeCharacterCreated, evt.Type)
	}
	if !evt.Timestamp.Equal(fixedTime) {
		t.Errorf("expected timestamp %v, got %v", fixedTime, evt.Timestamp)
	}

	var payload CharacterCreatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Name != "Test" {
		t.Errorf("expected name Test, got %s", payload.Name)
	}
}

func TestEmitter_EmitCampaignCreated(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitCampaignCreated(context.Background(), "camp-1", CampaignCreatedPayload{
		Name:       "Test Campaign",
		GameSystem: "DAGGERHEART",
		GmMode:     "HUMAN",
	})
	if err != nil {
		t.Fatalf("EmitCampaignCreated failed: %v", err)
	}

	if evt.Type != TypeCampaignCreated {
		t.Errorf("expected type %s, got %s", TypeCampaignCreated, evt.Type)
	}
	if evt.EntityType != "campaign" {
		t.Errorf("expected entity type campaign, got %s", evt.EntityType)
	}
	if evt.ActorType != ActorTypeSystem {
		t.Errorf("expected actor type system, got %s", evt.ActorType)
	}
}

func TestEmitter_EmitSessionStarted(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitSessionStarted(context.Background(), "camp-1", SessionStartedPayload{
		SessionID:   "sess-1",
		SessionName: "Session 1",
	})
	if err != nil {
		t.Fatalf("EmitSessionStarted failed: %v", err)
	}

	if evt.Type != TypeSessionStarted {
		t.Errorf("expected type %s, got %s", TypeSessionStarted, evt.Type)
	}
	if evt.SessionID != "sess-1" {
		t.Errorf("expected session ID sess-1, got %s", evt.SessionID)
	}
	if evt.EntityType != "session" {
		t.Errorf("expected entity type session, got %s", evt.EntityType)
	}
}

func TestEmitter_NilStore(t *testing.T) {
	emitter := &Emitter{store: nil}

	_, err := emitter.Emit(context.Background(), EmitInput{
		CampaignID: "camp-1",
		Type:       TypeCharacterCreated,
	})
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestEmitter_EmitCampaignForked(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitCampaignForked(context.Background(), "camp-2", CampaignForkedPayload{
		ParentCampaignID: "camp-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != TypeCampaignForked {
		t.Errorf("expected type %s, got %s", TypeCampaignForked, evt.Type)
	}
	if evt.EntityType != "campaign" {
		t.Errorf("expected entity type campaign, got %s", evt.EntityType)
	}
}

func TestEmitter_EmitParticipantJoined(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitParticipantJoined(context.Background(), "camp-1", ParticipantJoinedPayload{
		ParticipantID: "part-1",
		UserID:        "user-1",
		Name:          "Player",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != TypeParticipantJoined {
		t.Errorf("expected type %s, got %s", TypeParticipantJoined, evt.Type)
	}
	if evt.EntityID != "part-1" {
		t.Errorf("expected entity id part-1, got %s", evt.EntityID)
	}
}

func TestEmitter_EmitCharacterCreated(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitCharacterCreated(context.Background(), "camp-1", CharacterCreatedPayload{
		CharacterID: "char-1",
		Name:        "Hero",
		Kind:        "PC",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != TypeCharacterCreated {
		t.Errorf("expected type %s, got %s", TypeCharacterCreated, evt.Type)
	}
	if evt.EntityID != "char-1" {
		t.Errorf("expected entity id char-1, got %s", evt.EntityID)
	}
}

func TestEmitter_EmitProfileUpdated(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	// With actor ID (GM)
	evt, err := emitter.EmitProfileUpdated(context.Background(), "camp-1", "actor-1", ProfileUpdatedPayload{
		CharacterID: "char-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.ActorType != ActorTypeGM {
		t.Errorf("expected actor type GM, got %s", evt.ActorType)
	}
	if evt.ActorID != "actor-1" {
		t.Errorf("expected actor id actor-1, got %s", evt.ActorID)
	}

	// Without actor ID (system)
	evt, err = emitter.EmitProfileUpdated(context.Background(), "camp-1", "", ProfileUpdatedPayload{
		CharacterID: "char-2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.ActorType != ActorTypeSystem {
		t.Errorf("expected actor type system, got %s", evt.ActorType)
	}
}

func TestEmitter_EmitSessionEnded(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitSessionEnded(context.Background(), "camp-1", SessionEndedPayload{
		SessionID: "sess-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != TypeSessionEnded {
		t.Errorf("expected type %s, got %s", TypeSessionEnded, evt.Type)
	}
	if evt.SessionID != "sess-1" {
		t.Errorf("expected session id sess-1, got %s", evt.SessionID)
	}
}

func TestEmitter_EmitRollResolved(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitRollResolved(context.Background(), "camp-1", "sess-1", RollResolvedPayload{
		RequestID: "req-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != TypeRollResolved {
		t.Errorf("expected type %s, got %s", TypeRollResolved, evt.Type)
	}
	if evt.SessionID != "sess-1" {
		t.Errorf("expected session id sess-1, got %s", evt.SessionID)
	}
}

func TestEmitter_EmitOutcomeApplied(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitOutcomeApplied(context.Background(), "camp-1", "sess-1", OutcomeAppliedPayload{
		RequestID: "req-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != TypeOutcomeApplied {
		t.Errorf("expected type %s, got %s", TypeOutcomeApplied, evt.Type)
	}
}

func TestEmitter_EmitOutcomeRejected(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitOutcomeRejected(context.Background(), "camp-1", "sess-1", OutcomeRejectedPayload{
		RequestID: "req-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != TypeOutcomeRejected {
		t.Errorf("expected type %s, got %s", TypeOutcomeRejected, evt.Type)
	}
}

func TestEmitter_EmitNoteAdded(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	evt, err := emitter.EmitNoteAdded(context.Background(), "camp-1", "sess-1", "actor-1", NoteAddedPayload{
		Content: "A note",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != TypeNoteAdded {
		t.Errorf("expected type %s, got %s", TypeNoteAdded, evt.Type)
	}
	if evt.ActorType != ActorTypeParticipant {
		t.Errorf("expected actor type participant, got %s", evt.ActorType)
	}
	if evt.ActorID != "actor-1" {
		t.Errorf("expected actor id actor-1, got %s", evt.ActorID)
	}
}

func TestEmitter_EmitUnmarshalablePayload(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	// Use a channel (not JSON-serializable)
	_, err := emitter.Emit(context.Background(), EmitInput{
		CampaignID: "camp-1",
		Type:       TypeNoteAdded,
		Payload:    make(chan int),
	})
	if err == nil {
		t.Fatal("expected error for unmarshalable payload")
	}
}
