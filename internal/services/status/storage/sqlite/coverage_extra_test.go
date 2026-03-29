package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/status/domain"
)

func TestOpenAndCloseGuards(t *testing.T) {
	t.Parallel()

	if _, err := Open(" "); err == nil {
		t.Fatal("Open(empty path) error = nil, want path validation error")
	}

	var nilStore *Store
	if err := nilStore.Close(); err != nil {
		t.Fatalf("(*Store)(nil).Close() error = %v, want nil", err)
	}

	store, err := Open(filepath.Join(t.TempDir(), "status.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	store.sqlDB = nil
	if err := store.Close(); err != nil {
		t.Fatalf("Close(nil db) error = %v, want nil", err)
	}
}

func TestStoreContextCancellationBranches(t *testing.T) {
	t.Parallel()

	store, err := Open(filepath.Join(t.TempDir(), "status.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.PutOverride(cancelledCtx, domain.Override{
		Service:    "game",
		Capability: "game.service",
		Status:     domain.StatusMaintenance,
		Reason:     domain.OverrideReasonMaintenance,
		Detail:     "planned downtime",
		SetAt:      time.Now().UTC(),
	}); err == nil {
		t.Fatal("PutOverride(cancelled ctx) error = nil, want context error")
	}

	if err := store.DeleteOverride(cancelledCtx, "game", "game.service"); err == nil {
		t.Fatal("DeleteOverride(cancelled ctx) error = nil, want context error")
	}

	if _, err := store.ListOverrides(cancelledCtx); err == nil {
		t.Fatal("ListOverrides(cancelled ctx) error = nil, want context error")
	}
}
