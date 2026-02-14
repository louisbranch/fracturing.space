package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// seedRollOutcomeCharacter seeds a character with profile and state for roll
// outcome tests.
func seedRollOutcomeCharacter(t *testing.T, store *Store, campaignID, charID string, now time.Time, hope, hopeMax, stress, stressMax int) {
	t.Helper()
	seedCharacter(t, store, campaignID, charID, "Char-"+charID, character.CharacterKindPC, now)
	if err := store.PutDaggerheartCharacterProfile(context.Background(), storage.DaggerheartCharacterProfile{
		CampaignID:  campaignID,
		CharacterID: charID,
		Level:       1,
		HpMax:       18,
		StressMax:   stressMax,
		Evasion:     10,
	}); err != nil {
		t.Fatalf("seed character profile: %v", err)
	}
	if err := store.PutDaggerheartCharacterState(context.Background(), storage.DaggerheartCharacterState{
		CampaignID:  campaignID,
		CharacterID: charID,
		Hp:          18,
		Hope:        hope,
		HopeMax:     hopeMax,
		Stress:      stress,
		LifeState:   "alive",
	}); err != nil {
		t.Fatalf("seed character state: %v", err)
	}
}

func TestApplyRollOutcomeHopeAndStress(t *testing.T) {
	store := openTestCombinedStore(t)
	now := time.Date(2026, 2, 3, 16, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-ro", now)
	seedRollOutcomeCharacter(t, store, "camp-ro", "char-1", now, 3, 6, 2, 12)

	result, err := store.ApplyRollOutcome(context.Background(), storage.RollOutcomeApplyInput{
		CampaignID:     "camp-ro",
		SessionID:      "sess-1",
		RollSeq:        1,
		Targets:        []string{"char-1"},
		RequestID:      "req-1",
		EventTimestamp: now,
		CharacterDeltas: []storage.RollOutcomeDelta{
			{CharacterID: "char-1", HopeDelta: 2, StressDelta: 1},
		},
	})
	if err != nil {
		t.Fatalf("apply roll outcome: %v", err)
	}

	if len(result.UpdatedCharacterStates) != 1 {
		t.Fatalf("expected 1 updated character state, got %d", len(result.UpdatedCharacterStates))
	}
	state := result.UpdatedCharacterStates[0]
	if state.Hope != 5 {
		t.Fatalf("expected hope 5 (3+2), got %d", state.Hope)
	}
	if state.Stress != 3 {
		t.Fatalf("expected stress 3 (2+1), got %d", state.Stress)
	}

	// Verify applied changes
	if len(result.AppliedChanges) < 2 {
		t.Fatalf("expected at least 2 applied changes, got %d", len(result.AppliedChanges))
	}

	var hopeChange, stressChange *session.OutcomeAppliedChange
	for i := range result.AppliedChanges {
		ch := &result.AppliedChanges[i]
		switch ch.Field {
		case session.OutcomeFieldHope:
			hopeChange = ch
		case session.OutcomeFieldStress:
			stressChange = ch
		}
	}
	if hopeChange == nil || hopeChange.Before != 3 || hopeChange.After != 5 {
		t.Fatalf("expected hope change 3→5, got %+v", hopeChange)
	}
	if stressChange == nil || stressChange.Before != 2 || stressChange.After != 3 {
		t.Fatalf("expected stress change 2→3, got %+v", stressChange)
	}
}

func TestApplyRollOutcomeClamps(t *testing.T) {
	store := openTestCombinedStore(t)
	now := time.Date(2026, 2, 3, 16, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-clamp", now)

	// Hope starts at 5 with max 6; Stress starts at 10 with max 12
	seedRollOutcomeCharacter(t, store, "camp-clamp", "char-1", now, 5, 6, 10, 12)

	result, err := store.ApplyRollOutcome(context.Background(), storage.RollOutcomeApplyInput{
		CampaignID:     "camp-clamp",
		SessionID:      "sess-1",
		RollSeq:        1,
		Targets:        []string{"char-1"},
		RequestID:      "req-clamp",
		EventTimestamp: now,
		CharacterDeltas: []storage.RollOutcomeDelta{
			{CharacterID: "char-1", HopeDelta: 10, StressDelta: 10},
		},
	})
	if err != nil {
		t.Fatalf("apply clamped outcome: %v", err)
	}

	state := result.UpdatedCharacterStates[0]
	// Hope capped at HopeMax (6)
	if state.Hope != 6 {
		t.Fatalf("expected hope capped at 6, got %d", state.Hope)
	}
	// Stress capped at StressMax (12)
	if state.Stress != 12 {
		t.Fatalf("expected stress capped at 12, got %d", state.Stress)
	}

	// Test floor at 0: negative deltas beyond current value
	seedCampaign(t, store, "camp-floor", now)
	seedRollOutcomeCharacter(t, store, "camp-floor", "char-2", now, 2, 6, 1, 12)

	result2, err := store.ApplyRollOutcome(context.Background(), storage.RollOutcomeApplyInput{
		CampaignID:     "camp-floor",
		SessionID:      "sess-1",
		RollSeq:        1,
		Targets:        []string{"char-2"},
		RequestID:      "req-floor",
		EventTimestamp: now,
		CharacterDeltas: []storage.RollOutcomeDelta{
			{CharacterID: "char-2", HopeDelta: -10, StressDelta: -10},
		},
	})
	if err != nil {
		t.Fatalf("apply floored outcome: %v", err)
	}

	state2 := result2.UpdatedCharacterStates[0]
	if state2.Hope != 0 {
		t.Fatalf("expected hope floored at 0, got %d", state2.Hope)
	}
	if state2.Stress != 0 {
		t.Fatalf("expected stress floored at 0, got %d", state2.Stress)
	}
}

func TestApplyRollOutcomeGMFear(t *testing.T) {
	store := openTestCombinedStore(t)
	now := time.Date(2026, 2, 3, 16, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-fear", now)
	seedRollOutcomeCharacter(t, store, "camp-fear", "char-1", now, 3, 6, 0, 12)

	result, err := store.ApplyRollOutcome(context.Background(), storage.RollOutcomeApplyInput{
		CampaignID:     "camp-fear",
		SessionID:      "sess-1",
		RollSeq:        1,
		Targets:        []string{"char-1"},
		RequestID:      "req-fear",
		EventTimestamp: now,
		GMFearDelta:    3,
	})
	if err != nil {
		t.Fatalf("apply gm fear outcome: %v", err)
	}

	if !result.GMFearChanged {
		t.Fatal("expected gm fear changed")
	}
	if result.GMFearBefore != 0 {
		t.Fatalf("expected gm fear before 0, got %d", result.GMFearBefore)
	}
	if result.GMFearAfter != 3 {
		t.Fatalf("expected gm fear after 3, got %d", result.GMFearAfter)
	}

	// Verify snapshot was updated
	snap, err := store.GetDaggerheartSnapshot(context.Background(), "camp-fear")
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if snap.GMFear != 3 {
		t.Fatalf("expected snapshot gm fear 3, got %d", snap.GMFear)
	}
}

func TestApplyRollOutcomeIdempotent(t *testing.T) {
	store := openTestCombinedStore(t)
	now := time.Date(2026, 2, 3, 16, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-idem", now)
	seedRollOutcomeCharacter(t, store, "camp-idem", "char-1", now, 3, 6, 0, 12)

	input := storage.RollOutcomeApplyInput{
		CampaignID:     "camp-idem",
		SessionID:      "sess-1",
		RollSeq:        1,
		Targets:        []string{"char-1"},
		RequestID:      "req-idem",
		EventTimestamp: now,
		CharacterDeltas: []storage.RollOutcomeDelta{
			{CharacterID: "char-1", HopeDelta: 1},
		},
	}

	if _, err := store.ApplyRollOutcome(context.Background(), input); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	_, err := store.ApplyRollOutcome(context.Background(), input)
	if err == nil || !errors.Is(err, session.ErrOutcomeAlreadyApplied) {
		t.Fatalf("expected ErrOutcomeAlreadyApplied, got %v", err)
	}
}

func TestApplyRollOutcomeCharacterNotFound(t *testing.T) {
	store := openTestCombinedStore(t)
	now := time.Date(2026, 2, 3, 16, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-cnf", now)

	_, err := store.ApplyRollOutcome(context.Background(), storage.RollOutcomeApplyInput{
		CampaignID:     "camp-cnf",
		SessionID:      "sess-1",
		RollSeq:        1,
		Targets:        []string{"no-char"},
		RequestID:      "req-cnf",
		EventTimestamp: now,
	})
	if err == nil || !errors.Is(err, session.ErrOutcomeCharacterNotFound) {
		t.Fatalf("expected ErrOutcomeCharacterNotFound, got %v", err)
	}
}

func TestApplyRollOutcomeValidation(t *testing.T) {
	store := openTestCombinedStore(t)
	now := time.Date(2026, 2, 3, 16, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input storage.RollOutcomeApplyInput
	}{
		{
			name: "empty campaign id",
			input: storage.RollOutcomeApplyInput{
				SessionID:      "sess-1",
				RollSeq:        1,
				Targets:        []string{"char-1"},
				RequestID:      "req-1",
				EventTimestamp: now,
			},
		},
		{
			name: "empty session id",
			input: storage.RollOutcomeApplyInput{
				CampaignID:     "camp-1",
				RollSeq:        1,
				Targets:        []string{"char-1"},
				RequestID:      "req-1",
				EventTimestamp: now,
			},
		},
		{
			name: "zero roll seq",
			input: storage.RollOutcomeApplyInput{
				CampaignID:     "camp-1",
				SessionID:      "sess-1",
				Targets:        []string{"char-1"},
				RequestID:      "req-1",
				EventTimestamp: now,
			},
		},
		{
			name: "empty targets",
			input: storage.RollOutcomeApplyInput{
				CampaignID:     "camp-1",
				SessionID:      "sess-1",
				RollSeq:        1,
				RequestID:      "req-1",
				EventTimestamp: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.ApplyRollOutcome(context.Background(), tt.input)
			if err == nil {
				t.Fatalf("expected validation error for %s", tt.name)
			}
		})
	}
}
