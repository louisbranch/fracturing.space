package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestPutCampaignTurnRequiresSessionID(t *testing.T) {
	store := openTempStore(t)
	err := store.PutCampaignTurn(context.Background(), storage.CampaignTurnRecord{
		ID:         "turn-1",
		CampaignID: "camp-1",
		AgentID:    "agent-1",
		InputText:  "hello",
		Status:     "accepted",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err == nil || !strings.Contains(err.Error(), "session id is required") {
		t.Fatalf("error = %v, want session id validation", err)
	}
}

func TestPutCampaignTurnRequiresInputText(t *testing.T) {
	store := openTempStore(t)
	err := store.PutCampaignTurn(context.Background(), storage.CampaignTurnRecord{
		ID:         "turn-1",
		CampaignID: "camp-1",
		SessionID:  "session-1",
		AgentID:    "agent-1",
		Status:     "accepted",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err == nil || !strings.Contains(err.Error(), "input text is required") {
		t.Fatalf("error = %v, want input text validation", err)
	}
}

func TestPutCampaignTurnAndUpdateStatus(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)

	err := store.PutCampaignTurn(context.Background(), storage.CampaignTurnRecord{
		ID:                 "turn-1",
		CampaignID:         "camp-1",
		SessionID:          "session-1",
		AgentID:            "agent-1",
		RequesterUserID:    "user-1",
		ParticipantID:      "participant-1",
		ParticipantName:    "Alex",
		CorrelationMessage: "msg-1",
		InputText:          "hello",
		Status:             "accepted",
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		t.Fatalf("put campaign turn: %v", err)
	}

	err = store.UpdateCampaignTurnStatus(context.Background(), "turn-missing", "completed", now.Add(time.Minute))
	if err != storage.ErrNotFound {
		t.Fatalf("update missing campaign turn error = %v, want %v", err, storage.ErrNotFound)
	}

	err = store.UpdateCampaignTurnStatus(context.Background(), "turn-1", "completed", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("update campaign turn status: %v", err)
	}
}

func TestCampaignTurnEventsAppendListAndLatestSequence(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)

	evt1, err := store.AppendCampaignTurnEvent(context.Background(), storage.CampaignTurnEventRecord{
		CampaignID:         "camp-1",
		SessionID:          "session-1",
		TurnID:             "turn-1",
		Kind:               "ai.response",
		Content:            "First",
		ParticipantVisible: true,
		CorrelationMessage: "msg-1",
		CreatedAt:          now,
	})
	if err != nil {
		t.Fatalf("append campaign turn event 1: %v", err)
	}
	if evt1.SequenceID == 0 {
		t.Fatal("expected sequence id to be assigned")
	}

	evt2, err := store.AppendCampaignTurnEvent(context.Background(), storage.CampaignTurnEventRecord{
		CampaignID:         "camp-1",
		SessionID:          "session-1",
		TurnID:             "turn-1",
		Kind:               "ai.response",
		Content:            "Second",
		ParticipantVisible: false,
		CorrelationMessage: "msg-2",
		CreatedAt:          now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("append campaign turn event 2: %v", err)
	}

	_, err = store.AppendCampaignTurnEvent(context.Background(), storage.CampaignTurnEventRecord{
		CampaignID: "camp-2",
		SessionID:  "session-2",
		TurnID:     "turn-2",
		Kind:       "ai.response",
		Content:    "Other campaign",
		CreatedAt:  now,
	})
	if err != nil {
		t.Fatalf("append campaign turn event for second campaign: %v", err)
	}

	records, err := store.ListCampaignTurnEvents(context.Background(), "camp-1", 0, 10)
	if err != nil {
		t.Fatalf("list campaign turn events: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("event count = %d, want 2", len(records))
	}
	if records[0].SequenceID != evt1.SequenceID || records[1].SequenceID != evt2.SequenceID {
		t.Fatalf("sequence ids = [%d %d], want [%d %d]", records[0].SequenceID, records[1].SequenceID, evt1.SequenceID, evt2.SequenceID)
	}
	if !records[0].ParticipantVisible {
		t.Fatal("expected first event participant visibility true")
	}
	if records[1].ParticipantVisible {
		t.Fatal("expected second event participant visibility false")
	}

	afterFirst, err := store.ListCampaignTurnEvents(context.Background(), "camp-1", evt1.SequenceID, 10)
	if err != nil {
		t.Fatalf("list campaign turn events after first sequence: %v", err)
	}
	if len(afterFirst) != 1 || afterFirst[0].SequenceID != evt2.SequenceID {
		t.Fatalf("events after first = %+v, want sequence %d", afterFirst, evt2.SequenceID)
	}

	latest, err := store.GetLatestCampaignTurnEventSequence(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get latest campaign turn event sequence: %v", err)
	}
	if latest != evt2.SequenceID {
		t.Fatalf("latest sequence = %d, want %d", latest, evt2.SequenceID)
	}

	emptyLatest, err := store.GetLatestCampaignTurnEventSequence(context.Background(), "camp-missing")
	if err != nil {
		t.Fatalf("get latest campaign turn event sequence for missing campaign: %v", err)
	}
	if emptyLatest != 0 {
		t.Fatalf("latest sequence for missing campaign = %d, want 0", emptyLatest)
	}
}

func TestCampaignTurnStorageValidationAndContextErrors(t *testing.T) {
	store := openTempStore(t)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.PutCampaignTurn(canceledCtx, storage.CampaignTurnRecord{})
	if err == nil {
		t.Fatal("expected canceled context error from PutCampaignTurn")
	}

	err = store.UpdateCampaignTurnStatus(canceledCtx, "turn-1", "completed", time.Now().UTC())
	if err == nil {
		t.Fatal("expected canceled context error from UpdateCampaignTurnStatus")
	}

	_, err = store.AppendCampaignTurnEvent(canceledCtx, storage.CampaignTurnEventRecord{})
	if err == nil {
		t.Fatal("expected canceled context error from AppendCampaignTurnEvent")
	}

	_, err = store.ListCampaignTurnEvents(canceledCtx, "camp-1", 0, 10)
	if err == nil {
		t.Fatal("expected canceled context error from ListCampaignTurnEvents")
	}

	_, err = store.GetLatestCampaignTurnEventSequence(canceledCtx, "camp-1")
	if err == nil {
		t.Fatal("expected canceled context error from GetLatestCampaignTurnEventSequence")
	}

	err = store.UpdateCampaignTurnStatus(context.Background(), "", "completed", time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "turn id is required") {
		t.Fatalf("error = %v, want turn id validation", err)
	}

	err = store.UpdateCampaignTurnStatus(context.Background(), "turn-1", "", time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "status is required") {
		t.Fatalf("error = %v, want status validation", err)
	}

	_, err = store.AppendCampaignTurnEvent(context.Background(), storage.CampaignTurnEventRecord{CampaignID: "", TurnID: "turn-1", Kind: "ai.response"})
	if err == nil || !strings.Contains(err.Error(), "campaign id is required") {
		t.Fatalf("error = %v, want campaign id validation", err)
	}

	_, err = store.AppendCampaignTurnEvent(context.Background(), storage.CampaignTurnEventRecord{CampaignID: "camp-1", TurnID: "", Kind: "ai.response"})
	if err == nil || !strings.Contains(err.Error(), "turn id is required") {
		t.Fatalf("error = %v, want turn id validation", err)
	}

	_, err = store.AppendCampaignTurnEvent(context.Background(), storage.CampaignTurnEventRecord{CampaignID: "camp-1", TurnID: "turn-1", Kind: ""})
	if err == nil || !strings.Contains(err.Error(), "event kind is required") {
		t.Fatalf("error = %v, want event kind validation", err)
	}

	_, err = store.ListCampaignTurnEvents(context.Background(), "", 0, 10)
	if err == nil || !strings.Contains(err.Error(), "campaign id is required") {
		t.Fatalf("error = %v, want campaign id validation", err)
	}

	_, err = store.ListCampaignTurnEvents(context.Background(), "camp-1", 0, 0)
	if err == nil || !strings.Contains(err.Error(), "limit must be greater than zero") {
		t.Fatalf("error = %v, want limit validation", err)
	}

	_, err = store.GetLatestCampaignTurnEventSequence(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "campaign id is required") {
		t.Fatalf("error = %v, want campaign id validation", err)
	}

	var nilStore *Store
	err = nilStore.PutCampaignTurn(context.Background(), storage.CampaignTurnRecord{})
	if err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("error = %v, want storage not configured", err)
	}

	err = nilStore.UpdateCampaignTurnStatus(context.Background(), "turn-1", "completed", time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("error = %v, want storage not configured", err)
	}

	_, err = nilStore.AppendCampaignTurnEvent(context.Background(), storage.CampaignTurnEventRecord{})
	if err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("error = %v, want storage not configured", err)
	}

	_, err = nilStore.ListCampaignTurnEvents(context.Background(), "camp-1", 0, 10)
	if err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("error = %v, want storage not configured", err)
	}

	_, err = nilStore.GetLatestCampaignTurnEventSequence(context.Background(), "camp-1")
	if err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("error = %v, want storage not configured", err)
	}
}
