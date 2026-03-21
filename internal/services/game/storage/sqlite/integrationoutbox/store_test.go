package integrationoutbox_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	sqliteeventjournal "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/eventjournal"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/integrationoutbox"
)

func TestIntegrationOutbox_LeaseRetryAndSucceed(t *testing.T) {
	store, _ := openTestIntegrationOutboxStore(t)
	now := time.Date(2026, 3, 9, 12, 10, 0, 0, time.UTC)
	err := store.EnqueueIntegrationOutboxEvent(context.Background(), storage.IntegrationOutboxEvent{
		ID:            "evt-1",
		EventType:     "test.outbox_event",
		PayloadJSON:   `{"key":"value"}`,
		DedupeKey:     "test-dedupe-evt-1",
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("enqueue integration outbox event: %v", err)
	}

	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 10, now, time.Minute)
	if err != nil {
		t.Fatalf("lease integration outbox events: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased len = %d, want 1", len(leased))
	}
	if leased[0].Status != storage.IntegrationOutboxStatusLeased {
		t.Fatalf("leased status = %q, want %q", leased[0].Status, storage.IntegrationOutboxStatusLeased)
	}

	retryAt := now.Add(30 * time.Second)
	if err := store.MarkIntegrationOutboxRetry(context.Background(), "evt-1", "worker-1", retryAt, "temporary"); err != nil {
		t.Fatalf("mark retry: %v", err)
	}
	retried, err := store.GetIntegrationOutboxEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("get retried event: %v", err)
	}
	if retried.Status != storage.IntegrationOutboxStatusPending {
		t.Fatalf("status after retry = %q, want %q", retried.Status, storage.IntegrationOutboxStatusPending)
	}
	if retried.AttemptCount != 1 {
		t.Fatalf("attempt count after retry = %d, want 1", retried.AttemptCount)
	}

	leased, err = store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 10, retryAt, time.Minute)
	if err != nil {
		t.Fatalf("lease integration outbox events again: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased len after retry = %d, want 1", len(leased))
	}

	if err := store.MarkIntegrationOutboxSucceeded(context.Background(), "evt-1", "worker-1", retryAt.Add(time.Second)); err != nil {
		t.Fatalf("mark succeeded: %v", err)
	}
	done, err := store.GetIntegrationOutboxEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("get succeeded event: %v", err)
	}
	if done.Status != storage.IntegrationOutboxStatusSucceeded {
		t.Fatalf("status after success = %q, want %q", done.Status, storage.IntegrationOutboxStatusSucceeded)
	}
	if done.ProcessedAt == nil {
		t.Fatal("expected processed at to be set")
	}
}

func TestIntegrationOutbox_LeaseAndMarkDead(t *testing.T) {
	store, _ := openTestIntegrationOutboxStore(t)
	now := time.Date(2026, 3, 9, 12, 20, 0, 0, time.UTC)
	err := store.EnqueueIntegrationOutboxEvent(context.Background(), storage.IntegrationOutboxEvent{
		ID:            "evt-dead",
		EventType:     "test.dead_event",
		PayloadJSON:   `{"key":"value"}`,
		DedupeKey:     "test-dedupe-evt-dead",
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("enqueue integration outbox event: %v", err)
	}

	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 10, now, time.Minute)
	if err != nil {
		t.Fatalf("lease integration outbox events: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased len = %d, want 1", len(leased))
	}

	processedAt := now.Add(time.Minute)
	if err := store.MarkIntegrationOutboxDead(context.Background(), "evt-dead", "worker-1", "permanent failure", processedAt); err != nil {
		t.Fatalf("mark dead: %v", err)
	}
	done, err := store.GetIntegrationOutboxEvent(context.Background(), "evt-dead")
	if err != nil {
		t.Fatalf("get dead event: %v", err)
	}
	if done.Status != storage.IntegrationOutboxStatusDead {
		t.Fatalf("status after dead = %q, want %q", done.Status, storage.IntegrationOutboxStatusDead)
	}
	if done.ProcessedAt == nil || !done.ProcessedAt.Equal(processedAt) {
		t.Fatalf("processed at = %v, want %v", done.ProcessedAt, processedAt)
	}
}

func openTestIntegrationOutboxStore(t *testing.T) (*integrationoutbox.Store, *sqliteeventjournal.Store) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.sqlite")
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	root, err := sqliteeventjournal.Open(path, testKeyring(t), registries.Events)
	if err != nil {
		t.Fatalf("open events store: %v", err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Fatalf("close events store: %v", err)
		}
	})
	store := integrationoutbox.Bind(root.DB())
	if store == nil {
		t.Fatal("expected integration outbox store")
	}
	return store, root
}

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
