package daggerhearttestkit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestNewFakeDaggerheartStoreInitializesMaps(t *testing.T) {
	t.Parallel()

	store := NewFakeDaggerheartStore()
	if store == nil {
		t.Fatal("NewFakeDaggerheartStore() returned nil")
	}
	if store.Profiles == nil || store.States == nil || store.Snapshots == nil || store.Countdowns == nil || store.Adversaries == nil || store.EnvironmentEntities == nil {
		t.Fatalf("store maps not initialized: %+v", store)
	}
}

func TestFakeDaggerheartStoreCharacterProfilesCRUDAndPagination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewFakeDaggerheartStore()
	profiles := []projectionstore.DaggerheartCharacterProfile{
		{CampaignID: "camp-1", CharacterID: "char-1", Level: 1},
		{CampaignID: "camp-1", CharacterID: "char-2", Level: 2},
		{CampaignID: "camp-1", CharacterID: "char-3", Level: 3},
	}
	for _, profile := range profiles {
		if err := store.PutDaggerheartCharacterProfile(ctx, profile); err != nil {
			t.Fatalf("PutDaggerheartCharacterProfile() error = %v", err)
		}
	}

	got, err := store.GetDaggerheartCharacterProfile(ctx, "camp-1", "char-2")
	if err != nil {
		t.Fatalf("GetDaggerheartCharacterProfile() error = %v", err)
	}
	if got.Level != 2 {
		t.Fatalf("profile level = %d, want %d", got.Level, 2)
	}

	page1, err := store.ListDaggerheartCharacterProfiles(ctx, "camp-1", 2, "")
	if err != nil {
		t.Fatalf("ListDaggerheartCharacterProfiles(page1) error = %v", err)
	}
	if len(page1.Profiles) != 2 || page1.Profiles[0].CharacterID != "char-1" || page1.Profiles[1].CharacterID != "char-2" {
		t.Fatalf("page1 profiles = %+v", page1.Profiles)
	}
	if page1.NextPageToken != "char-2" {
		t.Fatalf("page1 next token = %q, want %q", page1.NextPageToken, "char-2")
	}

	page2, err := store.ListDaggerheartCharacterProfiles(ctx, "camp-1", 2, page1.NextPageToken)
	if err != nil {
		t.Fatalf("ListDaggerheartCharacterProfiles(page2) error = %v", err)
	}
	if len(page2.Profiles) != 1 || page2.Profiles[0].CharacterID != "char-3" {
		t.Fatalf("page2 profiles = %+v", page2.Profiles)
	}
	if page2.NextPageToken != "" {
		t.Fatalf("page2 next token = %q, want empty", page2.NextPageToken)
	}

	if err := store.DeleteDaggerheartCharacterProfile(ctx, "camp-1", "char-2"); err != nil {
		t.Fatalf("DeleteDaggerheartCharacterProfile() error = %v", err)
	}
	if _, err := store.GetDaggerheartCharacterProfile(ctx, "camp-1", "char-2"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("deleted profile error = %v, want ErrNotFound", err)
	}
}

func TestFakeDaggerheartStoreCharacterProfilesRejectInvalidPageSize(t *testing.T) {
	t.Parallel()

	store := NewFakeDaggerheartStore()
	if _, err := store.ListDaggerheartCharacterProfiles(context.Background(), "camp-1", 0, ""); err == nil {
		t.Fatal("expected page size error")
	}
}

func TestFakeDaggerheartStoreCharacterStatesAndSnapshotTrackPuts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewFakeDaggerheartStore()
	state := projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hope: 2}
	if err := store.PutDaggerheartCharacterState(ctx, state); err != nil {
		t.Fatalf("PutDaggerheartCharacterState() error = %v", err)
	}
	gotState, err := store.GetDaggerheartCharacterState(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("GetDaggerheartCharacterState() error = %v", err)
	}
	if gotState.Hope != 2 || store.StatePuts["camp-1"] != 1 {
		t.Fatalf("state = %+v, state puts = %d", gotState, store.StatePuts["camp-1"])
	}

	snap := projectionstore.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 3}
	if err := store.PutDaggerheartSnapshot(ctx, snap); err != nil {
		t.Fatalf("PutDaggerheartSnapshot() error = %v", err)
	}
	gotSnap, err := store.GetDaggerheartSnapshot(ctx, "camp-1")
	if err != nil {
		t.Fatalf("GetDaggerheartSnapshot() error = %v", err)
	}
	if gotSnap.GMFear != 3 || store.SnapPuts["camp-1"] != 1 {
		t.Fatalf("snapshot = %+v, snap puts = %d", gotSnap, store.SnapPuts["camp-1"])
	}
}

func TestFakeDaggerheartStoreCountdownsAdversariesAndEnvironmentEntities(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewFakeDaggerheartStore()
	countdown := projectionstore.DaggerheartCountdown{CampaignID: "camp-1", CountdownID: "cd-1", Name: "Clock"}
	if err := store.PutDaggerheartCountdown(ctx, countdown); err != nil {
		t.Fatalf("PutDaggerheartCountdown() error = %v", err)
	}
	if got, err := store.GetDaggerheartCountdown(ctx, "camp-1", "cd-1"); err != nil || got.Name != "Clock" {
		t.Fatalf("GetDaggerheartCountdown() = %+v, %v", got, err)
	}
	if list, err := store.ListDaggerheartCountdowns(ctx, "camp-1"); err != nil || len(list) != 1 {
		t.Fatalf("ListDaggerheartCountdowns() = %+v, %v", list, err)
	}
	if err := store.DeleteDaggerheartCountdown(ctx, "camp-1", "cd-1"); err != nil {
		t.Fatalf("DeleteDaggerheartCountdown() error = %v", err)
	}
	if err := store.DeleteDaggerheartCountdown(ctx, "camp-1", "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("missing countdown delete error = %v, want ErrNotFound", err)
	}

	adversary1 := projectionstore.DaggerheartAdversary{CampaignID: "camp-1", AdversaryID: "adv-1", SessionID: "sess-1"}
	adversary2 := projectionstore.DaggerheartAdversary{CampaignID: "camp-1", AdversaryID: "adv-2", SessionID: "sess-2"}
	if err := store.PutDaggerheartAdversary(ctx, adversary1); err != nil {
		t.Fatalf("PutDaggerheartAdversary(1) error = %v", err)
	}
	if err := store.PutDaggerheartAdversary(ctx, adversary2); err != nil {
		t.Fatalf("PutDaggerheartAdversary(2) error = %v", err)
	}
	if got, err := store.GetDaggerheartAdversary(ctx, "camp-1", "adv-1"); err != nil || got.AdversaryID != "adv-1" {
		t.Fatalf("GetDaggerheartAdversary() = %+v, %v", got, err)
	}
	if list, err := store.ListDaggerheartAdversaries(ctx, "camp-1", "sess-1"); err != nil || len(list) != 1 || list[0].AdversaryID != "adv-1" {
		t.Fatalf("ListDaggerheartAdversaries() = %+v, %v", list, err)
	}
	if err := store.DeleteDaggerheartAdversary(ctx, "camp-1", "adv-1"); err != nil {
		t.Fatalf("DeleteDaggerheartAdversary() error = %v", err)
	}
	if err := store.DeleteDaggerheartAdversary(ctx, "camp-1", "adv-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("missing adversary delete error = %v, want ErrNotFound", err)
	}

	now := time.Now().UTC()
	entity1 := projectionstore.DaggerheartEnvironmentEntity{CampaignID: "camp-1", EnvironmentEntityID: "env-1", SessionID: "sess-1", SceneID: "scene-1", CreatedAt: now, UpdatedAt: now}
	entity2 := projectionstore.DaggerheartEnvironmentEntity{CampaignID: "camp-1", EnvironmentEntityID: "env-2", SessionID: "sess-1", SceneID: "scene-2", CreatedAt: now, UpdatedAt: now}
	if err := store.PutDaggerheartEnvironmentEntity(ctx, entity1); err != nil {
		t.Fatalf("PutDaggerheartEnvironmentEntity(1) error = %v", err)
	}
	if err := store.PutDaggerheartEnvironmentEntity(ctx, entity2); err != nil {
		t.Fatalf("PutDaggerheartEnvironmentEntity(2) error = %v", err)
	}
	if got, err := store.GetDaggerheartEnvironmentEntity(ctx, "camp-1", "env-1"); err != nil || got.EnvironmentEntityID != "env-1" {
		t.Fatalf("GetDaggerheartEnvironmentEntity() = %+v, %v", got, err)
	}
	if list, err := store.ListDaggerheartEnvironmentEntities(ctx, "camp-1", "sess-1", "scene-2"); err != nil || len(list) != 1 || list[0].EnvironmentEntityID != "env-2" {
		t.Fatalf("ListDaggerheartEnvironmentEntities() = %+v, %v", list, err)
	}
	if err := store.DeleteDaggerheartEnvironmentEntity(ctx, "camp-1", "env-1"); err != nil {
		t.Fatalf("DeleteDaggerheartEnvironmentEntity() error = %v", err)
	}
	if err := store.DeleteDaggerheartEnvironmentEntity(ctx, "camp-1", "env-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("missing environment entity delete error = %v, want ErrNotFound", err)
	}
}

func TestFakeDaggerheartStoreHonorsConfiguredErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	store := NewFakeDaggerheartStore()
	store.PutErr = wantErr
	store.GetErr = wantErr

	if err := store.PutDaggerheartCharacterProfile(context.Background(), projectionstore.DaggerheartCharacterProfile{CampaignID: "camp-1", CharacterID: "char-1"}); !errors.Is(err, wantErr) {
		t.Fatalf("PutDaggerheartCharacterProfile() error = %v, want %v", err, wantErr)
	}
	if _, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-1"); !errors.Is(err, wantErr) {
		t.Fatalf("GetDaggerheartCharacterProfile() error = %v, want %v", err, wantErr)
	}
	if _, err := store.ListDaggerheartCountdowns(context.Background(), "camp-1"); !errors.Is(err, wantErr) {
		t.Fatalf("ListDaggerheartCountdowns() error = %v, want %v", err, wantErr)
	}
	if _, err := store.ListDaggerheartAdversaries(context.Background(), "camp-1", ""); !errors.Is(err, wantErr) {
		t.Fatalf("ListDaggerheartAdversaries() error = %v, want %v", err, wantErr)
	}
	if _, err := store.ListDaggerheartEnvironmentEntities(context.Background(), "camp-1", "", ""); !errors.Is(err, wantErr) {
		t.Fatalf("ListDaggerheartEnvironmentEntities() error = %v, want %v", err, wantErr)
	}
}
