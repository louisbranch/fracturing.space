package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/campaignartifact"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestCampaignArtifactRoundTripAndList(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 2, 0, 0, 0, time.UTC)

	first := campaignartifact.Artifact{
		CampaignID: "campaign-1",
		Path:       "memory.md",
		Content:    "Session notes",
		ReadOnly:   false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	second := campaignartifact.Artifact{
		CampaignID: "campaign-1",
		Path:       "skills.md",
		Content:    "GM skills",
		ReadOnly:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	for _, record := range []campaignartifact.Artifact{first, second} {
		if err := store.PutCampaignArtifact(context.Background(), record); err != nil {
			t.Fatalf("PutCampaignArtifact(%s) error = %v", record.Path, err)
		}
	}

	got, err := store.GetCampaignArtifact(context.Background(), "campaign-1", "memory.md")
	if err != nil {
		t.Fatalf("GetCampaignArtifact() error = %v", err)
	}
	if got.Content != "Session notes" || got.ReadOnly {
		t.Fatalf("unexpected artifact: %+v", got)
	}

	items, err := store.ListCampaignArtifacts(context.Background(), "campaign-1")
	if err != nil {
		t.Fatalf("ListCampaignArtifacts() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if items[0].Path != "memory.md" || items[1].Path != "skills.md" {
		t.Fatalf("artifact order = [%s %s], want [memory.md skills.md]", items[0].Path, items[1].Path)
	}
}

func TestGetCampaignArtifactReturnsNotFound(t *testing.T) {
	store := openTempStore(t)

	_, err := store.GetCampaignArtifact(context.Background(), "campaign-1", "missing.md")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetCampaignArtifact() error = %v, want storage.ErrNotFound", err)
	}
}
