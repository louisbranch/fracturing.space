package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }
func u64Ptr(v uint64) *uint64 { return &v }

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

func applyEvent(t *testing.T, a *Adapter, campaignID string, eventType event.Type, payload any) error {
	t.Helper()
	return a.ApplyEvent(context.Background(), event.Event{
		CampaignID:  campaignID,
		Type:        eventType,
		PayloadJSON: mustJSON(t, payload),
		Timestamp:   time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
	})
}

func TestNewAdapter(t *testing.T) {
	store := newMemoryDaggerheartStore()
	a := NewAdapter(store)
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestAdapterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	if a.ID() != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("expected daggerheart system id, got %v", a.ID())
	}
}

func TestAdapterVersion(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	if a.Version() != SystemVersion {
		t.Fatalf("expected %v, got %v", SystemVersion, a.Version())
	}
}

func TestApplyEventNilAdapter(t *testing.T) {
	var a *Adapter
	err := a.ApplyEvent(context.Background(), event.Event{})
	if err == nil {
		t.Fatal("expected error for nil adapter")
	}
}

func TestApplyEventNilStore(t *testing.T) {
	a := &Adapter{}
	err := a.ApplyEvent(context.Background(), event.Event{})
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestApplyEventUnknownType(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Type:        "action.unknown",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("unknown event types should be ignored, got: %v", err)
	}
}

func TestSnapshotNilAdapter(t *testing.T) {
	var a *Adapter
	_, err := a.Snapshot(context.Background(), "camp-1")
	if err == nil {
		t.Fatal("expected error for nil adapter")
	}
}

func TestSnapshotEmptyCampaignID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	_, err := a.Snapshot(context.Background(), "  ")
	if err == nil {
		t.Fatal("expected error for empty campaign id")
	}
}

func TestSnapshotSuccess(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.snaps["camp-1"] = storage.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 3}
	a := NewAdapter(store)
	snap, err := a.Snapshot(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := snap.(storage.DaggerheartSnapshot)
	if !ok {
		t.Fatal("expected DaggerheartSnapshot type")
	}
	if s.GMFear != 3 {
		t.Fatalf("expected gm fear 3, got %d", s.GMFear)
	}
}

// --- DamageApplied tests ---

func TestApplyDamageApplied(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1",
		Hp: 6, Hope: 2, Stress: 0, Armor: 2,
	}
	a := NewAdapter(store)

	err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
		CharacterID: "char-1",
		HpAfter:     intPtr(4),
		ArmorAfter:  intPtr(1),
		ArmorSpent:  1,
		Severity:    "minor",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state := store.states["camp-1:char-1"]
	if state.Hp != 4 {
		t.Fatalf("expected hp 4, got %d", state.Hp)
	}
	if state.Armor != 1 {
		t.Fatalf("expected armor 1, got %d", state.Armor)
	}
}

func TestApplyDamageAppliedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Type:        EventTypeDamageApplied,
		PayloadJSON: []byte(`{invalid`),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyDamageAppliedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
		CharacterID: "  ",
		HpAfter:     intPtr(4),
	})
	if err == nil {
		t.Fatal("expected error for empty character id")
	}
}

func TestApplyDamageAppliedInvalidArmorSpent(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
		CharacterID: "char-1",
		ArmorSpent:  -1,
	})
	if err == nil {
		t.Fatal("expected error for negative armor_spent")
	}
}

func TestApplyDamageAppliedInvalidMarks(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
		CharacterID: "char-1",
		Marks:       5,
	})
	if err == nil {
		t.Fatal("expected error for marks out of range")
	}
}

func TestApplyDamageAppliedInvalidRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
		CharacterID: "char-1",
		RollSeq:     u64Ptr(0),
	})
	if err == nil {
		t.Fatal("expected error for zero roll_seq")
	}
}

func TestApplyDamageAppliedInvalidSeverity(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
		CharacterID: "char-1",
		Severity:    "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid severity")
	}
}

func TestApplyDamageAppliedEmptySourceCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
		CharacterID:        "char-1",
		SourceCharacterIDs: []string{"char-2", "  "},
	})
	if err == nil {
		t.Fatal("expected error for empty source character id")
	}
}

func TestApplyDamageAppliedValidSeverities(t *testing.T) {
	for _, sev := range []string{"none", "minor", "major", "severe", "massive"} {
		store := newMemoryDaggerheartStore()
		store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
			CampaignID: "camp-1", CharacterID: "char-1",
			Hp: 6, Hope: 2,
		}
		a := NewAdapter(store)
		if err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
			CharacterID: "char-1",
			Severity:    sev,
		}); err != nil {
			t.Fatalf("severity %q should be valid: %v", sev, err)
		}
	}
}

// --- RestTaken tests ---

func TestApplyRestTaken(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1",
		Hp: 6, Hope: 2, Stress: 1,
	}
	a := NewAdapter(store)

	err := applyEvent(t, a, "camp-1", EventTypeRestTaken, RestTakenPayload{
		RestType:    "short",
		GMFearAfter: 2,
		CharacterStates: []RestCharacterStatePatch{
			{CharacterID: "char-1", HopeAfter: intPtr(3), StressAfter: intPtr(0)},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snap := store.snaps["camp-1"]
	if snap.GMFear != 2 {
		t.Fatalf("expected gm fear 2, got %d", snap.GMFear)
	}
	state := store.states["camp-1:char-1"]
	if state.Hope != 3 || state.Stress != 0 {
		t.Fatalf("expected hope=3 stress=0, got hope=%d stress=%d", state.Hope, state.Stress)
	}
}

func TestApplyRestTakenInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeRestTaken, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyRestTakenInvalidGMFear(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeRestTaken, RestTakenPayload{GMFearAfter: -1})
	if err == nil {
		t.Fatal("expected error for negative gm fear")
	}

	err = applyEvent(t, a, "camp-1", EventTypeRestTaken, RestTakenPayload{GMFearAfter: GMFearMax + 1})
	if err == nil {
		t.Fatal("expected error for gm fear exceeding max")
	}
}

func TestApplyRestTakenNegativeShortRests(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeRestTaken, RestTakenPayload{
		GMFearAfter:     0,
		ShortRestsAfter: -1,
	})
	if err == nil {
		t.Fatal("expected error for negative short_rests_after")
	}
}

func TestApplyRestTakenEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeRestTaken, RestTakenPayload{
		GMFearAfter:     0,
		ShortRestsAfter: 0,
		CharacterStates: []RestCharacterStatePatch{
			{CharacterID: "  "},
		},
	})
	if err == nil {
		t.Fatal("expected error for empty character_id in character_states")
	}
}

// --- DowntimeMoveApplied tests ---

func TestApplyDowntimeMoveApplied(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 2,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeDowntimeMoveApplied, DowntimeMoveAppliedPayload{
		CharacterID: "char-1",
		Move:        "craft",
		HopeAfter:   intPtr(3),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.states["camp-1:char-1"].Hope != 3 {
		t.Fatal("expected hope=3")
	}
}

func TestApplyDowntimeMoveAppliedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeDowntimeMoveApplied, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyDowntimeMoveAppliedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDowntimeMoveApplied, DowntimeMoveAppliedPayload{
		CharacterID: "",
	})
	if err == nil {
		t.Fatal("expected error for empty character_id")
	}
}

// --- LoadoutSwapped tests ---

func TestApplyLoadoutSwapped(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 2, Stress: 1,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeLoadoutSwapped, LoadoutSwappedPayload{
		CharacterID: "char-1",
		StressAfter: intPtr(2),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.states["camp-1:char-1"].Stress != 2 {
		t.Fatal("expected stress=2")
	}
}

func TestApplyLoadoutSwappedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeLoadoutSwapped, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyLoadoutSwappedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeLoadoutSwapped, LoadoutSwappedPayload{
		CharacterID: "",
	})
	if err == nil {
		t.Fatal("expected error for empty character_id")
	}
}

// --- CharacterStatePatched tests ---

func TestApplyCharacterStatePatched(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 2, HopeMax: 5,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCharacterStatePatched, CharacterStatePatchedPayload{
		CharacterID:    "char-1",
		HpAfter:        intPtr(4),
		HopeAfter:      intPtr(3),
		HopeMaxAfter:   intPtr(6),
		StressAfter:    intPtr(1),
		ArmorAfter:     intPtr(2),
		LifeStateAfter: strPtr("alive"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := store.states["camp-1:char-1"]
	if state.Hp != 4 || state.Hope != 3 || state.HopeMax != 6 || state.Stress != 1 || state.Armor != 2 {
		t.Fatalf("unexpected state: %+v", state)
	}
}

func TestApplyCharacterStatePatchedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeCharacterStatePatched, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyCharacterStatePatchedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCharacterStatePatched, CharacterStatePatchedPayload{
		CharacterID: "",
	})
	if err == nil {
		t.Fatal("expected error for empty character_id")
	}
}

// --- GMFearChanged tests ---

func TestApplyGMFearChanged(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.snaps["camp-1"] = storage.DaggerheartSnapshot{
		CampaignID: "camp-1", GMFear: 2, ConsecutiveShortRests: 1,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeGMFearChanged, GMFearChangedPayload{After: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.snaps["camp-1"].GMFear != 5 {
		t.Fatalf("expected gm fear 5, got %d", store.snaps["camp-1"].GMFear)
	}
	if store.snaps["camp-1"].ConsecutiveShortRests != 1 {
		t.Fatal("consecutive short rests should be preserved")
	}
}

func TestApplyGMFearChangedNoExistingSnapshot(t *testing.T) {
	store := newMemoryDaggerheartStore()
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeGMFearChanged, GMFearChangedPayload{After: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.snaps["camp-1"].GMFear != 3 {
		t.Fatalf("expected gm fear 3")
	}
}

func TestApplyGMFearChangedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeGMFearChanged, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyGMFearChangedOutOfRange(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGMFearChanged, GMFearChangedPayload{After: -1})
	if err == nil {
		t.Fatal("expected error for negative gm fear")
	}
	err = applyEvent(t, a, "camp-1", EventTypeGMFearChanged, GMFearChangedPayload{After: GMFearMax + 1})
	if err == nil {
		t.Fatal("expected error for gm fear over max")
	}
}

// --- GMMoveApplied tests ---

func TestApplyGMMoveApplied(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGMMoveApplied, GMMoveAppliedPayload{
		Move: "advance a threat", FearSpent: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyGMMoveAppliedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeGMMoveApplied, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyGMMoveAppliedEmptyMove(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGMMoveApplied, GMMoveAppliedPayload{Move: " "})
	if err == nil {
		t.Fatal("expected error for empty move")
	}
}

func TestApplyGMMoveAppliedNegativeFearSpent(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGMMoveApplied, GMMoveAppliedPayload{
		Move: "test", FearSpent: -1,
	})
	if err == nil {
		t.Fatal("expected error for negative fear_spent")
	}
}

func TestApplyGMMoveAppliedInvalidSeverity(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGMMoveApplied, GMMoveAppliedPayload{
		Move: "test", FearSpent: 1, Severity: "medium",
	})
	if err == nil {
		t.Fatal("expected error for invalid severity")
	}
}

// --- DeathMoveResolved tests ---

func TestApplyDeathMoveResolved(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1",
		Hp: 0, Hope: 1, Stress: 2, LifeState: LifeStateUnconscious,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeDeathMoveResolved, DeathMoveResolvedPayload{
		CharacterID:    "char-1",
		Move:           DeathMoveAvoidDeath,
		LifeStateAfter: LifeStateAlive,
		HpAfter:        intPtr(1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := store.states["camp-1:char-1"]
	if state.LifeState != LifeStateAlive {
		t.Fatalf("expected alive, got %s", state.LifeState)
	}
}

func TestApplyDeathMoveResolvedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeDeathMoveResolved, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyDeathMoveResolvedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDeathMoveResolved, DeathMoveResolvedPayload{
		CharacterID: "", Move: DeathMoveAvoidDeath, LifeStateAfter: LifeStateAlive,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyDeathMoveResolvedEmptyMove(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDeathMoveResolved, DeathMoveResolvedPayload{
		CharacterID: "char-1", Move: "", LifeStateAfter: LifeStateAlive,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyDeathMoveResolvedInvalidMove(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDeathMoveResolved, DeathMoveResolvedPayload{
		CharacterID: "char-1", Move: "invalid_move", LifeStateAfter: LifeStateAlive,
	})
	if err == nil {
		t.Fatal("expected error for invalid move")
	}
}

func TestApplyDeathMoveResolvedEmptyLifeState(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDeathMoveResolved, DeathMoveResolvedPayload{
		CharacterID: "char-1", Move: DeathMoveAvoidDeath, LifeStateAfter: "",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyDeathMoveResolvedInvalidLifeState(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDeathMoveResolved, DeathMoveResolvedPayload{
		CharacterID: "char-1", Move: DeathMoveAvoidDeath, LifeStateAfter: "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid life state")
	}
}

func TestApplyDeathMoveResolvedInvalidHopeDie(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDeathMoveResolved, DeathMoveResolvedPayload{
		CharacterID: "char-1", Move: DeathMoveRiskItAll, LifeStateAfter: LifeStateAlive,
		HopeDie: intPtr(0),
	})
	if err == nil {
		t.Fatal("expected error for invalid hope die")
	}
	err = applyEvent(t, a, "camp-1", EventTypeDeathMoveResolved, DeathMoveResolvedPayload{
		CharacterID: "char-1", Move: DeathMoveRiskItAll, LifeStateAfter: LifeStateAlive,
		HopeDie: intPtr(13),
	})
	if err == nil {
		t.Fatal("expected error for hope die > 12")
	}
}

func TestApplyDeathMoveResolvedInvalidFearDie(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDeathMoveResolved, DeathMoveResolvedPayload{
		CharacterID: "char-1", Move: DeathMoveRiskItAll, LifeStateAfter: LifeStateAlive,
		FearDie: intPtr(0),
	})
	if err == nil {
		t.Fatal("expected error for invalid fear die")
	}
}

// --- BlazeOfGloryResolved tests ---

func TestApplyBlazeOfGloryResolved(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1",
		Hp: 0, LifeState: LifeStateBlazeOfGlory,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeBlazeOfGloryResolved, BlazeOfGloryResolvedPayload{
		CharacterID: "char-1", LifeStateAfter: LifeStateDead,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.states["camp-1:char-1"].LifeState != LifeStateDead {
		t.Fatal("expected dead")
	}
}

func TestApplyBlazeOfGloryResolvedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeBlazeOfGloryResolved, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyBlazeOfGloryResolvedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeBlazeOfGloryResolved, BlazeOfGloryResolvedPayload{
		CharacterID: "", LifeStateAfter: LifeStateDead,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyBlazeOfGloryResolvedEmptyLifeState(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeBlazeOfGloryResolved, BlazeOfGloryResolvedPayload{
		CharacterID: "char-1", LifeStateAfter: "",
	})
	if err == nil {
		t.Fatal("expected error for empty life state")
	}
}

func TestApplyBlazeOfGloryResolvedInvalidLifeState(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeBlazeOfGloryResolved, BlazeOfGloryResolvedPayload{
		CharacterID: "char-1", LifeStateAfter: "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid life state")
	}
}

// --- AttackResolved tests ---

func TestApplyAttackResolved(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAttackResolved, AttackResolvedPayload{
		CharacterID: "char-1", RollSeq: 1, Targets: []string{"target-1"}, Outcome: "hit",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyAttackResolvedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeAttackResolved, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAttackResolvedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAttackResolved, AttackResolvedPayload{
		CharacterID: "", RollSeq: 1, Targets: []string{"t"}, Outcome: "hit",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAttackResolvedZeroRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAttackResolved, AttackResolvedPayload{
		CharacterID: "char-1", RollSeq: 0, Targets: []string{"t"}, Outcome: "hit",
	})
	if err == nil {
		t.Fatal("expected error for zero roll_seq")
	}
}

func TestApplyAttackResolvedEmptyTargets(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAttackResolved, AttackResolvedPayload{
		CharacterID: "char-1", RollSeq: 1, Targets: nil, Outcome: "hit",
	})
	if err == nil {
		t.Fatal("expected error for empty targets")
	}
}

func TestApplyAttackResolvedEmptyTargetValue(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAttackResolved, AttackResolvedPayload{
		CharacterID: "char-1", RollSeq: 1, Targets: []string{"t", "  "}, Outcome: "hit",
	})
	if err == nil {
		t.Fatal("expected error for blank target")
	}
}

func TestApplyAttackResolvedEmptyOutcome(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAttackResolved, AttackResolvedPayload{
		CharacterID: "char-1", RollSeq: 1, Targets: []string{"t"}, Outcome: " ",
	})
	if err == nil {
		t.Fatal("expected error for empty outcome")
	}
}

// --- ReactionResolved tests ---

func TestApplyReactionResolved(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeReactionResolved, ReactionResolvedPayload{
		CharacterID: "char-1", RollSeq: 1, Outcome: "dodge",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyReactionResolvedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeReactionResolved, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyReactionResolvedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeReactionResolved, ReactionResolvedPayload{
		CharacterID: "", RollSeq: 1, Outcome: "dodge",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyReactionResolvedZeroRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeReactionResolved, ReactionResolvedPayload{
		CharacterID: "char-1", RollSeq: 0, Outcome: "dodge",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyReactionResolvedEmptyOutcome(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeReactionResolved, ReactionResolvedPayload{
		CharacterID: "char-1", RollSeq: 1, Outcome: " ",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- DamageRollResolved tests ---

func TestApplyDamageRollResolved(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageRollResolved, DamageRollResolvedPayload{
		CharacterID: "char-1", RollSeq: 1, Rolls: []dice.Roll{{Sides: 6, Results: []int{4}, Total: 4}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyDamageRollResolvedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeDamageRollResolved, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyDamageRollResolvedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageRollResolved, DamageRollResolvedPayload{
		CharacterID: "", RollSeq: 1, Rolls: []dice.Roll{{Sides: 6, Results: []int{4}, Total: 4}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyDamageRollResolvedZeroRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageRollResolved, DamageRollResolvedPayload{
		CharacterID: "char-1", RollSeq: 0, Rolls: []dice.Roll{{Sides: 6, Results: []int{4}, Total: 4}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyDamageRollResolvedEmptyRolls(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeDamageRollResolved, DamageRollResolvedPayload{
		CharacterID: "char-1", RollSeq: 1,
	})
	if err == nil {
		t.Fatal("expected error for empty rolls")
	}
}

// --- GroupActionResolved tests ---

func TestApplyGroupActionResolved(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGroupActionResolved, GroupActionResolvedPayload{
		LeaderCharacterID: "char-1",
		LeaderRollSeq:     1,
		Supporters: []GroupActionSupporterRoll{
			{CharacterID: "char-2", RollSeq: 2, Success: true},
		},
		SupportSuccesses: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyGroupActionResolvedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeGroupActionResolved, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyGroupActionResolvedEmptyLeader(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGroupActionResolved, GroupActionResolvedPayload{
		LeaderCharacterID: "", LeaderRollSeq: 1,
	})
	if err == nil {
		t.Fatal("expected error for empty leader")
	}
}

func TestApplyGroupActionResolvedZeroLeaderRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGroupActionResolved, GroupActionResolvedPayload{
		LeaderCharacterID: "char-1", LeaderRollSeq: 0,
	})
	if err == nil {
		t.Fatal("expected error for zero leader roll seq")
	}
}

func TestApplyGroupActionResolvedNegativeSupportCounts(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGroupActionResolved, GroupActionResolvedPayload{
		LeaderCharacterID: "char-1", LeaderRollSeq: 1, SupportSuccesses: -1,
	})
	if err == nil {
		t.Fatal("expected error for negative support successes")
	}
}

func TestApplyGroupActionResolvedEmptySupporterCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGroupActionResolved, GroupActionResolvedPayload{
		LeaderCharacterID: "char-1", LeaderRollSeq: 1,
		Supporters: []GroupActionSupporterRoll{{CharacterID: " ", RollSeq: 2}},
	})
	if err == nil {
		t.Fatal("expected error for empty supporter character_id")
	}
}

func TestApplyGroupActionResolvedZeroSupporterRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeGroupActionResolved, GroupActionResolvedPayload{
		LeaderCharacterID: "char-1", LeaderRollSeq: 1,
		Supporters: []GroupActionSupporterRoll{{CharacterID: "char-2", RollSeq: 0}},
	})
	if err == nil {
		t.Fatal("expected error for zero supporter roll_seq")
	}
}

// --- TagTeamResolved tests ---

func TestApplyTagTeamResolved(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeTagTeamResolved, TagTeamResolvedPayload{
		FirstCharacterID:    "char-1",
		FirstRollSeq:        1,
		SecondCharacterID:   "char-2",
		SecondRollSeq:       2,
		SelectedCharacterID: "char-1",
		SelectedRollSeq:     1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyTagTeamResolvedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeTagTeamResolved, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyTagTeamResolvedEmptyFirstCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeTagTeamResolved, TagTeamResolvedPayload{
		FirstCharacterID: "", SecondCharacterID: "char-2",
		FirstRollSeq: 1, SecondRollSeq: 2,
		SelectedCharacterID: "char-2", SelectedRollSeq: 2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyTagTeamResolvedEmptySecondCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeTagTeamResolved, TagTeamResolvedPayload{
		FirstCharacterID: "char-1", SecondCharacterID: "",
		FirstRollSeq: 1, SecondRollSeq: 2,
		SelectedCharacterID: "char-1", SelectedRollSeq: 1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyTagTeamResolvedZeroRollSeqs(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeTagTeamResolved, TagTeamResolvedPayload{
		FirstCharacterID: "char-1", SecondCharacterID: "char-2",
		FirstRollSeq: 0, SecondRollSeq: 2,
		SelectedCharacterID: "char-1", SelectedRollSeq: 1,
	})
	if err == nil {
		t.Fatal("expected error for zero first_roll_seq")
	}
}

func TestApplyTagTeamResolvedEmptySelectedCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeTagTeamResolved, TagTeamResolvedPayload{
		FirstCharacterID: "char-1", SecondCharacterID: "char-2",
		FirstRollSeq: 1, SecondRollSeq: 2,
		SelectedCharacterID: "", SelectedRollSeq: 1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyTagTeamResolvedZeroSelectedRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeTagTeamResolved, TagTeamResolvedPayload{
		FirstCharacterID: "char-1", SecondCharacterID: "char-2",
		FirstRollSeq: 1, SecondRollSeq: 2,
		SelectedCharacterID: "char-1", SelectedRollSeq: 0,
	})
	if err == nil {
		t.Fatal("expected error for zero selected_roll_seq")
	}
}

func TestApplyTagTeamResolvedSelectedNotParticipant(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeTagTeamResolved, TagTeamResolvedPayload{
		FirstCharacterID: "char-1", SecondCharacterID: "char-2",
		FirstRollSeq: 1, SecondRollSeq: 2,
		SelectedCharacterID: "char-3", SelectedRollSeq: 1,
	})
	if err == nil {
		t.Fatal("expected error when selected doesn't match participant")
	}
}

func TestApplyTagTeamResolvedSelectedRollSeqMismatchFirst(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeTagTeamResolved, TagTeamResolvedPayload{
		FirstCharacterID: "char-1", SecondCharacterID: "char-2",
		FirstRollSeq: 1, SecondRollSeq: 2,
		SelectedCharacterID: "char-1", SelectedRollSeq: 99,
	})
	if err == nil {
		t.Fatal("expected error for roll_seq mismatch with first")
	}
}

func TestApplyTagTeamResolvedSelectedRollSeqMismatchSecond(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeTagTeamResolved, TagTeamResolvedPayload{
		FirstCharacterID: "char-1", SecondCharacterID: "char-2",
		FirstRollSeq: 1, SecondRollSeq: 2,
		SelectedCharacterID: "char-2", SelectedRollSeq: 99,
	})
	if err == nil {
		t.Fatal("expected error for roll_seq mismatch with second")
	}
}

// --- CountdownCreated tests ---

func TestApplyCountdownCreated(t *testing.T) {
	store := newMemoryDaggerheartStore()
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCountdownCreated, CountdownCreatedPayload{
		CountdownID: "cd-1", Name: "Doom", Kind: CountdownKindProgress,
		Current: 0, Max: 4, Direction: CountdownDirectionIncrease,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cd := store.countdowns["camp-1:cd-1"]
	if cd.Name != "Doom" || cd.Max != 4 {
		t.Fatalf("unexpected countdown: %+v", cd)
	}
}

func TestApplyCountdownCreatedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeCountdownCreated, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyCountdownCreatedEmptyID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownCreated, CountdownCreatedPayload{
		CountdownID: "", Name: "Doom", Kind: CountdownKindProgress,
		Max: 4, Direction: CountdownDirectionIncrease,
	})
	if err == nil {
		t.Fatal("expected error for empty countdown_id")
	}
}

func TestApplyCountdownCreatedEmptyName(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownCreated, CountdownCreatedPayload{
		CountdownID: "cd-1", Name: "", Kind: CountdownKindProgress,
		Max: 4, Direction: CountdownDirectionIncrease,
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestApplyCountdownCreatedZeroMax(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownCreated, CountdownCreatedPayload{
		CountdownID: "cd-1", Name: "Doom", Kind: CountdownKindProgress,
		Max: 0, Direction: CountdownDirectionIncrease,
	})
	if err == nil {
		t.Fatal("expected error for zero max")
	}
}

func TestApplyCountdownCreatedCurrentOutOfRange(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownCreated, CountdownCreatedPayload{
		CountdownID: "cd-1", Name: "Doom", Kind: CountdownKindProgress,
		Current: 5, Max: 4, Direction: CountdownDirectionIncrease,
	})
	if err == nil {
		t.Fatal("expected error for current > max")
	}
}

func TestApplyCountdownCreatedInvalidKind(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownCreated, CountdownCreatedPayload{
		CountdownID: "cd-1", Name: "Doom", Kind: "invalid",
		Max: 4, Direction: CountdownDirectionIncrease,
	})
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestApplyCountdownCreatedInvalidDirection(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownCreated, CountdownCreatedPayload{
		CountdownID: "cd-1", Name: "Doom", Kind: CountdownKindProgress,
		Max: 4, Direction: "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid direction")
	}
}

// --- CountdownUpdated tests ---

func TestApplyCountdownUpdated(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.countdowns["camp-1:cd-1"] = storage.DaggerheartCountdown{
		CampaignID: "camp-1", CountdownID: "cd-1", Current: 2, Max: 4,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCountdownUpdated, CountdownUpdatedPayload{
		CountdownID: "cd-1", Before: 2, After: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.countdowns["camp-1:cd-1"].Current != 3 {
		t.Fatal("expected current=3")
	}
}

func TestApplyCountdownUpdatedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeCountdownUpdated, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyCountdownUpdatedEmptyID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownUpdated, CountdownUpdatedPayload{
		CountdownID: "", Before: 0, After: 1,
	})
	if err == nil {
		t.Fatal("expected error for empty countdown_id")
	}
}

func TestApplyCountdownUpdatedNegativeValues(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownUpdated, CountdownUpdatedPayload{
		CountdownID: "cd-1", Before: -1, After: 1,
	})
	if err == nil {
		t.Fatal("expected error for negative before")
	}
}

func TestApplyCountdownUpdatedBeforeMismatch(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.countdowns["camp-1:cd-1"] = storage.DaggerheartCountdown{
		CampaignID: "camp-1", CountdownID: "cd-1", Current: 2, Max: 4,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCountdownUpdated, CountdownUpdatedPayload{
		CountdownID: "cd-1", Before: 0, After: 3,
	})
	if err == nil {
		t.Fatal("expected error for before mismatch")
	}
}

func TestApplyCountdownUpdatedAfterExceedsMax(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.countdowns["camp-1:cd-1"] = storage.DaggerheartCountdown{
		CampaignID: "camp-1", CountdownID: "cd-1", Current: 2, Max: 4,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCountdownUpdated, CountdownUpdatedPayload{
		CountdownID: "cd-1", Before: 2, After: 5,
	})
	if err == nil {
		t.Fatal("expected error for after > max")
	}
}

// --- CountdownDeleted tests ---

func TestApplyCountdownDeleted(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.countdowns["camp-1:cd-1"] = storage.DaggerheartCountdown{
		CampaignID: "camp-1", CountdownID: "cd-1", Current: 2, Max: 4,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCountdownDeleted, CountdownDeletedPayload{
		CountdownID: "cd-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := store.countdowns["camp-1:cd-1"]; ok {
		t.Fatal("expected countdown to be deleted")
	}
}

func TestApplyCountdownDeletedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeCountdownDeleted, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyCountdownDeletedEmptyID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownDeleted, CountdownDeletedPayload{
		CountdownID: "",
	})
	if err == nil {
		t.Fatal("expected error for empty countdown_id")
	}
}

// --- AdversaryRollResolved tests ---

func TestApplyAdversaryRollResolved(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryRollResolved, AdversaryRollResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 1, Roll: 15, Rolls: []int{15},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyAdversaryRollResolvedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeAdversaryRollResolved, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryRollResolvedEmptyAdversaryID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryRollResolved, AdversaryRollResolvedPayload{
		AdversaryID: "", RollSeq: 1, Roll: 15, Rolls: []int{15},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryRollResolvedZeroRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryRollResolved, AdversaryRollResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 0, Roll: 15, Rolls: []int{15},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryRollResolvedInvalidRoll(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryRollResolved, AdversaryRollResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 1, Roll: 0, Rolls: []int{0},
	})
	if err == nil {
		t.Fatal("expected error for roll < 1")
	}
	err = applyEvent(t, a, "camp-1", EventTypeAdversaryRollResolved, AdversaryRollResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 1, Roll: 21, Rolls: []int{21},
	})
	if err == nil {
		t.Fatal("expected error for roll > 20")
	}
}

func TestApplyAdversaryRollResolvedEmptyRolls(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryRollResolved, AdversaryRollResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 1, Roll: 15,
	})
	if err == nil {
		t.Fatal("expected error for empty rolls")
	}
}

// --- AdversaryAttackResolved tests ---

func TestApplyAdversaryAttackResolved(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryAttackResolved, AdversaryAttackResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 1, Targets: []string{"char-1"}, Roll: 15, Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyAdversaryAttackResolvedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeAdversaryAttackResolved, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryAttackResolvedEmptyAdversaryID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryAttackResolved, AdversaryAttackResolvedPayload{
		AdversaryID: "", RollSeq: 1, Targets: []string{"char-1"}, Roll: 15,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryAttackResolvedZeroRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryAttackResolved, AdversaryAttackResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 0, Targets: []string{"char-1"}, Roll: 15,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryAttackResolvedEmptyTargets(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryAttackResolved, AdversaryAttackResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 1, Targets: nil, Roll: 15,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryAttackResolvedEmptyTargetValue(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryAttackResolved, AdversaryAttackResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 1, Targets: []string{"char-1", " "}, Roll: 15,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryAttackResolvedInvalidRoll(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryAttackResolved, AdversaryAttackResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 1, Targets: []string{"char-1"}, Roll: 0,
	})
	if err == nil {
		t.Fatal("expected error for roll < 1")
	}
}

func TestApplyAdversaryAttackResolvedNegativeDifficulty(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryAttackResolved, AdversaryAttackResolvedPayload{
		AdversaryID: "adv-1", RollSeq: 1, Targets: []string{"char-1"}, Roll: 15, Difficulty: -1,
	})
	if err == nil {
		t.Fatal("expected error for negative difficulty")
	}
}

// --- AdversaryCreated tests ---

func TestApplyAdversaryCreated(t *testing.T) {
	store := newMemoryDaggerheartStore()
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryCreated, AdversaryCreatedPayload{
		AdversaryID: "adv-1", Name: "Goblin", HP: 3, HPMax: 5,
		StressMax: 2, Evasion: 10, Major: 5, Severe: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	adv := store.adversaries["camp-1:adv-1"]
	if adv.Name != "Goblin" || adv.HP != 3 {
		t.Fatalf("unexpected adversary: %+v", adv)
	}
}

func TestApplyAdversaryCreatedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeAdversaryCreated, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryCreatedEmptyID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryCreated, AdversaryCreatedPayload{
		AdversaryID: "", Name: "Goblin", HPMax: 5, Major: 5, Severe: 10,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryCreatedEmptyName(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryCreated, AdversaryCreatedPayload{
		AdversaryID: "adv-1", Name: "", HPMax: 5, Major: 5, Severe: 10,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- AdversaryUpdated tests ---

func TestApplyAdversaryUpdated(t *testing.T) {
	store := newMemoryDaggerheartStore()
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store.adversaries["camp-1:adv-1"] = storage.DaggerheartAdversary{
		CampaignID: "camp-1", AdversaryID: "adv-1", Name: "Goblin",
		HP: 3, HPMax: 5, StressMax: 2, Evasion: 10, Major: 5, Severe: 10,
		Conditions: []string{ConditionHidden},
		CreatedAt:  created,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryUpdated, AdversaryUpdatedPayload{
		AdversaryID: "adv-1", Name: "Hobgoblin", HP: 4, HPMax: 6,
		StressMax: 3, Evasion: 12, Major: 6, Severe: 12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	adv := store.adversaries["camp-1:adv-1"]
	if adv.Name != "Hobgoblin" || adv.HPMax != 6 {
		t.Fatalf("unexpected adversary: %+v", adv)
	}
	if adv.CreatedAt != created {
		t.Fatal("created_at should be preserved")
	}
	if !ConditionsEqual(adv.Conditions, []string{ConditionHidden}) {
		t.Fatalf("conditions = %v, want %v", adv.Conditions, []string{ConditionHidden})
	}
}

func TestApplyAdversaryUpdatedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeAdversaryUpdated, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryUpdatedEmptyID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryUpdated, AdversaryUpdatedPayload{
		AdversaryID: "", Name: "Goblin", HPMax: 5, Major: 5, Severe: 10,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryUpdatedEmptyName(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryUpdated, AdversaryUpdatedPayload{
		AdversaryID: "adv-1", Name: "", HPMax: 5, Major: 5, Severe: 10,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- AdversaryDamageApplied tests ---

func TestApplyAdversaryDamageApplied(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.adversaries["camp-1:adv-1"] = storage.DaggerheartAdversary{
		CampaignID: "camp-1", AdversaryID: "adv-1", Name: "Goblin",
		HP: 6, HPMax: 6, Stress: 0, StressMax: 6, Evasion: 10, Major: 5, Severe: 10,
		Armor: 1, CreatedAt: time.Now().Add(-time.Hour), UpdatedAt: time.Now().Add(-time.Hour),
	}

	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryDamageApplied, AdversaryDamageAppliedPayload{
		AdversaryID: "adv-1",
		HpAfter:     intPtr(3),
		ArmorAfter:  intPtr(0),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	adv := store.adversaries["camp-1:adv-1"]
	if adv.HP != 3 || adv.Armor != 0 {
		t.Fatalf("adversary hp/armor = %d/%d, want 3/0", adv.HP, adv.Armor)
	}
}

// --- AdversaryDeleted tests ---

func TestApplyAdversaryDeleted(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.adversaries["camp-1:adv-1"] = storage.DaggerheartAdversary{
		CampaignID: "camp-1", AdversaryID: "adv-1", Name: "Goblin",
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryDeleted, AdversaryDeletedPayload{
		AdversaryID: "adv-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyAdversaryDeletedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeAdversaryDeleted, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyAdversaryDeletedEmptyID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryDeleted, AdversaryDeletedPayload{
		AdversaryID: "",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- validateAdversaryStats tests ---

func TestValidateAdversaryStats(t *testing.T) {
	tests := []struct {
		name                                                        string
		hp, hpMax, stress, stressMax, evasion, major, severe, armor int
		wantError                                                   bool
	}{
		{"valid", 3, 5, 1, 2, 10, 5, 10, 2, false},
		{"zero_hp_max", 0, 0, 0, 0, 10, 5, 10, 0, true},
		{"hp_exceeds_max", 6, 5, 0, 0, 10, 5, 10, 0, true},
		{"hp_negative", -1, 5, 0, 0, 10, 5, 10, 0, true},
		{"negative_stress_max", 0, 5, 0, -1, 10, 5, 10, 0, true},
		{"stress_exceeds_max", 0, 5, 3, 2, 10, 5, 10, 0, true},
		{"stress_negative", 0, 5, -1, 2, 10, 5, 10, 0, true},
		{"negative_evasion", 0, 5, 0, 0, -1, 5, 10, 0, true},
		{"negative_major", 0, 5, 0, 0, 10, -1, 10, 0, true},
		{"negative_severe", 0, 5, 0, 0, 10, 5, -1, 0, true},
		{"severe_less_than_major", 0, 5, 0, 0, 10, 10, 5, 0, true},
		{"negative_armor", 0, 5, 0, 0, 10, 5, 10, -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAdversaryStats(tt.hp, tt.hpMax, tt.stress, tt.stressMax, tt.evasion, tt.major, tt.severe, tt.armor)
			if tt.wantError && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// --- ConditionChanged edge cases ---

func TestApplyConditionChangedInvalidJSON(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := a.ApplyEvent(context.Background(), event.Event{
		CampaignID: "camp-1", Type: EventTypeConditionChanged, PayloadJSON: []byte(`{bad`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyConditionChangedEmptyCharacterID(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeConditionChanged, ConditionChangedPayload{
		CharacterID: "", ConditionsAfter: []string{ConditionHidden},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyConditionChangedZeroRollSeq(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeConditionChanged, ConditionChangedPayload{
		CharacterID: "char-1", ConditionsAfter: []string{ConditionHidden}, RollSeq: u64Ptr(0),
	})
	if err == nil {
		t.Fatal("expected error for zero roll_seq")
	}
}

func TestApplyConditionChangedNilConditionsAfter(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeConditionChanged, ConditionChangedPayload{
		CharacterID: "char-1", ConditionsAfter: nil,
	})
	if err == nil {
		t.Fatal("expected error for nil conditions_after")
	}
}

func TestApplyConditionChangedInvalidAdded(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeConditionChanged, ConditionChangedPayload{
		CharacterID: "char-1", ConditionsAfter: []string{ConditionHidden},
		Added: []string{"mystery"},
	})
	if err == nil {
		t.Fatal("expected error for invalid added condition")
	}
}

func TestApplyConditionChangedInvalidRemoved(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeConditionChanged, ConditionChangedPayload{
		CharacterID: "char-1", ConditionsAfter: []string{ConditionHidden},
		Removed: []string{"mystery"},
	})
	if err == nil {
		t.Fatal("expected error for invalid removed condition")
	}
}

// --- statePatch validation edge cases ---

func TestApplyStatePatchHpOutOfRange(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, HopeMax: HopeMax,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
		CharacterID: "char-1", HpAfter: intPtr(HPMaxCap + 1),
	})
	if err == nil {
		t.Fatal("expected error for hp out of range")
	}
}

func TestApplyStatePatchHopeOutOfRange(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, HopeMax: HopeMax,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCharacterStatePatched, CharacterStatePatchedPayload{
		CharacterID: "char-1", HopeAfter: intPtr(HopeMax + 1),
	})
	if err == nil {
		t.Fatal("expected error for hope out of range")
	}
}

func TestApplyStatePatchHopeMaxOutOfRange(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, HopeMax: HopeMax,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCharacterStatePatched, CharacterStatePatchedPayload{
		CharacterID: "char-1", HopeMaxAfter: intPtr(HopeMax + 1),
	})
	if err == nil {
		t.Fatal("expected error for hope_max out of range")
	}
}

func TestApplyStatePatchStressOutOfRange(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, HopeMax: HopeMax,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCharacterStatePatched, CharacterStatePatchedPayload{
		CharacterID: "char-1", StressAfter: intPtr(StressMaxCap + 1),
	})
	if err == nil {
		t.Fatal("expected error for stress out of range")
	}
}

func TestApplyStatePatchArmorOutOfRange(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, HopeMax: HopeMax,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCharacterStatePatched, CharacterStatePatchedPayload{
		CharacterID: "char-1", ArmorAfter: intPtr(ArmorMaxCap + 1),
	})
	if err == nil {
		t.Fatal("expected error for armor out of range")
	}
}

func TestApplyStatePatchLifeStateValidation(t *testing.T) {
	store := newMemoryDaggerheartStore()
	store.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, HopeMax: HopeMax,
	}
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeCharacterStatePatched, CharacterStatePatchedPayload{
		CharacterID: "char-1", LifeStateAfter: strPtr("invalid"),
	})
	if err == nil {
		t.Fatal("expected error for invalid life state")
	}
}

func TestApplyRestTakenWithRestType(t *testing.T) {
	store := newMemoryDaggerheartStore()
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeRestTaken, RestTakenPayload{
		RestType:    "long",
		GMFearAfter: 3,
		CharacterStates: []RestCharacterStatePatch{
			{CharacterID: "char-1", ArmorAfter: intPtr(2)},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyAdversaryCreatedNegativeHP(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryCreated, AdversaryCreatedPayload{
		AdversaryID: "adv-1", Name: "Goblin", HP: -1, HPMax: 5,
		Major: 5, Severe: 10,
	})
	if err == nil {
		t.Fatal("expected error for negative HP")
	}
}

func TestApplyAdversaryUpdatedNotFound(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeAdversaryUpdated, AdversaryUpdatedPayload{
		AdversaryID: "missing", Name: "Goblin", HPMax: 5, Major: 5, Severe: 10,
	})
	if err == nil {
		t.Fatal("expected error for adversary not found")
	}
}

func TestApplyCountdownUpdatedNotFound(t *testing.T) {
	a := NewAdapter(newMemoryDaggerheartStore())
	err := applyEvent(t, a, "camp-1", EventTypeCountdownUpdated, CountdownUpdatedPayload{
		CountdownID: "missing", Before: 0, After: 1,
	})
	if err == nil {
		t.Fatal("expected error for countdown not found")
	}
}

// --- statePatch edge case: new character state ---

func TestApplyStatePatchCreatesNewState(t *testing.T) {
	store := newMemoryDaggerheartStore()
	a := NewAdapter(store)
	// Apply damage to a character that doesn't exist yet in store
	err := applyEvent(t, a, "camp-1", EventTypeDamageApplied, DamageAppliedPayload{
		CharacterID: "char-new",
		HpAfter:     intPtr(4),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := store.states["camp-1:char-new"]
	if state.Hp != 4 {
		t.Fatalf("expected hp 4, got %d", state.Hp)
	}
}

// --- conditionPatch edge case: new character state ---

func TestApplyConditionPatchCreatesNewState(t *testing.T) {
	store := newMemoryDaggerheartStore()
	a := NewAdapter(store)
	err := applyEvent(t, a, "camp-1", EventTypeConditionChanged, ConditionChangedPayload{
		CharacterID:     "char-new",
		ConditionsAfter: []string{ConditionHidden},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := store.states["camp-1:char-new"]
	if len(state.Conditions) != 1 || state.Conditions[0] != ConditionHidden {
		t.Fatalf("expected [hidden], got %v", state.Conditions)
	}
}
