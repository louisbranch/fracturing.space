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
		PlayerCount: 4,
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
	if loaded.PlayerCount != campaign.PlayerCount {
		t.Fatalf("expected player count %d, got %d", campaign.PlayerCount, loaded.PlayerCount)
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
			PlayerCount: 2,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "camp-2",
			Name:        "B",
			GmMode:      domain.GmModeHuman,
			PlayerCount: 3,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "camp-3",
			Name:        "C",
			GmMode:      domain.GmModeHybrid,
			PlayerCount: 4,
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

func TestParticipantStorePutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
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
