package bbolt

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/campaign/domain"
	sessiondomain "github.com/louisbranch/fracturing.space/internal/session/domain"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"go.etcd.io/bbolt"
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
		ID:               "camp-123",
		Name:             "Snowbound",
		GmMode:           domain.GmModeAI,
		ParticipantCount: 4,
		CharacterCount:   2,
		GmFear:           7,
		ThemePrompt:      "ice and steel",
		CreatedAt:        now,
		UpdatedAt:        now,
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
	if loaded.ParticipantCount != campaign.ParticipantCount {
		t.Fatalf("expected participant count %d, got %d", campaign.ParticipantCount, loaded.ParticipantCount)
	}
	if loaded.CharacterCount != campaign.CharacterCount {
		t.Fatalf("expected character count %d, got %d", campaign.CharacterCount, loaded.CharacterCount)
	}
	if loaded.GmFear != campaign.GmFear {
		t.Fatalf("expected gm fear %d, got %d", campaign.GmFear, loaded.GmFear)
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
			ID:               "camp-1",
			Name:             "A",
			GmMode:           domain.GmModeAI,
			ParticipantCount: 2,
			CharacterCount:   1,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "camp-2",
			Name:             "B",
			GmMode:           domain.GmModeHuman,
			ParticipantCount: 3,
			CharacterCount:   2,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "camp-3",
			Name:             "C",
			GmMode:           domain.GmModeHybrid,
			ParticipantCount: 4,
			CharacterCount:   3,
			CreatedAt:        now,
			UpdatedAt:        now,
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

func TestParticipantStorePutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Create campaign first since PutParticipant now requires it to exist
	campaign := domain.Campaign{
		ID:               "camp-456",
		Name:             "Test Campaign",
		GmMode:           domain.GmModeHuman,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	participant := domain.Participant{
		ID:          "part-123",
		CampaignID:  "camp-456",
		DisplayName: "Alice",
		Role:        domain.ParticipantRolePlayer,
		Controller:  domain.ControllerHuman,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutParticipant(context.Background(), participant); err != nil {
		t.Fatalf("put participant: %v", err)
	}

	// Verify participant count was incremented
	updatedCampaign, err := store.Get(context.Background(), "camp-456")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if updatedCampaign.ParticipantCount != 1 {
		t.Fatalf("expected participant count 1, got %d", updatedCampaign.ParticipantCount)
	}

	loaded, err := store.GetParticipant(context.Background(), "camp-456", "part-123")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if loaded.DisplayName != participant.DisplayName {
		t.Fatalf("expected display name %q, got %q", participant.DisplayName, loaded.DisplayName)
	}
	if loaded.ID != participant.ID {
		t.Fatalf("expected id %q, got %q", participant.ID, loaded.ID)
	}
	if loaded.CampaignID != participant.CampaignID {
		t.Fatalf("expected campaign id %q, got %q", participant.CampaignID, loaded.CampaignID)
	}
	if loaded.Role != participant.Role {
		t.Fatalf("expected role %v, got %v", participant.Role, loaded.Role)
	}
	if loaded.Controller != participant.Controller {
		t.Fatalf("expected controller %v, got %v", participant.Controller, loaded.Controller)
	}
	if !loaded.CreatedAt.Equal(now) {
		t.Fatalf("expected created_at %v, got %v", now, loaded.CreatedAt)
	}
	if !loaded.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated_at %v, got %v", now, loaded.UpdatedAt)
	}
}

func TestParticipantStorePutIncrementsCounterOnlyForNewRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaign := domain.Campaign{
		ID:               "camp-789",
		Name:             "Test Campaign",
		GmMode:           domain.GmModeHuman,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	participant := domain.Participant{
		ID:          "part-999",
		CampaignID:  "camp-789",
		DisplayName: "Alice",
		Role:        domain.ParticipantRolePlayer,
		Controller:  domain.ControllerHuman,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// First put should increment counter
	if err := store.PutParticipant(context.Background(), participant); err != nil {
		t.Fatalf("put participant: %v", err)
	}

	campaignAfterFirst, err := store.Get(context.Background(), "camp-789")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaignAfterFirst.ParticipantCount != 1 {
		t.Fatalf("expected participant count 1 after first put, got %d", campaignAfterFirst.ParticipantCount)
	}

	// Update participant display name
	participant.DisplayName = "Alice Updated"
	if err := store.PutParticipant(context.Background(), participant); err != nil {
		t.Fatalf("put participant update: %v", err)
	}

	// Counter should not increment on update
	campaignAfterUpdate, err := store.Get(context.Background(), "camp-789")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaignAfterUpdate.ParticipantCount != 1 {
		t.Fatalf("expected participant count 1 after update, got %d", campaignAfterUpdate.ParticipantCount)
	}

	// Verify participant was updated
	loaded, err := store.GetParticipant(context.Background(), "camp-789", "part-999")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if loaded.DisplayName != "Alice Updated" {
		t.Fatalf("expected updated display name, got %q", loaded.DisplayName)
	}
}

func TestParticipantStoreGetNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetParticipant(context.Background(), "camp-123", "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestParticipantStorePutEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.PutParticipant(context.Background(), domain.Participant{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestParticipantStorePutCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.PutParticipant(ctx, domain.Participant{ID: "part-123", CampaignID: "camp-123"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestParticipantStoreGetEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetParticipant(context.Background(), "", "part-123")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParticipantStoreGetCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.GetParticipant(ctx, "camp-123", "part-123")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParticipantStoreListByCampaign(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Create campaigns first since PutParticipant now requires them to exist
	campaigns := []domain.Campaign{
		{
			ID:               "camp-123",
			Name:             "Campaign 1",
			GmMode:           domain.GmModeHuman,
			ParticipantCount: 0,
			CharacterCount:   0,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "camp-456",
			Name:             "Campaign 2",
			GmMode:           domain.GmModeHuman,
			ParticipantCount: 0,
			CharacterCount:   0,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
	for _, campaign := range campaigns {
		if err := store.Put(context.Background(), campaign); err != nil {
			t.Fatalf("put campaign: %v", err)
		}
	}

	participants := []domain.Participant{
		{
			ID:          "part-1",
			CampaignID:  "camp-123",
			DisplayName: "Alice",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "part-2",
			CampaignID:  "camp-123",
			DisplayName: "Bob",
			Role:        domain.ParticipantRoleGM,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "part-3",
			CampaignID:  "camp-456",
			DisplayName: "Charlie",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerAI,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, participant := range participants {
		if err := store.PutParticipant(context.Background(), participant); err != nil {
			t.Fatalf("put participant: %v", err)
		}
	}

	list, err := store.ListParticipantsByCampaign(context.Background(), "camp-123")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(list))
	}
	foundAlice := false
	foundBob := false
	for _, p := range list {
		if p.ID == "part-1" && p.DisplayName == "Alice" {
			foundAlice = true
		}
		if p.ID == "part-2" && p.DisplayName == "Bob" {
			foundBob = true
		}
	}
	if !foundAlice {
		t.Fatal("expected to find Alice")
	}
	if !foundBob {
		t.Fatal("expected to find Bob")
	}
}

func TestParticipantStoreListByCampaignEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	list, err := store.ListParticipantsByCampaign(context.Background(), "camp-123")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 participants, got %d", len(list))
	}
}

func TestParticipantStoreListByCampaignCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.ListParticipantsByCampaign(ctx, "camp-123")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParticipantStoreListPagination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Create campaign first since PutParticipant now requires it to exist
	campaign := domain.Campaign{
		ID:               "camp-123",
		Name:             "Test Campaign",
		GmMode:           domain.GmModeHuman,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	participants := []domain.Participant{
		{
			ID:          "part-1",
			CampaignID:  "camp-123",
			DisplayName: "Alice",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "part-2",
			CampaignID:  "camp-123",
			DisplayName: "Bob",
			Role:        domain.ParticipantRoleGM,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "part-3",
			CampaignID:  "camp-123",
			DisplayName: "Charlie",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerAI,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, participant := range participants {
		if err := store.PutParticipant(context.Background(), participant); err != nil {
			t.Fatalf("put participant: %v", err)
		}
	}

	page, err := store.ListParticipants(context.Background(), "camp-123", 2, "")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(page.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(page.Participants))
	}
	if page.Participants[0].ID != "part-1" {
		t.Fatalf("expected first id part-1, got %q", page.Participants[0].ID)
	}
	if page.Participants[1].ID != "part-2" {
		t.Fatalf("expected second id part-2, got %q", page.Participants[1].ID)
	}
	expectedToken := "camp-123/part-2"
	if page.NextPageToken != expectedToken {
		t.Fatalf("expected next page token %q, got %q", expectedToken, page.NextPageToken)
	}

	page, err = store.ListParticipants(context.Background(), "camp-123", 2, page.NextPageToken)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(page.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(page.Participants))
	}
	if page.Participants[0].ID != "part-3" {
		t.Fatalf("expected id part-3, got %q", page.Participants[0].ID)
	}
	if page.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %q", page.NextPageToken)
	}
}

func TestParticipantStoreListPaginationPrefixFiltering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Create campaigns first since PutParticipant now requires them to exist
	campaigns := []domain.Campaign{
		{
			ID:               "camp-123",
			Name:             "Campaign 1",
			GmMode:           domain.GmModeHuman,
			ParticipantCount: 0,
			CharacterCount:   0,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "camp-456",
			Name:             "Campaign 2",
			GmMode:           domain.GmModeHuman,
			ParticipantCount: 0,
			CharacterCount:   0,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
	for _, campaign := range campaigns {
		if err := store.Put(context.Background(), campaign); err != nil {
			t.Fatalf("put campaign: %v", err)
		}
	}

	participants := []domain.Participant{
		{
			ID:          "part-1",
			CampaignID:  "camp-123",
			DisplayName: "Alice",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "part-2",
			CampaignID:  "camp-456",
			DisplayName: "Bob",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "part-3",
			CampaignID:  "camp-123",
			DisplayName: "Charlie",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, participant := range participants {
		if err := store.PutParticipant(context.Background(), participant); err != nil {
			t.Fatalf("put participant: %v", err)
		}
	}

	page, err := store.ListParticipants(context.Background(), "camp-123", 10, "")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(page.Participants) != 2 {
		t.Fatalf("expected 2 participants for camp-123, got %d", len(page.Participants))
	}
	foundAlice := false
	foundCharlie := false
	for _, p := range page.Participants {
		if p.ID == "part-1" && p.DisplayName == "Alice" {
			foundAlice = true
		}
		if p.ID == "part-3" && p.DisplayName == "Charlie" {
			foundCharlie = true
		}
		if p.CampaignID != "camp-123" {
			t.Fatalf("expected campaign id camp-123, got %q", p.CampaignID)
		}
	}
	if !foundAlice {
		t.Fatal("expected to find Alice")
	}
	if !foundCharlie {
		t.Fatal("expected to find Charlie")
	}
}

func TestParticipantStoreListEmptyPageToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Create campaign first since PutParticipant now requires it to exist
	campaign := domain.Campaign{
		ID:               "camp-123",
		Name:             "Test Campaign",
		GmMode:           domain.GmModeHuman,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	participant := domain.Participant{
		ID:          "part-1",
		CampaignID:  "camp-123",
		DisplayName: "Alice",
		Role:        domain.ParticipantRolePlayer,
		Controller:  domain.ControllerHuman,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutParticipant(context.Background(), participant); err != nil {
		t.Fatalf("put participant: %v", err)
	}

	page, err := store.ListParticipants(context.Background(), "camp-123", 10, "")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(page.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(page.Participants))
	}
	if page.Participants[0].ID != "part-1" {
		t.Fatalf("expected id part-1, got %q", page.Participants[0].ID)
	}
	if page.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %q", page.NextPageToken)
	}
}

func TestParticipantStoreListEmptyResult(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	page, err := store.ListParticipants(context.Background(), "camp-123", 10, "")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(page.Participants) != 0 {
		t.Fatalf("expected 0 participants, got %d", len(page.Participants))
	}
	if page.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %q", page.NextPageToken)
	}
}

func TestParticipantStoreListInvalidPageSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.ListParticipants(context.Background(), "camp-123", 0, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParticipantStoreListEmptyCampaignID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.ListParticipants(context.Background(), "", 10, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParticipantStoreListCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.ListParticipants(ctx, "camp-123", 10, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParticipantStoreListExactPageSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Create campaign first since PutParticipant now requires it to exist
	campaign := domain.Campaign{
		ID:               "camp-123",
		Name:             "Test Campaign",
		GmMode:           domain.GmModeHuman,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	participants := []domain.Participant{
		{
			ID:          "part-1",
			CampaignID:  "camp-123",
			DisplayName: "Alice",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "part-2",
			CampaignID:  "camp-123",
			DisplayName: "Bob",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, participant := range participants {
		if err := store.PutParticipant(context.Background(), participant); err != nil {
			t.Fatalf("put participant: %v", err)
		}
	}

	page, err := store.ListParticipants(context.Background(), "camp-123", 2, "")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(page.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(page.Participants))
	}
	if page.NextPageToken != "" {
		t.Fatalf("expected empty next page token when exactly page size matches, got %q", page.NextPageToken)
	}
}

func TestParticipantStoreListPageTokenResume(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Create campaign first since PutParticipant now requires it to exist
	campaign := domain.Campaign{
		ID:               "camp-123",
		Name:             "Test Campaign",
		GmMode:           domain.GmModeHuman,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	participants := []domain.Participant{
		{
			ID:          "part-1",
			CampaignID:  "camp-123",
			DisplayName: "Alice",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "part-2",
			CampaignID:  "camp-123",
			DisplayName: "Bob",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "part-3",
			CampaignID:  "camp-123",
			DisplayName: "Charlie",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, participant := range participants {
		if err := store.PutParticipant(context.Background(), participant); err != nil {
			t.Fatalf("put participant: %v", err)
		}
	}

	firstPage, err := store.ListParticipants(context.Background(), "camp-123", 1, "")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(firstPage.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(firstPage.Participants))
	}
	if firstPage.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	secondPage, err := store.ListParticipants(context.Background(), "camp-123", 1, firstPage.NextPageToken)
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(secondPage.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(secondPage.Participants))
	}
	if secondPage.Participants[0].ID == firstPage.Participants[0].ID {
		t.Fatal("expected different participant on second page")
	}
}

func TestCharacterStorePutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Create campaign first since PutCharacter now requires it to exist
	campaign := domain.Campaign{
		ID:               "camp-456",
		Name:             "Test Campaign",
		GmMode:           domain.GmModeHuman,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	character := domain.Character{
		ID:         "character-123",
		CampaignID: "camp-456",
		Name:       "Alice",
		Kind:       domain.CharacterKindPC,
		Notes:      "A brave warrior",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := store.PutCharacter(context.Background(), character); err != nil {
		t.Fatalf("put character: %v", err)
	}

	// Verify character count was incremented
	updatedCampaign, err := store.Get(context.Background(), "camp-456")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if updatedCampaign.CharacterCount != 1 {
		t.Fatalf("expected character count 1, got %d", updatedCampaign.CharacterCount)
	}

	loaded, err := store.GetCharacter(context.Background(), "camp-456", "character-123")
	if err != nil {
		t.Fatalf("get character: %v", err)
	}
	if loaded.Name != character.Name {
		t.Fatalf("expected name %q, got %q", character.Name, loaded.Name)
	}
	if loaded.ID != character.ID {
		t.Fatalf("expected id %q, got %q", character.ID, loaded.ID)
	}
	if loaded.CampaignID != character.CampaignID {
		t.Fatalf("expected campaign id %q, got %q", character.CampaignID, loaded.CampaignID)
	}
	if loaded.Kind != character.Kind {
		t.Fatalf("expected kind %v, got %v", character.Kind, loaded.Kind)
	}
	if loaded.Notes != character.Notes {
		t.Fatalf("expected notes %q, got %q", character.Notes, loaded.Notes)
	}
	if !loaded.CreatedAt.Equal(now) {
		t.Fatalf("expected created_at %v, got %v", now, loaded.CreatedAt)
	}
	if !loaded.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated_at %v, got %v", now, loaded.UpdatedAt)
	}
}

func TestCharacterStorePutIncrementsCounterOnlyForNewRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaign := domain.Campaign{
		ID:               "camp-999",
		Name:             "Test Campaign",
		GmMode:           domain.GmModeHuman,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	character := domain.Character{
		ID:         "character-888",
		CampaignID: "camp-999",
		Name:       "Hero",
		Kind:       domain.CharacterKindPC,
		Notes:      "A brave warrior",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// First put should increment counter
	if err := store.PutCharacter(context.Background(), character); err != nil {
		t.Fatalf("put character: %v", err)
	}

	campaignAfterFirst, err := store.Get(context.Background(), "camp-999")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaignAfterFirst.CharacterCount != 1 {
		t.Fatalf("expected character count 1 after first put, got %d", campaignAfterFirst.CharacterCount)
	}

	// Update character notes
	character.Notes = "An even braver warrior"
	if err := store.PutCharacter(context.Background(), character); err != nil {
		t.Fatalf("put character update: %v", err)
	}

	// Counter should not increment on update
	campaignAfterUpdate, err := store.Get(context.Background(), "camp-999")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaignAfterUpdate.CharacterCount != 1 {
		t.Fatalf("expected character count 1 after update, got %d", campaignAfterUpdate.CharacterCount)
	}

	// Verify character was updated
	loaded, err := store.GetCharacter(context.Background(), "camp-999", "character-888")
	if err != nil {
		t.Fatalf("get character: %v", err)
	}
	if loaded.Notes != "An even braver warrior" {
		t.Fatalf("expected updated notes, got %q", loaded.Notes)
	}
}

func TestCharacterStoreGetNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetCharacter(context.Background(), "camp-123", "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestCharacterStorePutEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.PutCharacter(context.Background(), domain.Character{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterStorePutCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	character := domain.Character{
		ID:         "character-123",
		CampaignID: "camp-456",
		Name:       "Alice",
		Kind:       domain.CharacterKindPC,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.PutCharacter(ctx, character); err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterStoreGetEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetCharacter(context.Background(), "", "character-123")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = store.GetCharacter(context.Background(), "camp-123", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterStoreGetCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.GetCharacter(ctx, "camp-123", "character-123")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterStorePutNPC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Create campaign first since PutCharacter now requires it to exist
	campaign := domain.Campaign{
		ID:               "camp-456",
		Name:             "Test Campaign",
		GmMode:           domain.GmModeHuman,
		ParticipantCount: 0,
		CharacterCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.Put(context.Background(), campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	character := domain.Character{
		ID:         "character-789",
		CampaignID: "camp-456",
		Name:       "Goblin",
		Kind:       domain.CharacterKindNPC,
		Notes:      "A small creature",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := store.PutCharacter(context.Background(), character); err != nil {
		t.Fatalf("put character: %v", err)
	}

	loaded, err := store.GetCharacter(context.Background(), "camp-456", "character-789")
	if err != nil {
		t.Fatalf("get character: %v", err)
	}
	if loaded.Kind != domain.CharacterKindNPC {
		t.Fatalf("expected kind NPC, got %v", loaded.Kind)
	}
}

func TestControlDefaultStorePutGM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	controller := domain.NewGmController()
	if err := store.PutControlDefault(context.Background(), "camp-123", "character-456", controller); err != nil {
		t.Fatalf("put control default: %v", err)
	}

	// Verify by reading directly from the bucket
	var loaded domain.CharacterController
	err = store.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(controlDefaultBucket))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		key := controlDefaultKey("camp-123", "character-456")
		payload := bucket.Get(key)
		if payload == nil {
			return storage.ErrNotFound
		}
		return json.Unmarshal(payload, &loaded)
	})
	if err != nil {
		t.Fatalf("read control default: %v", err)
	}
	if !loaded.IsGM {
		t.Fatal("expected controller to be GM")
	}
	if loaded.ParticipantID != "" {
		t.Fatalf("expected empty participant ID, got %q", loaded.ParticipantID)
	}
}

func TestControlDefaultStorePutParticipant(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	controller, err := domain.NewParticipantController("participant-789")
	if err != nil {
		t.Fatalf("new participant controller: %v", err)
	}
	if err := store.PutControlDefault(context.Background(), "camp-123", "character-456", controller); err != nil {
		t.Fatalf("put control default: %v", err)
	}

	// Verify by reading directly from the bucket
	var loaded domain.CharacterController
	err = store.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(controlDefaultBucket))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		key := controlDefaultKey("camp-123", "character-456")
		payload := bucket.Get(key)
		if payload == nil {
			return storage.ErrNotFound
		}
		return json.Unmarshal(payload, &loaded)
	})
	if err != nil {
		t.Fatalf("read control default: %v", err)
	}
	if loaded.IsGM {
		t.Fatal("expected controller to be participant")
	}
	if loaded.ParticipantID != "participant-789" {
		t.Fatalf("expected participant ID participant-789, got %q", loaded.ParticipantID)
	}
}

func TestControlDefaultStorePutOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// First, set GM controller
	gmController := domain.NewGmController()
	if err := store.PutControlDefault(context.Background(), "camp-123", "character-456", gmController); err != nil {
		t.Fatalf("put control default (GM): %v", err)
	}

	// Then overwrite with participant controller
	participantController, err := domain.NewParticipantController("participant-789")
	if err != nil {
		t.Fatalf("new participant controller: %v", err)
	}
	if err := store.PutControlDefault(context.Background(), "camp-123", "character-456", participantController); err != nil {
		t.Fatalf("put control default (participant): %v", err)
	}

	// Verify the overwrite
	var loaded domain.CharacterController
	err = store.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(controlDefaultBucket))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		key := controlDefaultKey("camp-123", "character-456")
		payload := bucket.Get(key)
		if payload == nil {
			return storage.ErrNotFound
		}
		return json.Unmarshal(payload, &loaded)
	})
	if err != nil {
		t.Fatalf("read control default: %v", err)
	}
	if loaded.IsGM {
		t.Fatal("expected controller to be participant after overwrite")
	}
	if loaded.ParticipantID != "participant-789" {
		t.Fatalf("expected participant ID participant-789, got %q", loaded.ParticipantID)
	}
}

func TestControlDefaultStorePutEmptyCampaignID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	controller := domain.NewGmController()
	if err := store.PutControlDefault(context.Background(), "", "character-456", controller); err == nil {
		t.Fatal("expected error")
	}
}

func TestControlDefaultStorePutEmptyCharacterID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	controller := domain.NewGmController()
	if err := store.PutControlDefault(context.Background(), "camp-123", "", controller); err == nil {
		t.Fatal("expected error")
	}
}

func TestControlDefaultStorePutInvalidController(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// Invalid controller: neither IsGM nor ParticipantID set
	controller := domain.CharacterController{}
	if err := store.PutControlDefault(context.Background(), "camp-123", "character-456", controller); err == nil {
		t.Fatal("expected error")
	}
}

func TestControlDefaultStorePutCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	controller := domain.NewGmController()
	if err := store.PutControlDefault(ctx, "camp-123", "character-456", controller); err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterProfileStorePutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	profile := domain.CharacterProfile{
		CampaignID:      "camp-123",
		CharacterID:     "char-456",
		Traits:          map[string]int{"agility": 1},
		HpMax:           6,
		StressMax:       4,
		Evasion:         3,
		MajorThreshold:  2,
		SevereThreshold: 5,
	}

	if err := store.PutCharacterProfile(context.Background(), profile); err != nil {
		t.Fatalf("put character profile: %v", err)
	}

	loaded, err := store.GetCharacterProfile(context.Background(), "camp-123", "char-456")
	if err != nil {
		t.Fatalf("get character profile: %v", err)
	}
	if loaded.HpMax != profile.HpMax {
		t.Fatalf("expected hp max %d, got %d", profile.HpMax, loaded.HpMax)
	}
	if loaded.Traits["agility"] != 1 {
		t.Fatalf("expected agility 1, got %d", loaded.Traits["agility"])
	}
}

func TestCharacterProfileStoreGetNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetCharacterProfile(context.Background(), "camp-123", "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestCharacterProfileStorePutEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.PutCharacterProfile(context.Background(), domain.CharacterProfile{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterProfileStorePutCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	profile := domain.CharacterProfile{CampaignID: "camp-123", CharacterID: "char-456"}
	if err := store.PutCharacterProfile(ctx, profile); err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterProfileStoreGetEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetCharacterProfile(context.Background(), "", "char-456")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = store.GetCharacterProfile(context.Background(), "camp-123", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterProfileStoreGetCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.GetCharacterProfile(ctx, "camp-123", "char-456")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterStateStorePutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	state := domain.CharacterState{
		CampaignID:  "camp-123",
		CharacterID: "char-456",
		Hope:        2,
		Stress:      1,
		Hp:          5,
	}

	if err := store.PutCharacterState(context.Background(), state); err != nil {
		t.Fatalf("put character state: %v", err)
	}

	loaded, err := store.GetCharacterState(context.Background(), "camp-123", "char-456")
	if err != nil {
		t.Fatalf("get character state: %v", err)
	}
	if loaded.Hope != 2 || loaded.Hp != 5 {
		t.Fatalf("expected hope 2 hp 5, got hope %d hp %d", loaded.Hope, loaded.Hp)
	}
}

func TestCharacterStateStoreGetNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetCharacterState(context.Background(), "camp-123", "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestCharacterStateStorePutEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.PutCharacterState(context.Background(), domain.CharacterState{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterStateStorePutCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state := domain.CharacterState{CampaignID: "camp-123", CharacterID: "char-456"}
	if err := store.PutCharacterState(ctx, state); err == nil {
		t.Fatal("expected error")
	}
}

func TestCharacterStateStoreGetEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetCharacterState(context.Background(), "", "char-456")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = store.GetCharacterState(context.Background(), "camp-123", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestApplyRollOutcomeUpdatesState ensures outcome application updates state and event.
func TestApplyRollOutcomeUpdatesState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	campaign := domain.Campaign{ID: "camp-1", Name: "Test", GmMode: domain.GmModeHuman}
	if err := store.Put(ctx, campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	state := domain.CharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hope: 5, Stress: 1, Hp: 4}
	if err := store.PutCharacterState(ctx, state); err != nil {
		t.Fatalf("put character state: %v", err)
	}
	profile := domain.CharacterProfile{CampaignID: "camp-1", CharacterID: "char-1", StressMax: 6}
	if err := store.PutCharacterProfile(ctx, profile); err != nil {
		t.Fatalf("put character profile: %v", err)
	}

	result, err := store.ApplyRollOutcome(ctx, storage.RollOutcomeApplyInput{
		CampaignID:           "camp-1",
		SessionID:            "sess-1",
		RollSeq:              5,
		Targets:              []string{"char-1"},
		RequiresComplication: false,
		RequestID:            "req-1",
		InvocationID:         "inv-1",
		ParticipantID:        "part-1",
		CharacterID:          "char-1",
		EventTimestamp:       time.Date(2026, 1, 26, 13, 0, 0, 0, time.UTC),
		CharacterDeltas: []storage.RollOutcomeDelta{
			{CharacterID: "char-1", HopeDelta: 1, StressDelta: -1},
		},
	})
	if err != nil {
		t.Fatalf("apply roll outcome: %v", err)
	}
	if len(result.UpdatedCharacterStates) != 1 {
		t.Fatalf("expected 1 updated state, got %d", len(result.UpdatedCharacterStates))
	}
	updated := result.UpdatedCharacterStates[0]
	if updated.Hope != 6 || updated.Stress != 0 {
		t.Fatalf("expected hope 6 stress 0, got hope %d stress %d", updated.Hope, updated.Stress)
	}
	if len(result.AppliedChanges) != 2 {
		t.Fatalf("expected 2 applied changes, got %d", len(result.AppliedChanges))
	}

	stored, err := store.GetCharacterState(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get character state: %v", err)
	}
	if stored.Hope != 6 || stored.Stress != 0 {
		t.Fatalf("expected stored hope 6 stress 0, got hope %d stress %d", stored.Hope, stored.Stress)
	}

	events, err := store.ListSessionEvents(ctx, "sess-1", 0, 10)
	if err != nil {
		t.Fatalf("list session events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != sessiondomain.SessionEventTypeOutcomeApplied {
		t.Fatalf("expected OUTCOME_APPLIED event, got %s", events[0].Type)
	}
}

// TestApplyRollOutcomeRejectsAlreadyApplied ensures duplicate apply is rejected.
func TestApplyRollOutcomeRejectsAlreadyApplied(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	campaign := domain.Campaign{ID: "camp-2", Name: "Test", GmMode: domain.GmModeHuman}
	if err := store.Put(ctx, campaign); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	state := domain.CharacterState{CampaignID: "camp-2", CharacterID: "char-2", Hope: 2, Stress: 1, Hp: 4}
	if err := store.PutCharacterState(ctx, state); err != nil {
		t.Fatalf("put character state: %v", err)
	}
	profile := domain.CharacterProfile{CampaignID: "camp-2", CharacterID: "char-2", StressMax: 6}
	if err := store.PutCharacterProfile(ctx, profile); err != nil {
		t.Fatalf("put character profile: %v", err)
	}

	_, err = store.ApplyRollOutcome(ctx, storage.RollOutcomeApplyInput{
		CampaignID: "camp-2",
		SessionID:  "sess-2",
		RollSeq:    9,
		Targets:    []string{"char-2"},
		CharacterDeltas: []storage.RollOutcomeDelta{
			{CharacterID: "char-2", HopeDelta: 1},
		},
	})
	if err != nil {
		t.Fatalf("apply roll outcome: %v", err)
	}

	_, err = store.ApplyRollOutcome(ctx, storage.RollOutcomeApplyInput{
		CampaignID: "camp-2",
		SessionID:  "sess-2",
		RollSeq:    9,
		Targets:    []string{"char-2"},
		CharacterDeltas: []storage.RollOutcomeDelta{
			{CharacterID: "char-2", HopeDelta: 1},
		},
	})
	if !errors.Is(err, sessiondomain.ErrOutcomeAlreadyApplied) {
		t.Fatalf("expected outcome already applied error, got %v", err)
	}
}

func TestCharacterStateStoreGetCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.GetCharacterState(ctx, "camp-123", "char-456")
	if err == nil {
		t.Fatal("expected error")
	}
}
