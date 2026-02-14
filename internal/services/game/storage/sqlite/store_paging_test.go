package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
)

func TestCampaignPaging(t *testing.T) {
	store := openTestStore(t)

	for _, id := range []string{"camp-1", "camp-2", "camp-3"} {
		if err := store.Put(context.Background(), campaign.Campaign{
			ID:        id,
			Name:      "Campaign",
			System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			Status:    campaign.CampaignStatusActive,
			GmMode:    campaign.GmModeHuman,
			CreatedAt: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		}); err != nil {
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
	if page.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	second, err := store.List(context.Background(), 2, page.NextPageToken)
	if err != nil {
		t.Fatalf("list campaigns page 2: %v", err)
	}
	if len(second.Campaigns) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(second.Campaigns))
	}
	if second.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %s", second.NextPageToken)
	}
}

func TestParticipantPaging(t *testing.T) {
	store := openTestStore(t)

	if err := store.Put(context.Background(), campaign.Campaign{
		ID:        "camp-1",
		Name:      "Campaign",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.CampaignStatusActive,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	for _, id := range []string{"p-1", "p-2", "p-3"} {
		if err := store.PutParticipant(context.Background(), participant.Participant{
			CampaignID:     "camp-1",
			ID:             id,
			Role:           participant.ParticipantRolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("put participant: %v", err)
		}
	}

	page, err := store.ListParticipants(context.Background(), "camp-1", 2, "")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(page.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(page.Participants))
	}
	if page.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	second, err := store.ListParticipants(context.Background(), "camp-1", 2, page.NextPageToken)
	if err != nil {
		t.Fatalf("list participants page 2: %v", err)
	}
	if len(second.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(second.Participants))
	}
	if second.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %s", second.NextPageToken)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "store.sqlite")
	store, err := OpenProjections(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}
