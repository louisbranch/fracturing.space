package daggerheartprojection_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sqlitecoreprojection "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/coreprojection"
)

type testStore struct {
	root       *sqlitecoreprojection.Store
	projection projectionstore.Store
}

func openTestStore(t *testing.T) *testStore {
	t.Helper()

	path := filepath.Join(t.TempDir(), "store.sqlite")
	root, err := sqlitecoreprojection.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})

	projection := root.DaggerheartProjectionStore()
	if projection == nil {
		t.Fatal("expected Daggerheart projection backend")
	}

	return &testStore{
		root:       root,
		projection: projection,
	}
}

func seedCampaign(t *testing.T, store *testStore, id string, now time.Time) storage.CampaignRecord {
	t.Helper()

	record := storage.CampaignRecord{
		ID:        id,
		Name:      "Campaign",
		Locale:    "en-US",
		System:    bridge.SystemIDDaggerheart,
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.root.Put(context.Background(), record); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	return record
}

func seedCharacter(t *testing.T, store *testStore, campaignID, charID, name string, kind character.Kind, now time.Time) storage.CharacterRecord {
	t.Helper()

	record := storage.CharacterRecord{
		ID:         charID,
		CampaignID: campaignID,
		Name:       name,
		Kind:       kind,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := store.root.PutCharacter(context.Background(), record); err != nil {
		t.Fatalf("seed character: %v", err)
	}
	return record
}

func (s *testStore) PutDaggerheartCharacterProfile(ctx context.Context, profile projectionstore.DaggerheartCharacterProfile) error {
	return s.projection.PutDaggerheartCharacterProfile(ctx, profile)
}

func (s *testStore) GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	return s.projection.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
}

func (s *testStore) ListDaggerheartCharacterProfiles(ctx context.Context, campaignID string, pageSize int, pageToken string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	return s.projection.ListDaggerheartCharacterProfiles(ctx, campaignID, pageSize, pageToken)
}

func (s *testStore) DeleteDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) error {
	return s.projection.DeleteDaggerheartCharacterProfile(ctx, campaignID, characterID)
}

func (s *testStore) PutDaggerheartCharacterState(ctx context.Context, state projectionstore.DaggerheartCharacterState) error {
	return s.projection.PutDaggerheartCharacterState(ctx, state)
}

func (s *testStore) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	return s.projection.GetDaggerheartCharacterState(ctx, campaignID, characterID)
}

func (s *testStore) PutDaggerheartSnapshot(ctx context.Context, snapshot projectionstore.DaggerheartSnapshot) error {
	return s.projection.PutDaggerheartSnapshot(ctx, snapshot)
}

func (s *testStore) GetDaggerheartSnapshot(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	return s.projection.GetDaggerheartSnapshot(ctx, campaignID)
}

func (s *testStore) PutDaggerheartCountdown(ctx context.Context, countdown projectionstore.DaggerheartCountdown) error {
	return s.projection.PutDaggerheartCountdown(ctx, countdown)
}

func (s *testStore) GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	return s.projection.GetDaggerheartCountdown(ctx, campaignID, countdownID)
}

func (s *testStore) ListDaggerheartCountdowns(ctx context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
	return s.projection.ListDaggerheartCountdowns(ctx, campaignID)
}

func (s *testStore) DeleteDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) error {
	return s.projection.DeleteDaggerheartCountdown(ctx, campaignID, countdownID)
}

func (s *testStore) PutDaggerheartAdversary(ctx context.Context, adversary projectionstore.DaggerheartAdversary) error {
	return s.projection.PutDaggerheartAdversary(ctx, adversary)
}

func (s *testStore) GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	return s.projection.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
}

func (s *testStore) ListDaggerheartAdversaries(ctx context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
	return s.projection.ListDaggerheartAdversaries(ctx, campaignID, sessionID)
}

func (s *testStore) DeleteDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) error {
	return s.projection.DeleteDaggerheartAdversary(ctx, campaignID, adversaryID)
}
