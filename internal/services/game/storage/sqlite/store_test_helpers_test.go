package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	sqliteeventjournal "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/eventjournal"
	sqliteprojectionapplyoutbox "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/projectionapplyoutbox"
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

func openTestEventsStore(t *testing.T) *sqliteeventjournal.Store {
	t.Helper()
	return openTestRawEventsStore(t, false)
}

type testEventStoreWithOutbox struct {
	*sqliteeventjournal.Store
	outbox *sqliteprojectionapplyoutbox.Store
}

func (s *testEventStoreWithOutbox) ProcessProjectionApplyOutbox(ctx context.Context, now time.Time, limit int, apply func(context.Context, event.Event) error) (int, error) {
	return s.outbox.ProcessProjectionApplyOutbox(ctx, now, limit, apply)
}

func (s *testEventStoreWithOutbox) ProcessProjectionApplyOutboxShadow(ctx context.Context, now time.Time, limit int) (int, error) {
	return s.outbox.ProcessProjectionApplyOutboxShadow(ctx, now, limit)
}

func (s *testEventStoreWithOutbox) GetProjectionApplyOutboxSummary(ctx context.Context) (storage.ProjectionApplyOutboxSummary, error) {
	return s.outbox.GetProjectionApplyOutboxSummary(ctx)
}

func (s *testEventStoreWithOutbox) ListProjectionApplyOutboxRows(ctx context.Context, status string, limit int) ([]storage.ProjectionApplyOutboxEntry, error) {
	return s.outbox.ListProjectionApplyOutboxRows(ctx, status, limit)
}

func (s *testEventStoreWithOutbox) RequeueProjectionApplyOutboxRow(ctx context.Context, campaignID string, seq uint64, now time.Time) (bool, error) {
	return s.outbox.RequeueProjectionApplyOutboxRow(ctx, campaignID, seq, now)
}

func (s *testEventStoreWithOutbox) RequeueProjectionApplyOutboxDeadRows(ctx context.Context, limit int, now time.Time) (int, error) {
	return s.outbox.RequeueProjectionApplyOutboxDeadRows(ctx, limit, now)
}

func openTestEventsStoreWithOutbox(t *testing.T, outboxEnabled bool) *testEventStoreWithOutbox {
	t.Helper()
	store := openTestRawEventsStore(t, outboxEnabled)
	outbox := store.ProjectionApplyOutboxStore()
	bound, ok := outbox.(*sqliteprojectionapplyoutbox.Store)
	if !ok || bound == nil {
		t.Fatal("expected projection apply outbox store")
	}
	return &testEventStoreWithOutbox{
		Store:  store,
		outbox: bound,
	}
}

func openTestRawEventsStore(t *testing.T, outboxEnabled bool) *sqliteeventjournal.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.sqlite")
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	store, err := sqliteeventjournal.Open(
		path,
		testKeyring(t),
		registries.Events,
		sqliteeventjournal.WithProjectionApplyOutboxEnabled(outboxEnabled),
	)
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
