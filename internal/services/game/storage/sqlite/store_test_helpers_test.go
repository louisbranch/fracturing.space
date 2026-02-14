package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
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
	path := filepath.Join(t.TempDir(), "events.sqlite")
	store, err := OpenEvents(path, testKeyring(t))
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

func seedCharacter(t *testing.T, store *Store, campaignID, charID, name string, kind character.CharacterKind, now time.Time) character.Character {
	t.Helper()
	c := character.Character{
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

func seedSession(t *testing.T, store *Store, campaignID, sessID string, now time.Time) session.Session {
	t.Helper()
	s := session.Session{
		ID:         sessID,
		CampaignID: campaignID,
		Name:       "Session " + sessID,
		Status:     session.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
	}
	if err := store.PutSession(context.Background(), s); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return s
}
