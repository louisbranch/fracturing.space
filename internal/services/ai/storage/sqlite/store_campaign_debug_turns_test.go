package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/debugtrace"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestCampaignDebugTurnStoreRoundTripAndPagination(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	ctx := context.Background()
	now := time.Unix(1713333333, 0).UTC()

	turns := []debugtrace.Turn{
		{
			ID:          "turn-2",
			CampaignID:  "campaign-1",
			SessionID:   "session-1",
			TurnToken:   "token-2",
			Provider:    provider.OpenAI,
			Model:       "gpt-5.4",
			Status:      debugtrace.StatusSucceeded,
			Usage:       provider.Usage{InputTokens: 20, OutputTokens: 10, TotalTokens: 30},
			StartedAt:   now.Add(time.Minute),
			UpdatedAt:   now.Add(2 * time.Minute),
			CompletedAt: ptrTime(now.Add(3 * time.Minute)),
			EntryCount:  2,
		},
		{
			ID:         "turn-1",
			CampaignID: "campaign-1",
			SessionID:  "session-1",
			TurnToken:  "token-1",
			Provider:   provider.OpenAI,
			Model:      "gpt-5.4-mini",
			Status:     debugtrace.StatusRunning,
			StartedAt:  now,
			UpdatedAt:  now.Add(time.Second),
			EntryCount: 1,
		},
		{
			ID:         "turn-other-session",
			CampaignID: "campaign-1",
			SessionID:  "session-2",
			StartedAt:  now,
			UpdatedAt:  now,
		},
	}
	for _, turn := range turns {
		if err := store.PutCampaignDebugTurn(ctx, turn); err != nil {
			t.Fatalf("PutCampaignDebugTurn(%s): %v", turn.ID, err)
		}
	}

	entries := []debugtrace.Entry{
		{
			TurnID:    "turn-2",
			Sequence:  1,
			Kind:      debugtrace.EntryKindModelResponse,
			Payload:   "Narration",
			CreatedAt: now.Add(90 * time.Second),
		},
		{
			TurnID:           "turn-2",
			Sequence:         2,
			Kind:             debugtrace.EntryKindToolCall,
			ToolName:         "scene_create",
			Payload:          `{"name":"Harbor"}`,
			PayloadTruncated: true,
			CallID:           "call-1",
			ResponseID:       "resp-1",
			IsError:          true,
			CreatedAt:        now.Add(100 * time.Second),
			Usage:            provider.Usage{InputTokens: 2, OutputTokens: 3, TotalTokens: 5},
		},
	}
	for _, entry := range entries {
		if err := store.PutCampaignDebugTurnEntry(ctx, entry); err != nil {
			t.Fatalf("PutCampaignDebugTurnEntry(%d): %v", entry.Sequence, err)
		}
	}

	page, err := store.ListCampaignDebugTurns(ctx, "campaign-1", "session-1", 1, "")
	if err != nil {
		t.Fatalf("ListCampaignDebugTurns first page: %v", err)
	}
	if len(page.Turns) != 1 || page.Turns[0].ID != "turn-2" || page.NextPageToken == "" {
		t.Fatalf("first page = %#v", page)
	}

	nextPage, err := store.ListCampaignDebugTurns(ctx, "campaign-1", "session-1", 1, page.NextPageToken)
	if err != nil {
		t.Fatalf("ListCampaignDebugTurns second page: %v", err)
	}
	if len(nextPage.Turns) != 1 || nextPage.Turns[0].ID != "turn-1" || nextPage.NextPageToken != "" {
		t.Fatalf("second page = %#v", nextPage)
	}

	gotTurn, err := store.GetCampaignDebugTurn(ctx, "campaign-1", "turn-2")
	if err != nil {
		t.Fatalf("GetCampaignDebugTurn: %v", err)
	}
	if gotTurn.Status != debugtrace.StatusSucceeded || gotTurn.Provider != provider.OpenAI || gotTurn.CompletedAt == nil {
		t.Fatalf("gotTurn = %#v", gotTurn)
	}

	gotEntries, err := store.ListCampaignDebugTurnEntries(ctx, "turn-2")
	if err != nil {
		t.Fatalf("ListCampaignDebugTurnEntries: %v", err)
	}
	if len(gotEntries) != 2 || gotEntries[1].ToolName != "scene_create" || !gotEntries[1].PayloadTruncated || !gotEntries[1].IsError {
		t.Fatalf("gotEntries = %#v", gotEntries)
	}
}

func TestCampaignDebugTurnStoreValidationAndTokenHelpers(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	ctx := context.Background()
	now := time.Unix(1713333333, 0).UTC()

	if _, _, err := parseDebugTurnPageToken("bad-token"); err == nil {
		t.Fatal("parseDebugTurnPageToken error = nil, want invalid token")
	}

	token := debugTurnPageToken(12345, " turn-1 ")
	millis, turnID, err := parseDebugTurnPageToken(token)
	if err != nil {
		t.Fatalf("parseDebugTurnPageToken: %v", err)
	}
	if millis != 12345 || turnID != "turn-1" {
		t.Fatalf("parsed token = (%d, %q)", millis, turnID)
	}

	if err := store.PutCampaignDebugTurn(ctx, debugtrace.Turn{}); err == nil {
		t.Fatal("PutCampaignDebugTurn error = nil, want validation failure")
	}
	if err := store.PutCampaignDebugTurnEntry(ctx, debugtrace.Entry{}); err == nil {
		t.Fatal("PutCampaignDebugTurnEntry error = nil, want validation failure")
	}
	if _, err := store.ListCampaignDebugTurns(ctx, "", "session-1", 10, ""); err == nil {
		t.Fatal("ListCampaignDebugTurns error = nil, want validation failure")
	}

	if _, err := store.GetCampaignDebugTurn(ctx, "campaign-1", "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetCampaignDebugTurn error = %v, want ErrNotFound", err)
	}

	if err := store.PutCampaignDebugTurn(ctx, debugtrace.Turn{
		ID:         "turn-1",
		CampaignID: "campaign-1",
		SessionID:  "session-1",
		StartedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("PutCampaignDebugTurn: %v", err)
	}
	if entries, err := store.ListCampaignDebugTurnEntries(ctx, "turn-1"); err != nil || len(entries) != 0 {
		t.Fatalf("ListCampaignDebugTurnEntries = (%#v, %v), want empty slice", entries, err)
	}
}

func TestCampaignDebugTurnStoreContextAndConfigurationErrors(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := openTempStore(t)
	if err := store.PutCampaignDebugTurn(ctx, debugtrace.Turn{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("PutCampaignDebugTurn canceled error = %v, want context.Canceled", err)
	}
	if err := store.PutCampaignDebugTurnEntry(ctx, debugtrace.Entry{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("PutCampaignDebugTurnEntry canceled error = %v, want context.Canceled", err)
	}
	if _, err := store.ListCampaignDebugTurns(ctx, "campaign-1", "session-1", 10, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListCampaignDebugTurns canceled error = %v, want context.Canceled", err)
	}
	if _, err := store.GetCampaignDebugTurn(ctx, "campaign-1", "turn-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetCampaignDebugTurn canceled error = %v, want context.Canceled", err)
	}
	if _, err := store.ListCampaignDebugTurnEntries(ctx, "turn-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListCampaignDebugTurnEntries canceled error = %v, want context.Canceled", err)
	}

	var nilStore *Store
	if err := nilStore.PutCampaignDebugTurn(context.Background(), debugtrace.Turn{
		ID:         "turn-1",
		CampaignID: "campaign-1",
		SessionID:  "session-1",
		StartedAt:  time.Unix(1713333333, 0).UTC(),
		UpdatedAt:  time.Unix(1713333333, 0).UTC(),
	}); err == nil {
		t.Fatal("PutCampaignDebugTurn nil store error = nil, want configuration failure")
	}
	if err := nilStore.PutCampaignDebugTurnEntry(context.Background(), debugtrace.Entry{
		TurnID:    "turn-1",
		Sequence:  1,
		CreatedAt: time.Unix(1713333333, 0).UTC(),
	}); err == nil {
		t.Fatal("PutCampaignDebugTurnEntry nil store error = nil, want configuration failure")
	}
	if _, err := nilStore.ListCampaignDebugTurns(context.Background(), "campaign-1", "session-1", 10, ""); err == nil {
		t.Fatal("ListCampaignDebugTurns nil store error = nil, want configuration failure")
	}
	if _, err := nilStore.GetCampaignDebugTurn(context.Background(), "campaign-1", "turn-1"); err == nil {
		t.Fatal("GetCampaignDebugTurn nil store error = nil, want configuration failure")
	}
	if _, err := nilStore.ListCampaignDebugTurnEntries(context.Background(), "turn-1"); err == nil {
		t.Fatal("ListCampaignDebugTurnEntries nil store error = nil, want configuration failure")
	}
}
