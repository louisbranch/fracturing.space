package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/status/domain"
	_ "modernc.org/sqlite"
)

func TestStore_override_lifecycle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	// Put override.
	ov := domain.Override{
		Service:    "game",
		Capability: "game.service",
		Status:     domain.StatusMaintenance,
		Reason:     domain.OverrideReasonMaintenance,
		Detail:     "planned downtime",
		SetAt:      now,
	}
	if err := store.PutOverride(ctx, ov); err != nil {
		t.Fatalf("PutOverride: %v", err)
	}

	// List overrides.
	overrides, err := store.ListOverrides(ctx)
	if err != nil {
		t.Fatalf("ListOverrides: %v", err)
	}
	if len(overrides) != 1 {
		t.Fatalf("got %d overrides, want 1", len(overrides))
	}
	got := overrides[0]
	if got.Service != "game" || got.Capability != "game.service" {
		t.Fatalf("override = %s/%s, want game/game.service", got.Service, got.Capability)
	}
	if got.Status != domain.StatusMaintenance {
		t.Fatalf("status = %v, want MAINTENANCE", got.Status)
	}
	if got.Detail != "planned downtime" {
		t.Fatalf("detail = %q, want planned downtime", got.Detail)
	}

	// Upsert override.
	ov.Detail = "extended"
	if err := store.PutOverride(ctx, ov); err != nil {
		t.Fatalf("PutOverride upsert: %v", err)
	}
	overrides, _ = store.ListOverrides(ctx)
	if len(overrides) != 1 {
		t.Fatalf("got %d overrides after upsert, want 1", len(overrides))
	}
	if overrides[0].Detail != "extended" {
		t.Fatalf("detail = %q, want extended", overrides[0].Detail)
	}

	// Delete override.
	if err := store.DeleteOverride(ctx, "game", "game.service"); err != nil {
		t.Fatalf("DeleteOverride: %v", err)
	}
	overrides, _ = store.ListOverrides(ctx)
	if len(overrides) != 0 {
		t.Fatalf("got %d overrides after delete, want 0", len(overrides))
	}
}

func TestStore_multiple_overrides(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now()

	for _, ov := range []domain.Override{
		{Service: "game", Capability: "cap.a", Status: domain.StatusMaintenance, SetAt: now},
		{Service: "game", Capability: "cap.b", Status: domain.StatusDegraded, SetAt: now},
		{Service: "web", Capability: "cap.c", Status: domain.StatusUnavailable, SetAt: now},
	} {
		if err := store.PutOverride(ctx, ov); err != nil {
			t.Fatalf("PutOverride: %v", err)
		}
	}

	overrides, err := store.ListOverrides(ctx)
	if err != nil {
		t.Fatalf("ListOverrides: %v", err)
	}
	if len(overrides) != 3 {
		t.Fatalf("got %d overrides, want 3", len(overrides))
	}
}
