package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/debugtrace"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

type debugTraceStoreStub struct {
	mu sync.Mutex

	turns   map[string]debugtrace.Turn
	entries map[string][]debugtrace.Entry

	listPage    debugtrace.Page
	listErr     error
	getErr      error
	putTurnErr  func(debugtrace.Turn) error
	putEntryErr func(debugtrace.Entry) error

	turnWrites  []debugtrace.Turn
	entryWrites []debugtrace.Entry
}

func newDebugTraceStoreStub() *debugTraceStoreStub {
	return &debugTraceStoreStub{
		turns:   make(map[string]debugtrace.Turn),
		entries: make(map[string][]debugtrace.Entry),
	}
}

func (s *debugTraceStoreStub) PutCampaignDebugTurn(_ context.Context, turn debugtrace.Turn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.putTurnErr != nil {
		if err := s.putTurnErr(turn); err != nil {
			return err
		}
	}
	s.turns[turn.ID] = turn
	s.turnWrites = append(s.turnWrites, turn)
	return nil
}

func (s *debugTraceStoreStub) PutCampaignDebugTurnEntry(_ context.Context, entry debugtrace.Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.putEntryErr != nil {
		if err := s.putEntryErr(entry); err != nil {
			return err
		}
	}
	s.entries[entry.TurnID] = append(s.entries[entry.TurnID], entry)
	s.entryWrites = append(s.entryWrites, entry)
	return nil
}

func (s *debugTraceStoreStub) ListCampaignDebugTurns(_ context.Context, campaignID string, sessionID string, pageSize int, pageToken string) (debugtrace.Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listErr != nil {
		return debugtrace.Page{}, s.listErr
	}
	if len(s.listPage.Turns) != 0 || s.listPage.NextPageToken != "" {
		return s.listPage, nil
	}
	page := debugtrace.Page{Turns: make([]debugtrace.Turn, 0, len(s.turns))}
	for _, turn := range s.turns {
		if turn.CampaignID == campaignID && turn.SessionID == sessionID {
			page.Turns = append(page.Turns, turn)
		}
	}
	if len(page.Turns) > pageSize {
		page.Turns = page.Turns[:pageSize]
		page.NextPageToken = pageToken
	}
	return page, nil
}

func (s *debugTraceStoreStub) GetCampaignDebugTurn(_ context.Context, _ string, turnID string) (debugtrace.Turn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return debugtrace.Turn{}, s.getErr
	}
	turn, ok := s.turns[turnID]
	if !ok {
		return debugtrace.Turn{}, storage.ErrNotFound
	}
	return turn, nil
}

func (s *debugTraceStoreStub) ListCampaignDebugTurnEntries(_ context.Context, turnID string) ([]debugtrace.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]debugtrace.Entry(nil), s.entries[turnID]...), nil
}

func TestCampaignDebugServiceListGetAndSubscribe(t *testing.T) {
	t.Parallel()

	store := newDebugTraceStoreStub()
	broker := NewCampaignDebugUpdateBroker()
	service, err := NewCampaignDebugService(CampaignDebugServiceConfig{
		DebugTraceStore: store,
		UpdateBroker:    broker,
	})
	if err != nil {
		t.Fatalf("NewCampaignDebugService: %v", err)
	}

	startedAt := time.Unix(1711111111, 0).UTC()
	store.turns["turn-1"] = debugtrace.Turn{
		ID:         "turn-1",
		CampaignID: "campaign-1",
		SessionID:  "session-1",
		Model:      "gpt-5.4",
		Status:     debugtrace.StatusRunning,
		StartedAt:  startedAt,
		UpdatedAt:  startedAt,
	}
	store.entries["turn-1"] = []debugtrace.Entry{{
		TurnID:    "turn-1",
		Sequence:  1,
		Kind:      debugtrace.EntryKindToolCall,
		ToolName:  "scene_create",
		CreatedAt: startedAt,
	}}

	page, err := service.ListCampaignDebugTurns(context.Background(), ListCampaignDebugTurnsInput{
		CampaignID: "campaign-1",
		SessionID:  "session-1",
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("ListCampaignDebugTurns: %v", err)
	}
	if len(page.Turns) != 1 || page.Turns[0].ID != "turn-1" {
		t.Fatalf("page = %#v", page)
	}

	turn, err := service.GetCampaignDebugTurn(context.Background(), GetCampaignDebugTurnInput{
		CampaignID: "campaign-1",
		TurnID:     "turn-1",
	})
	if err != nil {
		t.Fatalf("GetCampaignDebugTurn: %v", err)
	}
	if turn.Turn.ID != "turn-1" || len(turn.Entries) != 1 || turn.Entries[0].ToolName != "scene_create" {
		t.Fatalf("turn = %#v", turn)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, unsubscribe, err := service.SubscribeCampaignDebugUpdates(ctx, SubscribeCampaignDebugUpdatesInput{
		CampaignID: " campaign-1 ",
		SessionID:  " session-1 ",
	})
	if err != nil {
		t.Fatalf("SubscribeCampaignDebugUpdates: %v", err)
	}
	defer unsubscribe()

	expected := CampaignDebugTurnUpdate{
		Turn:            store.turns["turn-1"],
		AppendedEntries: []debugtrace.Entry{{TurnID: "turn-1", Sequence: 2, Kind: debugtrace.EntryKindToolResult}},
	}
	broker.Publish("campaign-1", "session-1", expected)

	select {
	case got := <-updates:
		if got.Turn.ID != "turn-1" || len(got.AppendedEntries) != 1 || got.AppendedEntries[0].Sequence != 2 {
			t.Fatalf("update = %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update")
	}
}

func TestCampaignDebugServiceValidationAndErrorMapping(t *testing.T) {
	t.Parallel()

	if _, err := NewCampaignDebugService(CampaignDebugServiceConfig{}); err == nil {
		t.Fatal("expected missing store error")
	}

	store := newDebugTraceStoreStub()
	service, err := NewCampaignDebugService(CampaignDebugServiceConfig{DebugTraceStore: store})
	if err != nil {
		t.Fatalf("NewCampaignDebugService: %v", err)
	}

	if _, err := service.ListCampaignDebugTurns(context.Background(), ListCampaignDebugTurnsInput{}); ErrorKindOf(err) != ErrKindInvalidArgument {
		t.Fatalf("ListCampaignDebugTurns error kind = %v, want %v", ErrorKindOf(err), ErrKindInvalidArgument)
	}

	store.getErr = storage.ErrNotFound
	if _, err := service.GetCampaignDebugTurn(context.Background(), GetCampaignDebugTurnInput{
		CampaignID: "campaign-1",
		TurnID:     "missing",
	}); ErrorKindOf(err) != ErrKindNotFound {
		t.Fatalf("GetCampaignDebugTurn error kind = %v, want %v", ErrorKindOf(err), ErrKindNotFound)
	}

	if _, _, err := service.SubscribeCampaignDebugUpdates(context.Background(), SubscribeCampaignDebugUpdatesInput{
		CampaignID: "campaign-1",
		SessionID:  "session-1",
	}); ErrorKindOf(err) != ErrKindFailedPrecondition {
		t.Fatalf("SubscribeCampaignDebugUpdates error kind = %v, want %v", ErrorKindOf(err), ErrKindFailedPrecondition)
	}
}

func TestCampaignDebugUpdateBrokerPublishesLatestAndUnsubscribes(t *testing.T) {
	t.Parallel()

	broker := NewCampaignDebugUpdateBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, unsubscribe := broker.Subscribe(ctx, "campaign-1", "session-1")
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	for seq := 1; seq <= 40; seq++ {
		broker.Publish("campaign-1", "session-1", CampaignDebugTurnUpdate{
			Turn: debugtrace.Turn{ID: "turn-1"},
			AppendedEntries: []debugtrace.Entry{{
				TurnID:    "turn-1",
				Sequence:  seq,
				CreatedAt: time.Unix(int64(seq), 0).UTC(),
			}},
		})
	}

	var last CampaignDebugTurnUpdate
	for i := 0; i < campaignDebugUpdateBuffer; i++ {
		last = <-ch
	}
	if got := last.AppendedEntries[0].Sequence; got != 40 {
		t.Fatalf("last sequence = %d, want %d", got, 40)
	}

	unsubscribe()
	if _, ok := <-ch; ok {
		t.Fatal("expected closed subscription channel")
	}
}

func TestCampaignDebugTraceRecorderRecordsAndPublishesEntries(t *testing.T) {
	t.Parallel()

	store := newDebugTraceStoreStub()
	broker := NewCampaignDebugUpdateBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, unsubscribe := broker.Subscribe(ctx, "campaign-1", "session-1")
	defer unsubscribe()

	now := time.Unix(1712222222, 0).UTC()
	clockValues := []time.Time{now, now.Add(time.Second), now.Add(2 * time.Second), now.Add(3 * time.Second)}
	clockIndex := 0
	clock := func() time.Time {
		value := clockValues[clockIndex]
		if clockIndex < len(clockValues)-1 {
			clockIndex++
		}
		return value
	}

	recorder := newCampaignDebugTraceRecorder(
		context.Background(),
		store,
		clock,
		broker,
		func() (string, error) { return "turn-1", nil },
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		debugtrace.Turn{
			CampaignID: "campaign-1",
			SessionID:  "session-1",
			Model:      "gpt-5.4",
		},
	)
	if recorder == nil {
		t.Fatal("recorder = nil")
	}
	if recorder.TurnID() != "turn-1" {
		t.Fatalf("TurnID = %q, want %q", recorder.TurnID(), "turn-1")
	}

	initial := <-updates
	if initial.Turn.Status != debugtrace.StatusRunning || len(initial.AppendedEntries) != 0 {
		t.Fatalf("initial update = %#v", initial)
	}

	recorder.RecordProviderStep(context.Background(), orchestration.ProviderOutput{
		ConversationID: "resp-1",
		OutputText:     "Scene opens.",
		ToolCalls: []orchestration.ProviderToolCall{{
			CallID:    "call-1",
			Name:      "scene_create",
			Arguments: `{"name":"Harbor"}`,
		}},
		Usage: provider.Usage{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
	})

	stepUpdate := <-updates
	if len(stepUpdate.AppendedEntries) != 2 {
		t.Fatalf("provider step update = %#v", stepUpdate)
	}
	if store.entryWrites[0].Kind != debugtrace.EntryKindModelResponse || store.entryWrites[1].Kind != debugtrace.EntryKindToolCall {
		t.Fatalf("entryWrites = %#v", store.entryWrites)
	}

	recorder.RecordToolResult(context.Background(), orchestration.ProviderToolCall{Name: "scene_create"}, orchestration.ProviderToolResult{
		CallID:  "call-1",
		Output:  `{"scene_id":"scene-1"}`,
		IsError: false,
	})
	resultUpdate := <-updates
	if len(resultUpdate.AppendedEntries) != 1 || resultUpdate.AppendedEntries[0].Kind != debugtrace.EntryKindToolResult {
		t.Fatalf("tool result update = %#v", resultUpdate)
	}

	recorder.Finish(context.Background(), errors.New("scene delayed"))
	finishUpdate := <-updates
	if finishUpdate.Turn.Status != debugtrace.StatusFailed || finishUpdate.Turn.LastError != "scene delayed" {
		t.Fatalf("finish update = %#v", finishUpdate)
	}

	finalTurn := store.turns["turn-1"]
	if finalTurn.EntryCount != 3 {
		t.Fatalf("EntryCount = %d, want %d", finalTurn.EntryCount, 3)
	}
	if finalTurn.Usage.TotalTokens != 15 {
		t.Fatalf("Usage = %#v", finalTurn.Usage)
	}
	if finalTurn.CompletedAt == nil {
		t.Fatal("CompletedAt = nil, want terminal timestamp")
	}
}

func TestCampaignDebugTraceRecorderHelpers(t *testing.T) {
	t.Parallel()

	if recorder := newCampaignDebugTraceRecorder(context.Background(), nil, nil, nil, nil, nil, debugtrace.Turn{}); recorder != nil {
		t.Fatalf("recorder = %#v, want nil", recorder)
	}

	long := "é" + strings.Repeat("a", campaignDebugPayloadMaxBytes)
	payload, truncated := truncateCampaignDebugPayload(long)
	if !truncated {
		t.Fatal("truncateCampaignDebugPayload truncated = false, want true")
	}
	if len(payload) > campaignDebugPayloadMaxBytes {
		t.Fatalf("payload len = %d, want <= %d", len(payload), campaignDebugPayloadMaxBytes)
	}
	if !strings.HasSuffix(payload, campaignDebugPayloadSuffix) {
		t.Fatalf("payload suffix missing: %q", payload)
	}
}

func TestCampaignDebugTraceRecorderFailurePaths(t *testing.T) {
	t.Parallel()

	now := time.Unix(1712222222, 0).UTC()
	clock := func() time.Time { return now }

	t.Run("startup id generation failure disables recorder", func(t *testing.T) {
		t.Parallel()

		recorder := newCampaignDebugTraceRecorder(
			context.Background(),
			newDebugTraceStoreStub(),
			clock,
			nil,
			func() (string, error) { return "", errors.New("id failed") },
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			debugtrace.Turn{CampaignID: "campaign-1", SessionID: "session-1"},
		)
		if recorder == nil || !recorder.disabled || recorder.TurnID() != "" {
			t.Fatalf("recorder = %#v", recorder)
		}
	})

	t.Run("startup persistence failure disables recorder", func(t *testing.T) {
		t.Parallel()

		store := newDebugTraceStoreStub()
		store.putTurnErr = func(debugtrace.Turn) error { return errors.New("put turn failed") }
		recorder := newCampaignDebugTraceRecorder(
			context.Background(),
			store,
			clock,
			nil,
			func() (string, error) { return "turn-1", nil },
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			debugtrace.Turn{CampaignID: "campaign-1", SessionID: "session-1"},
		)
		if recorder == nil || !recorder.disabled || recorder.TurnID() != "turn-1" {
			t.Fatalf("recorder = %#v", recorder)
		}
	})

	t.Run("empty provider step updates turn without entries", func(t *testing.T) {
		t.Parallel()

		store := newDebugTraceStoreStub()
		recorder := newCampaignDebugTraceRecorder(
			context.Background(),
			store,
			clock,
			nil,
			func() (string, error) { return "turn-1", nil },
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			debugtrace.Turn{CampaignID: "campaign-1", SessionID: "session-1"},
		)
		recorder.RecordProviderStep(context.Background(), orchestration.ProviderOutput{})

		if len(store.entryWrites) != 0 {
			t.Fatalf("entryWrites = %#v, want no entries", store.entryWrites)
		}
		if len(store.turnWrites) < 2 {
			t.Fatalf("turnWrites = %#v, want startup and empty-step updates", store.turnWrites)
		}
	})

	t.Run("entry persistence failure disables recorder", func(t *testing.T) {
		t.Parallel()

		store := newDebugTraceStoreStub()
		store.putEntryErr = func(debugtrace.Entry) error { return errors.New("put entry failed") }
		recorder := newCampaignDebugTraceRecorder(
			context.Background(),
			store,
			clock,
			nil,
			func() (string, error) { return "turn-1", nil },
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			debugtrace.Turn{CampaignID: "campaign-1", SessionID: "session-1"},
		)

		recorder.RecordToolResult(context.Background(), orchestration.ProviderToolCall{Name: "scene_create"}, orchestration.ProviderToolResult{
			CallID: "call-1",
			Output: "{}",
		})

		if !recorder.disabled {
			t.Fatal("recorder.disabled = false, want true")
		}
		if len(store.entryWrites) != 0 {
			t.Fatalf("entryWrites = %#v, want no persisted entries", store.entryWrites)
		}
	})

	t.Run("turn update failure disables recorder after entry batch", func(t *testing.T) {
		t.Parallel()

		store := newDebugTraceStoreStub()
		store.putTurnErr = func(turn debugtrace.Turn) error {
			if turn.EntryCount > 0 {
				return errors.New("update turn failed")
			}
			return nil
		}
		recorder := newCampaignDebugTraceRecorder(
			context.Background(),
			store,
			clock,
			nil,
			func() (string, error) { return "turn-1", nil },
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			debugtrace.Turn{CampaignID: "campaign-1", SessionID: "session-1"},
		)

		recorder.RecordToolResult(context.Background(), orchestration.ProviderToolCall{Name: "scene_create"}, orchestration.ProviderToolResult{
			CallID: "call-1",
			Output: "{}",
		})

		if !recorder.disabled {
			t.Fatal("recorder.disabled = false, want true")
		}
		if len(store.entryWrites) != 1 {
			t.Fatalf("entryWrites = %#v, want one persisted entry before disable", store.entryWrites)
		}
	})
}
