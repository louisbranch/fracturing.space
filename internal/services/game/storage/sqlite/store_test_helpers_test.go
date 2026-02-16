package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/migrations"
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

// openTestCombinedStore opens a projections store with a keyring so both
// projections tables and the event integrity path are available in
// ApplyRollOutcome.
func openTestCombinedStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "combined.sqlite")
	store, err := OpenProjections(path)
	if err != nil {
		t.Fatalf("open combined store: %v", err)
	}
	// Attach the keyring so event appends within ApplyRollOutcome work.
	store.keyring = testKeyring(t)
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		_ = store.Close()
		t.Fatalf("build registries: %v", err)
	}
	store.eventRegistry = registries.Events

	// Run events migrations on the same database so the events tables exist.
	if err := runMigrations(store.sqlDB, migrations.EventsFS, "events"); err != nil {
		_ = store.Close()
		t.Fatalf("run events migrations: %v", err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close combined store: %v", err)
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
