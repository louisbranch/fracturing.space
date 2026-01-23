package bbolt

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
)

func TestCampaignStorePutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaign := domain.Campaign{
		ID:          "camp-123",
		Name:        "Snowbound",
		GmMode:      domain.GmModeAI,
		PlayerSlots: 4,
		ThemePrompt: "ice and steel",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	loaded, err := store.Get(context.Background(), "camp-123")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if loaded.Name != campaign.Name {
		t.Fatalf("expected name %q, got %q", campaign.Name, loaded.Name)
	}
	if loaded.ID != campaign.ID {
		t.Fatalf("expected id %q, got %q", campaign.ID, loaded.ID)
	}
	if loaded.GmMode != campaign.GmMode {
		t.Fatalf("expected gm mode %v, got %v", campaign.GmMode, loaded.GmMode)
	}
	if loaded.PlayerSlots != campaign.PlayerSlots {
		t.Fatalf("expected player slots %d, got %d", campaign.PlayerSlots, loaded.PlayerSlots)
	}
	if loaded.ThemePrompt != campaign.ThemePrompt {
		t.Fatalf("expected theme prompt %q, got %q", campaign.ThemePrompt, loaded.ThemePrompt)
	}
	if !loaded.CreatedAt.Equal(now) {
		t.Fatalf("expected created_at %v, got %v", now, loaded.CreatedAt)
	}
	if !loaded.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated_at %v, got %v", now, loaded.UpdatedAt)
	}
}

func TestCampaignStoreGetNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.Get(context.Background(), "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestCampaignStorePutEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.Put(context.Background(), domain.Campaign{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCampaignStorePutCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Put(ctx, domain.Campaign{ID: "camp-123"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCampaignStoreGetEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCampaignStoreGetCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.Get(ctx, "camp-123")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCampaignStoreListPagination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaigns := []domain.Campaign{
		{
			ID:          "camp-1",
			Name:        "A",
			GmMode:      domain.GmModeAI,
			PlayerSlots: 2,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "camp-2",
			Name:        "B",
			GmMode:      domain.GmModeHuman,
			PlayerSlots: 3,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "camp-3",
			Name:        "C",
			GmMode:      domain.GmModeHybrid,
			PlayerSlots: 4,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, campaign := range campaigns {
		if err := store.Put(context.Background(), campaign); err != nil {
			t.Fatalf("put campaign: %v", err)
		}
	}

	page, err := store.List(context.Background(), 2, "")
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	if len(page.Campaigns) != 2 {
		t.Fatalf("expected 2 campaigns, got %d", len(page.Campaigns))
	}
	if page.Campaigns[0].ID != "camp-1" {
		t.Fatalf("expected first id camp-1, got %q", page.Campaigns[0].ID)
	}
	if page.NextPageToken != "camp-2" {
		t.Fatalf("expected next page token camp-2, got %q", page.NextPageToken)
	}

	page, err = store.List(context.Background(), 2, page.NextPageToken)
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	if len(page.Campaigns) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(page.Campaigns))
	}
	if page.Campaigns[0].ID != "camp-3" {
		t.Fatalf("expected id camp-3, got %q", page.Campaigns[0].ID)
	}
	if page.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %q", page.NextPageToken)
	}
}

func TestCampaignStoreListInvalidPageSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.List(context.Background(), 0, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCampaignStoreListCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.List(ctx, 1, "")
	if err == nil {
		t.Fatal("expected error")
	}
}
