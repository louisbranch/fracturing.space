package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func openTestProjectionsStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "projections.sqlite")
	store, err := OpenProjections(path)
	if err != nil {
		t.Fatalf("open projections store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close projections store: %v", err)
		}
	})
	return store
}

func TestSaveAndGetProjectionWatermark(t *testing.T) {
	store := openTestProjectionsStore(t)
	ctx := context.Background()

	// Save a watermark.
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	wm := storage.ProjectionWatermark{
		CampaignID: "camp-1",
		AppliedSeq: 42,
		UpdatedAt:  now,
	}
	if err := store.SaveProjectionWatermark(ctx, wm); err != nil {
		t.Fatalf("save watermark: %v", err)
	}

	// Get it back.
	got, err := store.GetProjectionWatermark(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get watermark: %v", err)
	}
	if got.CampaignID != "camp-1" {
		t.Fatalf("campaign_id = %s, want camp-1", got.CampaignID)
	}
	if got.AppliedSeq != 42 {
		t.Fatalf("applied_seq = %d, want 42", got.AppliedSeq)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, now)
	}
}

func TestGetProjectionWatermark_NotFound(t *testing.T) {
	store := openTestProjectionsStore(t)
	ctx := context.Background()

	_, err := store.GetProjectionWatermark(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent watermark")
	}
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSaveProjectionWatermark_Upsert(t *testing.T) {
	store := openTestProjectionsStore(t)
	ctx := context.Background()

	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	// First save.
	if err := store.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
		CampaignID: "camp-1",
		AppliedSeq: 10,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("save watermark: %v", err)
	}

	// Update to higher seq.
	if err := store.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
		CampaignID: "camp-1",
		AppliedSeq: 20,
		UpdatedAt:  later,
	}); err != nil {
		t.Fatalf("upsert watermark: %v", err)
	}

	got, err := store.GetProjectionWatermark(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get watermark: %v", err)
	}
	if got.AppliedSeq != 20 {
		t.Fatalf("applied_seq = %d, want 20", got.AppliedSeq)
	}
	if !got.UpdatedAt.Equal(later) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, later)
	}
}

func TestListProjectionWatermarks(t *testing.T) {
	store := openTestProjectionsStore(t)
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Empty list.
	wms, err := store.ListProjectionWatermarks(ctx)
	if err != nil {
		t.Fatalf("list watermarks: %v", err)
	}
	if len(wms) != 0 {
		t.Fatalf("expected empty list, got %d", len(wms))
	}

	// Add two watermarks.
	for _, camp := range []string{"camp-1", "camp-2"} {
		if err := store.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
			CampaignID: camp,
			AppliedSeq: 10,
			UpdatedAt:  now,
		}); err != nil {
			t.Fatalf("save watermark %s: %v", camp, err)
		}
	}

	wms, err = store.ListProjectionWatermarks(ctx)
	if err != nil {
		t.Fatalf("list watermarks: %v", err)
	}
	if len(wms) != 2 {
		t.Fatalf("expected 2 watermarks, got %d", len(wms))
	}
}
