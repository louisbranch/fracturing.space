package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
)

func testKeyring(t *testing.T) *integrity.Keyring {
	t.Helper()
	keyring, err := integrity.NewKeyring(
		map[string][]byte{"test-key-1": []byte("0123456789abcdef0123456789abcdef")},
		"test-key-1",
	)
	if err != nil {
		t.Fatalf("create test keyring: %v", err)
	}
	return keyring
}

func openTestEventsStore(t *testing.T) *Store {
	t.Helper()
	return openTestEventsStoreWithOutbox(t, false)
}

func openTestEventsStoreWithOutbox(t *testing.T, outboxEnabled bool) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.sqlite")
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	store, err := OpenEvents(path, testKeyring(t), registries.Events, WithProjectionApplyOutboxEnabled(outboxEnabled))
	if err != nil {
		t.Fatalf("open events store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close events store: %v", err)
		}
	})
	return store
}

func openTestContentStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "content.sqlite")
	store, err := OpenContent(path)
	if err != nil {
		t.Fatalf("open content store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close content store: %v", err)
		}
	})
	return store
}

func seedCharacter(t *testing.T, store *Store, campaignID, charID, name string, kind character.Kind, now time.Time) storage.CharacterRecord {
	t.Helper()
	c := storage.CharacterRecord{
		ID:         charID,
		CampaignID: campaignID,
		Name:       name,
		Kind:       kind,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := store.PutCharacter(context.Background(), c); err != nil {
		t.Fatalf("seed character: %v", err)
	}
	return c
}

func seedSession(t *testing.T, store *Store, campaignID, sessID string, now time.Time) storage.SessionRecord {
	t.Helper()
	s := storage.SessionRecord{
		ID:         sessID,
		CampaignID: campaignID,
		Name:       "Session " + sessID,
		Status:     session.StatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
	}
	if err := store.PutSession(context.Background(), s); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return s
}
