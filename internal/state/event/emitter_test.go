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

func TestEmitter_EmitCharacterStateChanged(t *testing.T) {
	store := &mockStore{}
	emitter := NewEmitter(store)

	hpBefore := 10
	hpAfter := 8
	evt, err := emitter.EmitCharacterStateChanged(context.Background(), "camp-1", "sess-1", "participant-1", CharacterStateChangedPayload{
		CharacterID: "char-1",
		HpBefore:    &hpBefore,
		HpAfter:     &hpAfter,
		SystemState: map[string]any{"hope_before": 3, "hope_after": 4},
	})
	if err != nil {
		t.Fatalf("EmitCharacterStateChanged failed: %v", err)
	}

	if evt.Type != TypeCharacterStateChanged {
		t.Errorf("expected type %s, got %s", TypeCharacterStateChanged, evt.Type)
	}
	if evt.SessionID != "sess-1" {
		t.Errorf("expected session ID sess-1, got %s", evt.SessionID)
	}
	if evt.ActorID != "participant-1" {
		t.Errorf("expected actor ID participant-1, got %s", evt.ActorID)
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
