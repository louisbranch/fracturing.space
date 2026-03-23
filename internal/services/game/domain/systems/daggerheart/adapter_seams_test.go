package daggerheart

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	daggerheartadapter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/adapter"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestHandleRestTaken_ReturnsSnapshotWriteError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.putSnapshotErr = errors.New("snapshot write failed")
	adapter := NewAdapter(store)

	err := adapter.HandleRestTaken(context.Background(), event.Event{CampaignID: "camp-1"}, daggerheartpayload.RestTakenPayload{
		GMFear:     2,
		ShortRests: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "put daggerheart snapshot: snapshot write failed") {
		t.Fatalf("handleRestTaken() error = %v, want wrapped snapshot write error", err)
	}
}

func TestHandleRestTaken_StopsOnClearTemporaryArmorError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.getCharacterStateErr = errors.New("character read failed")
	adapter := NewAdapter(store)

	err := adapter.HandleRestTaken(context.Background(), event.Event{CampaignID: "camp-1"}, daggerheartpayload.RestTakenPayload{
		GMFear:       2,
		ShortRests:   1,
		RefreshRest:  true,
		Participants: []ids.CharacterID{"char-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "get daggerheart character state: character read failed") {
		t.Fatalf("handleRestTaken() error = %v, want wrapped character read error", err)
	}
}

func TestClearRestTemporaryArmor_PutErrorWhenChanged(t *testing.T) {
	store := newFaultDaggerheartStore()
	adapter := NewAdapter(store)
	if err := store.PutDaggerheartCharacterState(context.Background(), projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       3,
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Source: "ritual", Duration: "short_rest", Amount: 2},
		},
	}); err != nil {
		t.Fatalf("seed character state: %v", err)
	}
	store.putCharacterStateErr = errors.New("character write failed")

	err := adapter.ClearRestTemporaryArmor(context.Background(), "camp-1", "char-1", true, false)
	if err == nil || !strings.Contains(err.Error(), "put daggerheart character state: character write failed") {
		t.Fatalf("clearRestTemporaryArmor() error = %v, want wrapped write error", err)
	}
}

func TestClearRestTemporaryArmor_NoChange(t *testing.T) {
	store := newFaultDaggerheartStore()
	adapter := NewAdapter(store)
	if err := store.PutDaggerheartCharacterState(context.Background(), projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       2,
	}); err != nil {
		t.Fatalf("seed character state: %v", err)
	}

	if err := adapter.ClearRestTemporaryArmor(context.Background(), "camp-1", "char-1", true, false); err != nil {
		t.Fatalf("clearRestTemporaryArmor() error = %v, want nil", err)
	}
}

func TestHandleGMFearChanged_UsesExistingShortRests(t *testing.T) {
	store := newFaultDaggerheartStore()
	adapter := NewAdapter(store)
	if err := store.PutDaggerheartSnapshot(context.Background(), projectionstore.DaggerheartSnapshot{
		CampaignID:            "camp-1",
		GMFear:                1,
		ConsecutiveShortRests: 7,
	}); err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	if err := adapter.HandleGMFearChanged(context.Background(), event.Event{CampaignID: "camp-1"}, daggerheartpayload.GMFearChangedPayload{
		Value: 3,
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

	if err := adapter.HandleGMFearChanged(context.Background(), event.Event{CampaignID: "camp-1"}, daggerheartpayload.GMFearChangedPayload{
		Value: 4,
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

	err := adapter.HandleGMFearChanged(context.Background(), event.Event{CampaignID: "camp-1"}, daggerheartpayload.GMFearChangedPayload{
		Value: 3,
	})
	if err == nil || !strings.Contains(err.Error(), "put daggerheart snapshot: snapshot write failed") {
		t.Fatalf("handleGMFearChanged() error = %v, want wrapped snapshot write error", err)
	}
}

func TestHandleSceneCountdownAdvanced_Branches(t *testing.T) {
	t.Run("get error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.getCountdownErr = errors.New("countdown read failed")
		adapter := NewAdapter(store)

		err := adapter.HandleSceneCountdownAdvanced(context.Background(), event.Event{CampaignID: "camp-1"}, daggerheartpayload.SceneCountdownAdvancedPayload{
			CountdownID:     "cd-1",
			BeforeRemaining: 3,
			AfterRemaining:  2,
			AdvancedBy:      1,
			StatusBefore:    "active",
			StatusAfter:     "active",
		})
		if err == nil || err.Error() != "countdown read failed" {
			t.Fatalf("HandleSceneCountdownAdvanced() error = %v, want countdown read failed", err)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartCountdown(context.Background(), projectionstore.DaggerheartCountdown{
			CampaignID:     "camp-1",
			CountdownID:    "cd-1",
			Name:           "Doom",
			StartingValue:  4,
			RemainingValue: 1,
			LoopBehavior:   "none",
			Status:         "active",
		}); err != nil {
			t.Fatalf("seed countdown: %v", err)
		}

		err := adapter.HandleSceneCountdownAdvanced(context.Background(), event.Event{CampaignID: "camp-1"}, daggerheartpayload.SceneCountdownAdvancedPayload{
			CountdownID:     "cd-1",
			BeforeRemaining: 1,
			AfterRemaining:  5,
			AdvancedBy:      4,
			StatusBefore:    "active",
			StatusAfter:     "active",
		})
		if err == nil {
			t.Fatal("expected out-of-range error")
		}
	})
}

func TestApplyAdversaryConditionPatch_WrapsStoreErrors(t *testing.T) {
	t.Run("get error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		store.getAdversaryErr = errors.New("adversary read failed")
		adapter := NewAdapter(store)

		err := adapter.ApplyAdversaryConditionPatch(context.Background(), "camp-1", "adv-1", []rules.ConditionState{mustTestConditionState(t, "hidden")})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart adversary: adversary read failed") {
			t.Fatalf("applyAdversaryConditionPatch() error = %v, want wrapped read error", err)
		}
	})

	t.Run("put error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartAdversary(context.Background(), projectionstore.DaggerheartAdversary{
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

		err := adapter.ApplyAdversaryConditionPatch(context.Background(), "camp-1", "adv-1", []rules.ConditionState{mustTestConditionState(t, "hidden")})
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
		hope := 3

		err := adapter.ApplyStatePatch(context.Background(), "camp-1", "char-1", daggerheartadapter.StatePatch{Hope: &hope})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character state: character read failed") {
			t.Fatalf("applyStatePatch() error = %v, want wrapped read error", err)
		}
	})

	t.Run("profile read error", func(t *testing.T) {
		store := newFaultDaggerheartStore()
		adapter := NewAdapter(store)
		if err := store.PutDaggerheartCharacterState(context.Background(), projectionstore.DaggerheartCharacterState{
			CampaignID:  "camp-1",
			CharacterID: "char-1",
			Armor:       2,
		}); err != nil {
			t.Fatalf("seed character state: %v", err)
		}
		store.getCharacterProfileErr = errors.New("profile read failed")
		hope := 3

		err := adapter.ApplyStatePatch(context.Background(), "camp-1", "char-1", daggerheartadapter.StatePatch{Hope: &hope})
		if err == nil || !strings.Contains(err.Error(), "get daggerheart character profile: profile read failed") {
			t.Fatalf("applyStatePatch() error = %v, want wrapped profile read error", err)
		}
	})
}

func TestApplyConditionPatch_WrapsPutError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.putCharacterStateErr = errors.New("character write failed")
	adapter := NewAdapter(store)

	err := adapter.ApplyConditionPatch(context.Background(), "camp-1", "char-1", []rules.ConditionState{mustTestConditionState(t, "hidden")})
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

func (s *faultDaggerheartStore) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if s.getCharacterStateErr != nil {
		return projectionstore.DaggerheartCharacterState{}, s.getCharacterStateErr
	}
	return s.parityDaggerheartStore.GetDaggerheartCharacterState(ctx, campaignID, characterID)
}

func (s *faultDaggerheartStore) PutDaggerheartCharacterState(ctx context.Context, state projectionstore.DaggerheartCharacterState) error {
	if s.putCharacterStateErr != nil {
		return s.putCharacterStateErr
	}
	return s.parityDaggerheartStore.PutDaggerheartCharacterState(ctx, state)
}

func (s *faultDaggerheartStore) GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	if s.getCharacterProfileErr != nil {
		return projectionstore.DaggerheartCharacterProfile{}, s.getCharacterProfileErr
	}
	return s.parityDaggerheartStore.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
}

func (s *faultDaggerheartStore) DeleteDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) error {
	if s.deleteCharacterProfileErr != nil {
		return s.deleteCharacterProfileErr
	}
	return s.parityDaggerheartStore.DeleteDaggerheartCharacterProfile(ctx, campaignID, characterID)
}

func (s *faultDaggerheartStore) GetDaggerheartSnapshot(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	if s.getSnapshotErr != nil {
		return projectionstore.DaggerheartSnapshot{}, s.getSnapshotErr
	}
	return s.parityDaggerheartStore.GetDaggerheartSnapshot(ctx, campaignID)
}

func (s *faultDaggerheartStore) PutDaggerheartSnapshot(ctx context.Context, snap projectionstore.DaggerheartSnapshot) error {
	if s.putSnapshotErr != nil {
		return s.putSnapshotErr
	}
	return s.parityDaggerheartStore.PutDaggerheartSnapshot(ctx, snap)
}

func (s *faultDaggerheartStore) GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	if s.getCountdownErr != nil {
		return projectionstore.DaggerheartCountdown{}, s.getCountdownErr
	}
	return s.parityDaggerheartStore.GetDaggerheartCountdown(ctx, campaignID, countdownID)
}

func (s *faultDaggerheartStore) PutDaggerheartCountdown(ctx context.Context, countdown projectionstore.DaggerheartCountdown) error {
	if s.putCountdownErr != nil {
		return s.putCountdownErr
	}
	return s.parityDaggerheartStore.PutDaggerheartCountdown(ctx, countdown)
}

func (s *faultDaggerheartStore) GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if s.getAdversaryErr != nil {
		return projectionstore.DaggerheartAdversary{}, s.getAdversaryErr
	}
	return s.parityDaggerheartStore.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
}

func (s *faultDaggerheartStore) PutDaggerheartAdversary(ctx context.Context, adversary projectionstore.DaggerheartAdversary) error {
	if s.putAdversaryErr != nil {
		return s.putAdversaryErr
	}
	return s.parityDaggerheartStore.PutDaggerheartAdversary(ctx, adversary)
}
