package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript/transcripttest"
	_ "modernc.org/sqlite"
)

func TestStoreContracts(t *testing.T) {
	t.Parallel()

	transcripttest.RunStoreContract(t, func(t *testing.T) transcripttest.Store {
		t.Helper()

		store, err := Open(filepath.Join(t.TempDir(), "play.sqlite"))
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		baseTime := time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC)
		var callCount int64
		store.now = func() time.Time {
			next := atomic.AddInt64(&callCount, 1)
			return baseTime.Add(time.Duration(next) * time.Second)
		}
		return store
	})
}

func TestOpenWith(t *testing.T) {
	t.Parallel()

	t.Run("creates parent dir and stores injected clock", func(t *testing.T) {
		t.Parallel()

		db := &fakeDBHandle{}
		now := time.Date(2026, time.March, 19, 12, 0, 0, 0, time.UTC)
		var (
			gotDirs         []string
			gotPath         string
			migrationsCalls int
		)
		store, err := openWith(filepath.Join("data", "play.sqlite"), storeOpeners{
			mkdirAll: func(path string, _ os.FileMode) error {
				gotDirs = append(gotDirs, path)
				return nil
			},
			openDB: func(path string) (databaseHandle, error) {
				gotPath = path
				return db, nil
			},
			applyMigrations: func(handle databaseHandle, clock func() time.Time) error {
				migrationsCalls++
				if handle != db {
					t.Fatalf("migration handle = %#v, want %#v", handle, db)
				}
				if clock() != now {
					t.Fatalf("clock() = %v, want %v", clock(), now)
				}
				return nil
			},
			now: func() time.Time { return now },
		})
		if err != nil {
			t.Fatalf("openWith() error = %v", err)
		}
		if gotPath != filepath.Join("data", "play.sqlite") {
			t.Fatalf("open path = %q", gotPath)
		}
		if !reflect.DeepEqual(gotDirs, []string{"data"}) {
			t.Fatalf("mkdir dirs = %#v, want %#v", gotDirs, []string{"data"})
		}
		if migrationsCalls != 1 {
			t.Fatalf("migration calls = %d, want 1", migrationsCalls)
		}
		if store.now() != now {
			t.Fatalf("store.now() = %v, want %v", store.now(), now)
		}
	})

	t.Run("migration failure closes db", func(t *testing.T) {
		t.Parallel()

		db := &fakeDBHandle{}
		_, err := openWith("play.sqlite", storeOpeners{
			mkdirAll: func(string, os.FileMode) error { return nil },
			openDB:   func(string) (databaseHandle, error) { return db, nil },
			applyMigrations: func(databaseHandle, func() time.Time) error {
				return errors.New("bad migration")
			},
			now: time.Now,
		})
		if err == nil || err.Error() != "apply play sqlite migrations: bad migration" {
			t.Fatalf("error = %v", err)
		}
		if db.closeCalls != 1 {
			t.Fatalf("close calls = %d, want 1", db.closeCalls)
		}
	})

	t.Run("rejects blank path before side effects", func(t *testing.T) {
		t.Parallel()

		_, err := openWith("   ", storeOpeners{
			mkdirAll: func(string, os.FileMode) error {
				t.Fatal("mkdirAll should not be called for blank path")
				return nil
			},
			openDB: func(string) (databaseHandle, error) {
				t.Fatal("openDB should not be called for blank path")
				return nil, nil
			},
			applyMigrations: func(databaseHandle, func() time.Time) error {
				t.Fatal("applyMigrations should not be called for blank path")
				return nil
			},
			now: time.Now,
		})
		if err == nil || err.Error() != "storage path is required" {
			t.Fatalf("error = %v, want %q", err, "storage path is required")
		}
	})

	t.Run("open db failure is wrapped", func(t *testing.T) {
		t.Parallel()

		_, err := openWith("play.sqlite", storeOpeners{
			mkdirAll: func(string, os.FileMode) error { return nil },
			openDB: func(string) (databaseHandle, error) {
				return nil, errors.New("open failed")
			},
			applyMigrations: func(databaseHandle, func() time.Time) error {
				t.Fatal("applyMigrations should not run after open failure")
				return nil
			},
			now: time.Now,
		})
		if err == nil || err.Error() != "open sqlite store: open failed" {
			t.Fatalf("error = %v, want %q", err, "open sqlite store: open failed")
		}
	})
}

func TestStoreGuardPaths(t *testing.T) {
	t.Parallel()

	var nilStore *Store
	if err := nilStore.Close(); err != nil {
		t.Fatalf("nil Close() error = %v", err)
	}

	store := &Store{}

	if _, err := store.LatestSequence(context.Background(), transcript.Scope{}); err == nil || err.Error() != "store is required" {
		t.Fatalf("LatestSequence() error = %v, want %q", err, "store is required")
	}
	if _, err := store.AppendMessage(context.Background(), transcript.AppendRequest{}); err == nil || err.Error() != "store is required" {
		t.Fatalf("AppendMessage() error = %v, want %q", err, "store is required")
	}
	if _, err := store.HistoryAfter(context.Background(), transcript.HistoryAfterQuery{}); err == nil || err.Error() != "store is required" {
		t.Fatalf("HistoryAfter() error = %v, want %q", err, "store is required")
	}
	if _, err := store.HistoryBefore(context.Background(), transcript.HistoryBeforeQuery{}); err == nil || err.Error() != "store is required" {
		t.Fatalf("HistoryBefore() error = %v, want %q", err, "store is required")
	}
}

func TestStoreBuildMessageUsesFallbackActorFields(t *testing.T) {
	t.Parallel()

	store := &Store{now: func() time.Time {
		return time.Date(2026, time.March, 19, 12, 0, 1, 0, time.UTC)
	}}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE transcript_messages (
			campaign_id TEXT NOT NULL,
			session_id TEXT NOT NULL,
			sequence_id INTEGER NOT NULL,
			message_id TEXT NOT NULL,
			sent_at_utc TEXT NOT NULL,
			participant_id TEXT NOT NULL,
			participant_name TEXT NOT NULL,
			body TEXT NOT NULL,
			client_message_id TEXT
		)`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer tx.Rollback()

	message, err := store.buildMessage(context.Background(), tx, transcript.AppendRequest{
		Scope: transcript.Scope{CampaignID: "c1", SessionID: "s1"},
		Actor: transcript.MessageActor{},
		Body:  "hello",
	})
	if err != nil {
		t.Fatalf("buildMessage() error = %v", err)
	}
	if message.SequenceID != 1 {
		t.Fatalf("sequence_id = %d, want 1", message.SequenceID)
	}
	if message.Actor.ParticipantID != "participant" || message.Actor.Name != "participant" {
		t.Fatalf("actor = %#v", message.Actor)
	}
	if message.SentAt != "2026-03-19T12:00:01Z" {
		t.Fatalf("sent_at = %q, want %q", message.SentAt, "2026-03-19T12:00:01Z")
	}
}

func TestRetryClassifiers(t *testing.T) {
	t.Parallel()

	if !isAppendRetryable(errors.New("database is locked")) {
		t.Fatal("expected locked error string to be retryable")
	}
	if !isAppendRetryable(errors.New("UNIQUE CONSTRAINT FAILED: transcript_messages.client_message_id")) {
		t.Fatal("expected unique constraint string to be retryable")
	}
	if isAppendRetryable(errors.New("validation failed")) {
		t.Fatal("expected non-sqlite error to be non-retryable")
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE dedupe (value TEXT PRIMARY KEY)`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO dedupe (value) VALUES ('a')`); err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	_, err = db.Exec(`INSERT INTO dedupe (value) VALUES ('a')`)
	if err == nil {
		t.Fatal("duplicate insert error = nil, want non-nil")
	}
	if !isUniqueConstraintError(err) {
		t.Fatalf("expected sqlite duplicate insert to be classified as unique constraint: %v", err)
	}
}

type fakeDBHandle struct {
	closeCalls int
}

func (f *fakeDBHandle) BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error) {
	return nil, errors.New("unused")
}

func (f *fakeDBHandle) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func (f *fakeDBHandle) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unused")
}

func (f *fakeDBHandle) Close() error {
	f.closeCalls++
	return nil
}
