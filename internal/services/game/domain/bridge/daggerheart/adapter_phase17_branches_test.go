package daggerheart

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestAdapterSnapshot_ReturnsStoredSnapshot(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	if err := store.PutDaggerheartSnapshot(context.Background(), projectionstore.DaggerheartSnapshot{
		CampaignID:            "camp-1",
		GMFear:                4,
		ConsecutiveShortRests: 2,
	}); err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	got, err := adapter.Snapshot(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	snapshot, ok := got.(projectionstore.DaggerheartSnapshot)
	if !ok {
		t.Fatalf("Snapshot() type = %T, want projectionstore.DaggerheartSnapshot", got)
	}
	if snapshot.GMFear != 4 || snapshot.ConsecutiveShortRests != 2 {
		t.Fatalf("Snapshot() = %+v, want gm_fear=4 short_rests=2", snapshot)
	}
}

func TestHandleDowntimeMoveApplied_ErrorBranches(t *testing.T) {
	t.Run("character state read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.getCharacterStateErr = errors.New("character read failed")
		adapter := NewAdapter(store)

		err := adapter.handleDowntimeMoveApplied(context.Background(), event.Event{CampaignID: "camp-1"}, DowntimeMoveAppliedPayload{
			ActorCharacterID:  "char-1",
			TargetCharacterID: "char-1",
			Move:              "prepare",
			Hope:              intPtr(3),
		})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character state: character read failed") {
			t.Fatalf("handleDowntimeMoveApplied() error = %v, want wrapped read error", err)
		}
	})

	t.Run("profile read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartCharacterState(context.Background(), projectionstore.DaggerheartCharacterState{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Hope:        2,
			HopeMax:     6,
		}); err != nil {
			t.Fatalf("seed character state: %v", err)
		}
		store.getCharacterProfileErr = errors.New("profile read failed")

		err := adapter.handleDowntimeMoveApplied(context.Background(), event.Event{CampaignID: "camp-1"}, DowntimeMoveAppliedPayload{
			ActorCharacterID:  "char-1",
			TargetCharacterID: "char-1",
			Move:              "prepare",
			Hope:              intPtr(3),
		})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character profile: profile read failed") {
			t.Fatalf("handleDowntimeMoveApplied() error = %v, want wrapped profile read error", err)
		}
	})

	t.Run("projection validation error", func(t *testing.T) {
		store := newParityDaggerheartStore()
		adapter := NewAdapter(store)

		err := adapter.handleDowntimeMoveApplied(context.Background(), event.Event{CampaignID: "camp-1"}, DowntimeMoveAppliedPayload{
			ActorCharacterID:  "char-1",
			TargetCharacterID: "char-1",
			Move:              "prepare",
			Hope:              intPtr(-1),
		})
		if err == nil || !strings.Contains(err.Error(), "character_state hope must be in range") {
			t.Fatalf("handleDowntimeMoveApplied() error = %v, want projection validation error", err)
		}
	})

	t.Run("state write error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.putCharacterStateErr = errors.New("character write failed")
		adapter := NewAdapter(store)

		err := adapter.handleDowntimeMoveApplied(context.Background(), event.Event{CampaignID: "camp-1"}, DowntimeMoveAppliedPayload{
			ActorCharacterID:  "char-1",
			TargetCharacterID: "char-1",
			Move:              "prepare",
			Hope:              intPtr(3),
		})
		if err == nil || !strings.Contains(err.Error(), "put daggerheart character state: character write failed") {
			t.Fatalf("handleDowntimeMoveApplied() error = %v, want wrapped write error", err)
		}
	})
}

func TestHandleCharacterTemporaryArmorApplied_ErrorBranches(t *testing.T) {
	t.Run("character state read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.getCharacterStateErr = errors.New("character read failed")
		adapter := NewAdapter(store)

		err := adapter.handleCharacterTemporaryArmorApplied(context.Background(), event.Event{CampaignID: "camp-1"}, CharacterTemporaryArmorAppliedPayload{
			CharacterID: "char-1",
			Source:      "ritual",
			Duration:    "short_rest",
			Amount:      2,
		})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character state: character read failed") {
			t.Fatalf("handleCharacterTemporaryArmorApplied() error = %v, want wrapped read error", err)
		}
	})

	t.Run("profile read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartCharacterState(context.Background(), projectionstore.DaggerheartCharacterState{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Armor:       1,
		}); err != nil {
			t.Fatalf("seed character state: %v", err)
		}
		store.getCharacterProfileErr = errors.New("profile read failed")

		err := adapter.handleCharacterTemporaryArmorApplied(context.Background(), event.Event{CampaignID: "camp-1"}, CharacterTemporaryArmorAppliedPayload{
			CharacterID: "char-1",
			Source:      "ritual",
			Duration:    "short_rest",
			Amount:      2,
		})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character profile: profile read failed") {
			t.Fatalf("handleCharacterTemporaryArmorApplied() error = %v, want wrapped profile read error", err)
		}
	})

	t.Run("projection validation error", func(t *testing.T) {
		store := newParityDaggerheartStore()
		adapter := NewAdapter(store)

		err := adapter.handleCharacterTemporaryArmorApplied(context.Background(), event.Event{CampaignID: "camp-1"}, CharacterTemporaryArmorAppliedPayload{
			CharacterID: "char-1",
			Source:      "ritual",
			Duration:    "short_rest",
			Amount:      1000,
		})
		if err == nil || !strings.Contains(err.Error(), "character_state armor must be in range") {
			t.Fatalf("handleCharacterTemporaryArmorApplied() error = %v, want projection validation error", err)
		}
	})

	t.Run("state write error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.putCharacterStateErr = errors.New("character write failed")
		adapter := NewAdapter(store)

		err := adapter.handleCharacterTemporaryArmorApplied(context.Background(), event.Event{CampaignID: "camp-1"}, CharacterTemporaryArmorAppliedPayload{
			CharacterID: "char-1",
			Source:      "ritual",
			Duration:    "short_rest",
			Amount:      2,
		})
		if err == nil || !strings.Contains(err.Error(), "put daggerheart character state: character write failed") {
			t.Fatalf("handleCharacterTemporaryArmorApplied() error = %v, want wrapped write error", err)
		}
	})
}

func TestHandleGMFearChanged_RejectsOutOfRangeAfter(t *testing.T) {
	adapter := NewAdapter(newParityDaggerheartStore())
	err := adapter.handleGMFearChanged(context.Background(), event.Event{CampaignID: "camp-1"}, GMFearChangedPayload{
		Value: GMFearMax + 1,
	})
	if err == nil || !strings.Contains(err.Error(), "gm_fear_changed value must be in range") {
		t.Fatalf("handleGMFearChanged() error = %v, want out-of-range error", err)
	}
}

func TestHandleAdversaryCreated_RejectsInvalidStats(t *testing.T) {
	adapter := NewAdapter(newParityDaggerheartStore())
	err := adapter.handleAdversaryCreated(context.Background(), event.Event{CampaignID: "camp-1", Timestamp: time.Unix(0, 0).UTC()}, AdversaryCreatedPayload{
		AdversaryID: "adv-1",
		Name:        "Goblin",
		Kind:        "bruiser",
		HP:          7,
		HPMax:       6,
		Stress:      0,
		StressMax:   1,
		Evasion:     10,
		Major:       2,
		Severe:      4,
		Armor:       0,
	})
	if err == nil || !strings.Contains(err.Error(), "hp must be in range") {
		t.Fatalf("handleAdversaryCreated() error = %v, want stats validation error", err)
	}
}

func TestHandleAdversaryUpdated_ErrorBranches(t *testing.T) {
	t.Run("stats validation error", func(t *testing.T) {
		adapter := NewAdapter(newParityDaggerheartStore())
		err := adapter.handleAdversaryUpdated(context.Background(), event.Event{CampaignID: "camp-1"}, AdversaryUpdatedPayload{
			AdversaryID: "adv-1",
			Name:        "Goblin",
			HP:          5,
			HPMax:       6,
			Stress:      0,
			StressMax:   1,
			Evasion:     10,
			Major:       3,
			Severe:      2,
			Armor:       0,
		})
		if err == nil || !strings.Contains(err.Error(), "severe_threshold must be >= major_threshold") {
			t.Fatalf("handleAdversaryUpdated() error = %v, want stats validation error", err)
		}
	})

	t.Run("adversary read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.getAdversaryErr = errors.New("adversary read failed")
		adapter := NewAdapter(store)
		err := adapter.handleAdversaryUpdated(context.Background(), event.Event{CampaignID: "camp-1"}, AdversaryUpdatedPayload{
			AdversaryID: "adv-1",
			Name:        "Goblin",
			HP:          5,
			HPMax:       6,
			Stress:      0,
			StressMax:   1,
			Evasion:     10,
			Major:       2,
			Severe:      3,
			Armor:       0,
		})
		if err == nil || err.Error() != "adversary read failed" {
			t.Fatalf("handleAdversaryUpdated() error = %v, want adversary read failed", err)
		}
	})

	t.Run("adversary write error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartAdversary(context.Background(), projectionstore.DaggerheartAdversary{
			CampaignID:  "camp-1",
			AdversaryID: "adv-1",
			Name:        "Goblin",
			HP:          5,
			HPMax:       6,
			Stress:      0,
			StressMax:   1,
			Evasion:     10,
			Major:       2,
			Severe:      3,
			Armor:       0,
			CreatedAt:   time.Unix(0, 0).UTC(),
			UpdatedAt:   time.Unix(0, 0).UTC(),
		}); err != nil {
			t.Fatalf("seed adversary: %v", err)
		}
		store.putAdversaryErr = errors.New("adversary write failed")

		err := adapter.handleAdversaryUpdated(context.Background(), event.Event{
			CampaignID: "camp-1",
			Timestamp:  time.Unix(10, 0).UTC(),
		}, AdversaryUpdatedPayload{
			AdversaryID: "adv-1",
			Name:        "Goblin Captain",
			Kind:        "leader",
			HP:          4,
			HPMax:       6,
			Stress:      0,
			StressMax:   1,
			Evasion:     10,
			Major:       2,
			Severe:      3,
			Armor:       1,
		})
		if err == nil || err.Error() != "adversary write failed" {
			t.Fatalf("handleAdversaryUpdated() error = %v, want adversary write failed", err)
		}
	})
}

func TestApplyConditionPatch_ErrorBranches(t *testing.T) {
	t.Run("character state read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.getCharacterStateErr = errors.New("character read failed")
		adapter := NewAdapter(store)

		err := adapter.applyConditionPatch(context.Background(), "camp-1", "char-1", []ConditionState{mustTestConditionState(t, "hidden")})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character state: character read failed") {
			t.Fatalf("applyConditionPatch() error = %v, want wrapped read error", err)
		}
	})

	t.Run("character profile read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartCharacterState(context.Background(), projectionstore.DaggerheartCharacterState{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Armor:       1,
		}); err != nil {
			t.Fatalf("seed character state: %v", err)
		}
		store.getCharacterProfileErr = errors.New("profile read failed")

		err := adapter.applyConditionPatch(context.Background(), "camp-1", "char-1", []ConditionState{mustTestConditionState(t, "hidden")})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character profile: profile read failed") {
			t.Fatalf("applyConditionPatch() error = %v, want wrapped profile read error", err)
		}
	})
}
