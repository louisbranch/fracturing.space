package daggerheart

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestHandleRestTaken_ReturnsSnapshotWriteError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.putSnapshotErr = errors.New("snapshot write failed")
	adapter := NewAdapter(store)

	err := adapter.handleRestTaken(context.Background(), event.Event{CampaignID: "camp-1"}, RestTakenPayload{
		GMFearAfter:     2,
		ShortRestsAfter: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "put daggerheart snapshot: snapshot write failed") {
		t.Fatalf("handleRestTaken() error = %v, want wrapped snapshot write error", err)
	}
}

func TestHandleRestTaken_StopsOnClearTemporaryArmorError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.getCharacterStateErr = errors.New("character read failed")
	adapter := NewAdapter(store)

	err := adapter.handleRestTaken(context.Background(), event.Event{CampaignID: "camp-1"}, RestTakenPayload{
		GMFearAfter:     2,
		ShortRestsAfter: 1,
		RefreshRest:     true,
		CharacterStates: []RestCharacterStatePatch{
			{CharacterID: "char-1"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "get daggerheart character state: character read failed") {
		t.Fatalf("handleRestTaken() error = %v, want wrapped character read error", err)
	}
}

func TestHandleRestTaken_PropagatesPatchWriteError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.putCharacterStateErr = errors.New("character write failed")
	adapter := NewAdapter(store)
	hopeAfter := 3

	err := adapter.handleRestTaken(context.Background(), event.Event{CampaignID: "camp-1"}, RestTakenPayload{
		GMFearAfter:     2,
		ShortRestsAfter: 1,
		CharacterStates: []RestCharacterStatePatch{
			{CharacterID: "char-1", HopeAfter: &hopeAfter},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "put daggerheart character state: character write failed") {
		t.Fatalf("handleRestTaken() error = %v, want wrapped character write error", err)
	}
}

func TestClearRestTemporaryArmor_PutErrorWhenChanged(t *testing.T) {
	store := newFaultDaggerheartStore()
	adapter := NewAdapter(store)
	if err := store.PutDaggerheartCharacterState(context.Background(), storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       3,
		TemporaryArmor: []storage.DaggerheartTemporaryArmor{
			{Source: "ritual", Duration: "short_rest", Amount: 2},
		},
	}); err != nil {
		t.Fatalf("seed character state: %v", err)
	}
	store.putCharacterStateErr = errors.New("character write failed")

	err := adapter.clearRestTemporaryArmor(context.Background(), "camp-1", "char-1", true, false)
	if err == nil || !strings.Contains(err.Error(), "put daggerheart character state: character write failed") {
		t.Fatalf("clearRestTemporaryArmor() error = %v, want wrapped write error", err)
	}
}

func TestClearRestTemporaryArmor_NoChange(t *testing.T) {
	store := newFaultDaggerheartStore()
	adapter := NewAdapter(store)
	if err := store.PutDaggerheartCharacterState(context.Background(), storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       2,
	}); err != nil {
		t.Fatalf("seed character state: %v", err)
	}

	if err := adapter.clearRestTemporaryArmor(context.Background(), "camp-1", "char-1", true, false); err != nil {
		t.Fatalf("clearRestTemporaryArmor() error = %v, want nil", err)
	}
}

func TestHandleGMFearChanged_UsesExistingShortRests(t *testing.T) {
	store := newFaultDaggerheartStore()
	adapter := NewAdapter(store)
	if err := store.PutDaggerheartSnapshot(context.Background(), storage.DaggerheartSnapshot{
		CampaignID:            "camp-1",
		GMFear:                1,
		ConsecutiveShortRests: 7,
	}); err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	if err := adapter.handleGMFearChanged(context.Background(), event.Event{CampaignID: "camp-1"}, GMFearChangedPayload{
		Before: 1,
		After:  3,
	}); err != nil {
		t.Fatalf("handleGMFearChanged() error = %v", err)
	}
	snapshot, err := store.GetDaggerheartSnapshot(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetDaggerheartSnapshot() error = %v", err)
	}
	if snapshot.GMFear != 3 || snapshot.ConsecutiveShortRests != 7 {
		t.Fatalf("snapshot = %+v, want gm_fear=3 short_rests=7", snapshot)
	}
}

func TestHandleGMFearChanged_GetSnapshotErrorFallsBackToZero(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.getSnapshotErr = errors.New("snapshot read failed")
	adapter := NewAdapter(store)

	if err := adapter.handleGMFearChanged(context.Background(), event.Event{CampaignID: "camp-1"}, GMFearChangedPayload{
		Before: 0,
		After:  4,
	}); err != nil {
		t.Fatalf("handleGMFearChanged() error = %v", err)
	}
	store.getSnapshotErr = nil
	snapshot, err := store.GetDaggerheartSnapshot(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("GetDaggerheartSnapshot() error = %v", err)
	}
	if snapshot.ConsecutiveShortRests != 0 {
		t.Fatalf("ConsecutiveShortRests = %d, want 0", snapshot.ConsecutiveShortRests)
	}
}

func TestHandleGMFearChanged_ReturnsPutError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.putSnapshotErr = errors.New("snapshot write failed")
	adapter := NewAdapter(store)

	err := adapter.handleGMFearChanged(context.Background(), event.Event{CampaignID: "camp-1"}, GMFearChangedPayload{
		Before: 1,
		After:  3,
	})
	if err == nil || !strings.Contains(err.Error(), "put daggerheart snapshot: snapshot write failed") {
		t.Fatalf("handleGMFearChanged() error = %v, want wrapped snapshot write error", err)
	}
}

func TestHandleCountdownUpdated_Branches(t *testing.T) {
	t.Run("get error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.getCountdownErr = errors.New("countdown read failed")
		adapter := NewAdapter(store)

		err := adapter.handleCountdownUpdated(context.Background(), event.Event{CampaignID: "camp-1"}, CountdownUpdatedPayload{
			CountdownID: "cd-1",
			Before:      1,
			After:       2,
		})
		if err == nil || err.Error() != "countdown read failed" {
			t.Fatalf("handleCountdownUpdated() error = %v, want countdown read failed", err)
		}
	})

	t.Run("projection mismatch", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartCountdown(context.Background(), storage.DaggerheartCountdown{
			CampaignID:  "camp-1",
			CountdownID: "cd-1",
			Name:        "Doom",
			Kind:        "progress",
			Current:     1,
			Max:         4,
			Direction:   "increase",
		}); err != nil {
			t.Fatalf("seed countdown: %v", err)
		}

		err := adapter.handleCountdownUpdated(context.Background(), event.Event{CampaignID: "camp-1"}, CountdownUpdatedPayload{
			CountdownID: "cd-1",
			Before:      2,
			After:       3,
		})
		if err == nil {
			t.Fatal("expected projection mismatch error")
		}
	})
}

func TestApplyAdversaryConditionPatch_WrapsStoreErrors(t *testing.T) {
	t.Run("get error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.getAdversaryErr = errors.New("adversary read failed")
		adapter := NewAdapter(store)

		err := adapter.applyAdversaryConditionPatch(context.Background(), "camp-1", "adv-1", []string{"hidden"})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart adversary: adversary read failed") {
			t.Fatalf("applyAdversaryConditionPatch() error = %v, want wrapped read error", err)
		}
	})

	t.Run("put error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartAdversary(context.Background(), storage.DaggerheartAdversary{
			CampaignID:  "camp-1",
			AdversaryID: "adv-1",
			Name:        "Adversary",
			HP:          10,
			HPMax:       10,
			StressMax:   6,
			Evasion:     10,
			Major:       2,
			Severe:      4,
			Armor:       1,
			CreatedAt:   time.Unix(0, 0).UTC(),
			UpdatedAt:   time.Unix(0, 0).UTC(),
		}); err != nil {
			t.Fatalf("seed adversary: %v", err)
		}
		store.putAdversaryErr = errors.New("adversary write failed")

		err := adapter.applyAdversaryConditionPatch(context.Background(), "camp-1", "adv-1", []string{"hidden"})
		if err == nil || !strings.Contains(err.Error(), "put daggerheart adversary: adversary write failed") {
			t.Fatalf("applyAdversaryConditionPatch() error = %v, want wrapped write error", err)
		}
	})
}

func TestApplyStatePatch_Branches(t *testing.T) {
	t.Run("character read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.getCharacterStateErr = errors.New("character read failed")
		adapter := NewAdapter(store)
		hopeAfter := 3

		err := adapter.applyStatePatch(context.Background(), "camp-1", "char-1", nil, &hopeAfter, nil, nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character state: character read failed") {
			t.Fatalf("applyStatePatch() error = %v, want wrapped read error", err)
		}
	})

	t.Run("profile read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartCharacterState(context.Background(), storage.DaggerheartCharacterState{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Armor:       2,
		}); err != nil {
			t.Fatalf("seed character state: %v", err)
		}
		store.getCharacterProfileErr = errors.New("profile read failed")
		hopeAfter := 3

		err := adapter.applyStatePatch(context.Background(), "camp-1", "char-1", nil, &hopeAfter, nil, nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character profile: profile read failed") {
			t.Fatalf("applyStatePatch() error = %v, want wrapped profile read error", err)
		}
	})
}

func TestApplyConditionPatch_WrapsPutError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.putCharacterStateErr = errors.New("character write failed")
	adapter := NewAdapter(store)

	err := adapter.applyConditionPatch(context.Background(), "camp-1", "char-1", []string{"hidden"})
	if err == nil || !strings.Contains(err.Error(), "put daggerheart character state: character write failed") {
		t.Fatalf("applyConditionPatch() error = %v, want wrapped write error", err)
	}
}

type faultDaggerheartStore struct {
	*parityDaggerheartStore

	getCharacterStateErr      error
	putCharacterStateErr      error
	getCharacterProfileErr    error
	deleteCharacterProfileErr error
	getSnapshotErr            error
	putSnapshotErr            error
	getCountdownErr           error
	putCountdownErr           error
	getAdversaryErr           error
	putAdversaryErr           error
}

func newFaultDaggerheartStore() *faultDaggerheartStore {
	return &faultDaggerheartStore{
		parityDaggerheartStore: newParityDaggerheartStore(),
	}
}

func (s *faultDaggerheartStore) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	if s.getCharacterStateErr != nil {
		return storage.DaggerheartCharacterState{}, s.getCharacterStateErr
	}
	return s.parityDaggerheartStore.GetDaggerheartCharacterState(ctx, campaignID, characterID)
}

func (s *faultDaggerheartStore) PutDaggerheartCharacterState(ctx context.Context, state storage.DaggerheartCharacterState) error {
	if s.putCharacterStateErr != nil {
		return s.putCharacterStateErr
	}
	return s.parityDaggerheartStore.PutDaggerheartCharacterState(ctx, state)
}

func (s *faultDaggerheartStore) GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	if s.getCharacterProfileErr != nil {
		return storage.DaggerheartCharacterProfile{}, s.getCharacterProfileErr
	}
	return s.parityDaggerheartStore.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
}

func (s *faultDaggerheartStore) DeleteDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) error {
	if s.deleteCharacterProfileErr != nil {
		return s.deleteCharacterProfileErr
	}
	return s.parityDaggerheartStore.DeleteDaggerheartCharacterProfile(ctx, campaignID, characterID)
}

func (s *faultDaggerheartStore) GetDaggerheartSnapshot(ctx context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	if s.getSnapshotErr != nil {
		return storage.DaggerheartSnapshot{}, s.getSnapshotErr
	}
	return s.parityDaggerheartStore.GetDaggerheartSnapshot(ctx, campaignID)
}

func (s *faultDaggerheartStore) PutDaggerheartSnapshot(ctx context.Context, snap storage.DaggerheartSnapshot) error {
	if s.putSnapshotErr != nil {
		return s.putSnapshotErr
	}
	return s.parityDaggerheartStore.PutDaggerheartSnapshot(ctx, snap)
}

func (s *faultDaggerheartStore) GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	if s.getCountdownErr != nil {
		return storage.DaggerheartCountdown{}, s.getCountdownErr
	}
	return s.parityDaggerheartStore.GetDaggerheartCountdown(ctx, campaignID, countdownID)
}

func (s *faultDaggerheartStore) PutDaggerheartCountdown(ctx context.Context, countdown storage.DaggerheartCountdown) error {
	if s.putCountdownErr != nil {
		return s.putCountdownErr
	}
	return s.parityDaggerheartStore.PutDaggerheartCountdown(ctx, countdown)
}

func (s *faultDaggerheartStore) GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	if s.getAdversaryErr != nil {
		return storage.DaggerheartAdversary{}, s.getAdversaryErr
	}
	return s.parityDaggerheartStore.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
}

func (s *faultDaggerheartStore) PutDaggerheartAdversary(ctx context.Context, adversary storage.DaggerheartAdversary) error {
	if s.putAdversaryErr != nil {
		return s.putAdversaryErr
	}
	return s.parityDaggerheartStore.PutDaggerheartAdversary(ctx, adversary)
}
