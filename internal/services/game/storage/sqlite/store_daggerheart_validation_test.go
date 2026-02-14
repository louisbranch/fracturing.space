package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestDaggerheartValidationGuards(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// PutDaggerheartCharacterProfile
	if err := store.PutDaggerheartCharacterProfile(ctx, storage.DaggerheartCharacterProfile{CharacterID: "c"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutDaggerheartCharacterProfile")
	}
	if err := store.PutDaggerheartCharacterProfile(ctx, storage.DaggerheartCharacterProfile{CampaignID: "c"}); err == nil {
		t.Fatal("expected error for empty character ID in PutDaggerheartCharacterProfile")
	}

	// GetDaggerheartCharacterProfile
	if _, err := store.GetDaggerheartCharacterProfile(ctx, "", "ch"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetDaggerheartCharacterProfile")
	}
	if _, err := store.GetDaggerheartCharacterProfile(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty character ID in GetDaggerheartCharacterProfile")
	}

	// PutDaggerheartCharacterState
	if err := store.PutDaggerheartCharacterState(ctx, storage.DaggerheartCharacterState{CharacterID: "c"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutDaggerheartCharacterState")
	}
	if err := store.PutDaggerheartCharacterState(ctx, storage.DaggerheartCharacterState{CampaignID: "c"}); err == nil {
		t.Fatal("expected error for empty character ID in PutDaggerheartCharacterState")
	}

	// GetDaggerheartCharacterState
	if _, err := store.GetDaggerheartCharacterState(ctx, "", "ch"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetDaggerheartCharacterState")
	}
	if _, err := store.GetDaggerheartCharacterState(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty character ID in GetDaggerheartCharacterState")
	}

	// PutDaggerheartSnapshot
	if err := store.PutDaggerheartSnapshot(ctx, storage.DaggerheartSnapshot{}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutDaggerheartSnapshot")
	}

	// PutDaggerheartCountdown
	if err := store.PutDaggerheartCountdown(ctx, storage.DaggerheartCountdown{CountdownID: "cd"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutDaggerheartCountdown")
	}
	if err := store.PutDaggerheartCountdown(ctx, storage.DaggerheartCountdown{CampaignID: "c"}); err == nil {
		t.Fatal("expected error for empty countdown ID in PutDaggerheartCountdown")
	}

	// GetDaggerheartCountdown
	if _, err := store.GetDaggerheartCountdown(ctx, "", "cd"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetDaggerheartCountdown")
	}
	if _, err := store.GetDaggerheartCountdown(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty countdown ID in GetDaggerheartCountdown")
	}

	// ListDaggerheartCountdowns
	if _, err := store.ListDaggerheartCountdowns(ctx, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in ListDaggerheartCountdowns")
	}

	// DeleteDaggerheartCountdown
	if err := store.DeleteDaggerheartCountdown(ctx, "", "cd"); err == nil {
		t.Fatal("expected error for empty campaign ID in DeleteDaggerheartCountdown")
	}
	if err := store.DeleteDaggerheartCountdown(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty countdown ID in DeleteDaggerheartCountdown")
	}

	// PutDaggerheartAdversary
	if err := store.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{AdversaryID: "a"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutDaggerheartAdversary")
	}
	if err := store.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{CampaignID: "c"}); err == nil {
		t.Fatal("expected error for empty adversary ID in PutDaggerheartAdversary")
	}

	// GetDaggerheartAdversary
	if _, err := store.GetDaggerheartAdversary(ctx, "", "a"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetDaggerheartAdversary")
	}
	if _, err := store.GetDaggerheartAdversary(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty adversary ID in GetDaggerheartAdversary")
	}

	// ListDaggerheartAdversaries
	if _, err := store.ListDaggerheartAdversaries(ctx, "", ""); err == nil {
		t.Fatal("expected error for empty campaign ID in ListDaggerheartAdversaries")
	}

	// DeleteDaggerheartAdversary
	if err := store.DeleteDaggerheartAdversary(ctx, "", "a"); err == nil {
		t.Fatal("expected error for empty campaign ID in DeleteDaggerheartAdversary")
	}
	if err := store.DeleteDaggerheartAdversary(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty adversary ID in DeleteDaggerheartAdversary")
	}
}

func TestDaggerheartNilStoreErrors(t *testing.T) {
	ctx := context.Background()
	var s *Store

	if err := s.PutDaggerheartCharacterProfile(ctx, storage.DaggerheartCharacterProfile{CampaignID: "c", CharacterID: "ch"}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartCharacterProfile")
	}
	if _, err := s.GetDaggerheartCharacterProfile(ctx, "c", "ch"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartCharacterProfile")
	}
	if err := s.PutDaggerheartCharacterState(ctx, storage.DaggerheartCharacterState{CampaignID: "c", CharacterID: "ch"}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartCharacterState")
	}
	if _, err := s.GetDaggerheartCharacterState(ctx, "c", "ch"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartCharacterState")
	}
	if err := s.PutDaggerheartSnapshot(ctx, storage.DaggerheartSnapshot{CampaignID: "c"}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartSnapshot")
	}
	if _, err := s.GetDaggerheartSnapshot(ctx, "c"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartSnapshot")
	}
	if err := s.PutDaggerheartCountdown(ctx, storage.DaggerheartCountdown{CampaignID: "c", CountdownID: "cd"}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartCountdown")
	}
	if _, err := s.GetDaggerheartCountdown(ctx, "c", "cd"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartCountdown")
	}
	if _, err := s.ListDaggerheartCountdowns(ctx, "c"); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartCountdowns")
	}
	if err := s.DeleteDaggerheartCountdown(ctx, "c", "cd"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartCountdown")
	}
	if err := s.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{CampaignID: "c", AdversaryID: "a"}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartAdversary")
	}
	if _, err := s.GetDaggerheartAdversary(ctx, "c", "a"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartAdversary")
	}
	if _, err := s.ListDaggerheartAdversaries(ctx, "c", ""); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartAdversaries")
	}
	if err := s.DeleteDaggerheartAdversary(ctx, "c", "a"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartAdversary")
	}
}

func TestDaggerheartCancelledContextErrors(t *testing.T) {
	store := openTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.PutDaggerheartCharacterProfile(ctx, storage.DaggerheartCharacterProfile{CampaignID: "c", CharacterID: "ch"}); err == nil {
		t.Fatal("expected context error from PutDaggerheartCharacterProfile")
	}
	if _, err := store.GetDaggerheartCharacterProfile(ctx, "c", "ch"); err == nil {
		t.Fatal("expected context error from GetDaggerheartCharacterProfile")
	}
	if err := store.PutDaggerheartCharacterState(ctx, storage.DaggerheartCharacterState{CampaignID: "c", CharacterID: "ch"}); err == nil {
		t.Fatal("expected context error from PutDaggerheartCharacterState")
	}
	if _, err := store.GetDaggerheartCharacterState(ctx, "c", "ch"); err == nil {
		t.Fatal("expected context error from GetDaggerheartCharacterState")
	}
	if err := store.PutDaggerheartSnapshot(ctx, storage.DaggerheartSnapshot{CampaignID: "c"}); err == nil {
		t.Fatal("expected context error from PutDaggerheartSnapshot")
	}
	if _, err := store.GetDaggerheartSnapshot(ctx, "c"); err == nil {
		t.Fatal("expected context error from GetDaggerheartSnapshot")
	}
	if err := store.PutDaggerheartCountdown(ctx, storage.DaggerheartCountdown{CampaignID: "c", CountdownID: "cd"}); err == nil {
		t.Fatal("expected context error from PutDaggerheartCountdown")
	}
	if _, err := store.GetDaggerheartCountdown(ctx, "c", "cd"); err == nil {
		t.Fatal("expected context error from GetDaggerheartCountdown")
	}
	if _, err := store.ListDaggerheartCountdowns(ctx, "c"); err == nil {
		t.Fatal("expected context error from ListDaggerheartCountdowns")
	}
	if err := store.DeleteDaggerheartCountdown(ctx, "c", "cd"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartCountdown")
	}
	if err := store.PutDaggerheartAdversary(ctx, storage.DaggerheartAdversary{CampaignID: "c", AdversaryID: "a", CreatedAt: time.Now(), UpdatedAt: time.Now()}); err == nil {
		t.Fatal("expected context error from PutDaggerheartAdversary")
	}
	if _, err := store.GetDaggerheartAdversary(ctx, "c", "a"); err == nil {
		t.Fatal("expected context error from GetDaggerheartAdversary")
	}
	if _, err := store.ListDaggerheartAdversaries(ctx, "c", ""); err == nil {
		t.Fatal("expected context error from ListDaggerheartAdversaries")
	}
	if err := store.DeleteDaggerheartAdversary(ctx, "c", "a"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartAdversary")
	}
}
