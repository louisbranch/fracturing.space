package bbolt

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
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

func TestActorStorePutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	actor := domain.Actor{
		ID:         "actor-123",
		CampaignID: "camp-456",
		Name:       "Alice",
		Kind:       domain.ActorKindPC,
		Notes:      "A brave warrior",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := store.PutActor(context.Background(), actor); err != nil {
		t.Fatalf("put actor: %v", err)
	}

	loaded, err := store.GetActor(context.Background(), "camp-456", "actor-123")
	if err != nil {
		t.Fatalf("get actor: %v", err)
	}
	if loaded.Name != actor.Name {
		t.Fatalf("expected name %q, got %q", actor.Name, loaded.Name)
	}
	if loaded.ID != actor.ID {
		t.Fatalf("expected id %q, got %q", actor.ID, loaded.ID)
	}
	if loaded.CampaignID != actor.CampaignID {
		t.Fatalf("expected campaign id %q, got %q", actor.CampaignID, loaded.CampaignID)
	}
	if loaded.Kind != actor.Kind {
		t.Fatalf("expected kind %v, got %v", actor.Kind, loaded.Kind)
	}
	if loaded.Notes != actor.Notes {
		t.Fatalf("expected notes %q, got %q", actor.Notes, loaded.Notes)
	}
	if !loaded.CreatedAt.Equal(now) {
		t.Fatalf("expected created_at %v, got %v", now, loaded.CreatedAt)
	}
	if !loaded.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated_at %v, got %v", now, loaded.UpdatedAt)
	}
}

func TestActorStoreGetNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetActor(context.Background(), "camp-123", "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestActorStorePutEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.PutActor(context.Background(), domain.Actor{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestActorStorePutCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	actor := domain.Actor{
		ID:         "actor-123",
		CampaignID: "camp-456",
		Name:       "Alice",
		Kind:       domain.ActorKindPC,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.PutActor(ctx, actor); err == nil {
		t.Fatal("expected error")
	}
}

func TestActorStoreGetEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetActor(context.Background(), "", "actor-123")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = store.GetActor(context.Background(), "camp-123", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestActorStoreGetCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.GetActor(ctx, "camp-123", "actor-123")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestActorStorePutNPC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	actor := domain.Actor{
		ID:         "actor-789",
		CampaignID: "camp-456",
		Name:       "Goblin",
		Kind:       domain.ActorKindNPC,
		Notes:      "A small creature",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := store.PutActor(context.Background(), actor); err != nil {
		t.Fatalf("put actor: %v", err)
	}

	loaded, err := store.GetActor(context.Background(), "camp-456", "actor-789")
	if err != nil {
		t.Fatalf("get actor: %v", err)
	}
	if loaded.Kind != domain.ActorKindNPC {
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
	if err := store.PutControlDefault(context.Background(), "camp-123", "actor-456", controller); err != nil {
		t.Fatalf("put control default: %v", err)
	}

	// Verify by reading directly from the bucket
	var loaded domain.ActorController
	err = store.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(controlDefaultBucket))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		key := controlDefaultKey("camp-123", "actor-456")
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
	if err := store.PutControlDefault(context.Background(), "camp-123", "actor-456", controller); err != nil {
		t.Fatalf("put control default: %v", err)
	}

	// Verify by reading directly from the bucket
	var loaded domain.ActorController
	err = store.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(controlDefaultBucket))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		key := controlDefaultKey("camp-123", "actor-456")
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
	if err := store.PutControlDefault(context.Background(), "camp-123", "actor-456", gmController); err != nil {
		t.Fatalf("put control default (GM): %v", err)
	}

	// Then overwrite with participant controller
	participantController, err := domain.NewParticipantController("participant-789")
	if err != nil {
		t.Fatalf("new participant controller: %v", err)
	}
	if err := store.PutControlDefault(context.Background(), "camp-123", "actor-456", participantController); err != nil {
		t.Fatalf("put control default (participant): %v", err)
	}

	// Verify the overwrite
	var loaded domain.ActorController
	err = store.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(controlDefaultBucket))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		key := controlDefaultKey("camp-123", "actor-456")
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
	if err := store.PutControlDefault(context.Background(), "", "actor-456", controller); err == nil {
		t.Fatal("expected error")
	}
}

func TestControlDefaultStorePutEmptyActorID(t *testing.T) {
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
	controller := domain.ActorController{}
	if err := store.PutControlDefault(context.Background(), "camp-123", "actor-456", controller); err == nil {
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
	if err := store.PutControlDefault(ctx, "camp-123", "actor-456", controller); err == nil {
		t.Fatal("expected error")
	}
}
