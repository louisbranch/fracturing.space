package sqlite

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func testEvent(campaignID string, typ event.Type, sessionID string) event.Event {
	return event.Event{
		CampaignID:  campaignID,
		Timestamp:   time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC),
		Type:        typ,
		SessionID:   sessionID,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "campaign",
		EntityID:    campaignID,
		PayloadJSON: []byte(`{}`),
	}
}

func TestAppendAndGetBySeq(t *testing.T) {
	store := openTestEventsStore(t)

	evt := testEvent("camp-evt", event.Type("campaign.created"), "")
	stored, err := store.AppendEvent(context.Background(), evt)
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	if stored.Seq != 1 {
		t.Fatalf("expected seq 1, got %d", stored.Seq)
	}
	if stored.Hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if stored.ChainHash == "" {
		t.Fatal("expected non-empty chain hash")
	}
	if stored.Signature == "" {
		t.Fatal("expected non-empty signature")
	}
	if stored.SignatureKeyID == "" {
		t.Fatal("expected non-empty signature key id")
	}

	got, err := store.GetEventBySeq(context.Background(), "camp-evt", 1)
	if err != nil {
		t.Fatalf("get event by seq: %v", err)
	}
	if got.Hash != stored.Hash {
		t.Fatalf("expected hash to match")
	}
	if got.CampaignID != "camp-evt" {
		t.Fatalf("expected campaign id to match")
	}
}

func TestAppendAndGetByHash(t *testing.T) {
	store := openTestEventsStore(t)

	evt := testEvent("camp-hash", event.Type("campaign.created"), "")
	stored, err := store.AppendEvent(context.Background(), evt)
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	got, err := store.GetEventByHash(context.Background(), stored.Hash)
	if err != nil {
		t.Fatalf("get event by hash: %v", err)
	}
	if got.Seq != stored.Seq || got.CampaignID != stored.CampaignID {
		t.Fatalf("expected event to match by hash lookup")
	}
}

func TestAppendChainIntegrity(t *testing.T) {
	store := openTestEventsStore(t)
	campaignID := "camp-chain"

	var events []event.Event
	for i := 0; i < 3; i++ {
		evt := testEvent(campaignID, event.Type("campaign.created"), "")
		evt.Timestamp = time.Date(2026, 2, 3, 12, i, 0, 0, time.UTC)
		stored, err := store.AppendEvent(context.Background(), evt)
		if err != nil {
			t.Fatalf("append event %d: %v", i+1, err)
		}
		events = append(events, stored)
	}

	if events[0].Seq != 1 || events[1].Seq != 2 || events[2].Seq != 3 {
		t.Fatalf("expected sequential seq numbers")
	}

	// First event has empty PrevHash
	if events[0].PrevHash != "" {
		t.Fatalf("expected first event prev hash to be empty, got %q", events[0].PrevHash)
	}

	// Event N PrevHash = Event N-1 ChainHash
	if events[1].PrevHash != events[0].ChainHash {
		t.Fatalf("expected event 2 prev hash to equal event 1 chain hash")
	}
	if events[2].PrevHash != events[1].ChainHash {
		t.Fatalf("expected event 3 prev hash to equal event 2 chain hash")
	}
}

func TestAppendIdempotent(t *testing.T) {
	store := openTestEventsStore(t)

	evt := testEvent("camp-idem", event.Type("campaign.created"), "")
	first, err := store.AppendEvent(context.Background(), evt)
	if err != nil {
		t.Fatalf("first append: %v", err)
	}

	// Second append of the same event should return the stored event
	second, err := store.AppendEvent(context.Background(), evt)
	if err != nil {
		t.Fatalf("second append: %v", err)
	}
	if second.Hash != first.Hash {
		t.Fatalf("expected idempotent append to return same hash")
	}
}

func TestListEvents(t *testing.T) {
	store := openTestEventsStore(t)
	campaignID := "camp-list-evt"

	for i := 0; i < 5; i++ {
		evt := testEvent(campaignID, event.Type("campaign.created"), "")
		evt.Timestamp = time.Date(2026, 2, 3, 12, i, 0, 0, time.UTC)
		if _, err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("append event %d: %v", i+1, err)
		}
	}

	// afterSeq=0, limit=3 → 3 results
	page1, err := store.ListEvents(context.Background(), campaignID, 0, 3)
	if err != nil {
		t.Fatalf("list events page 1: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("expected 3 events, got %d", len(page1))
	}

	// afterSeq=3 → 2 results
	page2, err := store.ListEvents(context.Background(), campaignID, 3, 10)
	if err != nil {
		t.Fatalf("list events page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("expected 2 events, got %d", len(page2))
	}
}

func TestListEventsBySession(t *testing.T) {
	store := openTestEventsStore(t)
	campaignID := "camp-sess-evt"

	// 3 events in session A, 2 in session B
	for i := 0; i < 3; i++ {
		evt := testEvent(campaignID, event.Type("session.started"), "sess-a")
		evt.Timestamp = time.Date(2026, 2, 3, 12, i, 0, 0, time.UTC)
		if _, err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("append event sess-a %d: %v", i+1, err)
		}
	}
	for i := 0; i < 2; i++ {
		evt := testEvent(campaignID, event.Type("session.started"), "sess-b")
		evt.Timestamp = time.Date(2026, 2, 3, 13, i, 0, 0, time.UTC)
		if _, err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("append event sess-b %d: %v", i+1, err)
		}
	}

	sessA, err := store.ListEventsBySession(context.Background(), campaignID, "sess-a", 0, 100)
	if err != nil {
		t.Fatalf("list events by session A: %v", err)
	}
	if len(sessA) != 3 {
		t.Fatalf("expected 3 events for sess-a, got %d", len(sessA))
	}

	sessB, err := store.ListEventsBySession(context.Background(), campaignID, "sess-b", 0, 100)
	if err != nil {
		t.Fatalf("list events by session B: %v", err)
	}
	if len(sessB) != 2 {
		t.Fatalf("expected 2 events for sess-b, got %d", len(sessB))
	}
}

func TestGetLatestEventSeq(t *testing.T) {
	store := openTestEventsStore(t)
	campaignID := "camp-latest"

	// Empty campaign returns 0
	seq, err := store.GetLatestEventSeq(context.Background(), campaignID)
	if err != nil {
		t.Fatalf("get latest event seq (empty): %v", err)
	}
	if seq != 0 {
		t.Fatalf("expected seq 0 for empty campaign, got %d", seq)
	}

	for i := 0; i < 3; i++ {
		evt := testEvent(campaignID, event.Type("campaign.created"), "")
		evt.Timestamp = time.Date(2026, 2, 3, 12, i, 0, 0, time.UTC)
		if _, err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("append event %d: %v", i+1, err)
		}
	}

	seq, err = store.GetLatestEventSeq(context.Background(), campaignID)
	if err != nil {
		t.Fatalf("get latest event seq: %v", err)
	}
	if seq != 3 {
		t.Fatalf("expected seq 3, got %d", seq)
	}
}

func TestListEventsPage(t *testing.T) {
	store := openTestEventsStore(t)
	campaignID := "camp-page"

	for i := 0; i < 10; i++ {
		evt := testEvent(campaignID, event.Type("campaign.created"), "")
		evt.Timestamp = time.Date(2026, 2, 3, 12, i, 0, 0, time.UTC)
		if _, err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("append event %d: %v", i+1, err)
		}
	}

	// Forward pagination, ascending
	result, err := store.ListEventsPage(context.Background(), storage.ListEventsPageRequest{
		CampaignID: campaignID,
		PageSize:   3,
	})
	if err != nil {
		t.Fatalf("list events page: %v", err)
	}
	if len(result.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(result.Events))
	}
	if result.TotalCount != 10 {
		t.Fatalf("expected total count 10, got %d", result.TotalCount)
	}
	if !result.HasNextPage {
		t.Fatal("expected has next page")
	}
	if result.HasPrevPage {
		t.Fatal("expected no prev page for first page")
	}

	// Descending order
	descResult, err := store.ListEventsPage(context.Background(), storage.ListEventsPageRequest{
		CampaignID: campaignID,
		PageSize:   3,
		Descending: true,
	})
	if err != nil {
		t.Fatalf("list events page descending: %v", err)
	}
	if len(descResult.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(descResult.Events))
	}
	// Descending: first event should have highest seq
	if descResult.Events[0].Seq != 10 {
		t.Fatalf("expected first event seq 10 in desc order, got %d", descResult.Events[0].Seq)
	}

	// Backward pagination (CursorReverse) - simulates "previous page" from cursor 7
	revResult, err := store.ListEventsPage(context.Background(), storage.ListEventsPageRequest{
		CampaignID:    campaignID,
		PageSize:      3,
		CursorSeq:     7,
		CursorDir:     "bwd",
		CursorReverse: true,
	})
	if err != nil {
		t.Fatalf("list events page reverse: %v", err)
	}
	if len(revResult.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(revResult.Events))
	}
	// CursorReverse reverses results back to ascending order
	if revResult.Events[0].Seq >= revResult.Events[2].Seq {
		t.Fatalf("expected ascending order after cursor reverse, got seq %d >= %d",
			revResult.Events[0].Seq, revResult.Events[2].Seq)
	}
	if !revResult.HasNextPage {
		t.Fatal("expected has next page with cursor reverse")
	}

	// FilterClause: only session events
	sessEvt := testEvent(campaignID, event.Type("session.started"), "sess-filter")
	sessEvt.Timestamp = time.Date(2026, 2, 3, 13, 0, 0, 0, time.UTC)
	if _, err := store.AppendEvent(context.Background(), sessEvt); err != nil {
		t.Fatalf("append session event: %v", err)
	}

	filterResult, err := store.ListEventsPage(context.Background(), storage.ListEventsPageRequest{
		CampaignID:   campaignID,
		PageSize:     20,
		FilterClause: "session_id = ?",
		FilterParams: []any{"sess-filter"},
	})
	if err != nil {
		t.Fatalf("list events page with filter: %v", err)
	}
	if len(filterResult.Events) != 1 {
		t.Fatalf("expected 1 filtered event, got %d", len(filterResult.Events))
	}
	if filterResult.TotalCount != 1 {
		t.Fatalf("expected total count 1 with filter, got %d", filterResult.TotalCount)
	}

	// CursorReverse + Descending: reverse of descending sort
	revDescResult, err := store.ListEventsPage(context.Background(), storage.ListEventsPageRequest{
		CampaignID:    campaignID,
		PageSize:      3,
		CursorSeq:     7,
		CursorDir:     "bwd",
		CursorReverse: true,
		Descending:    true,
	})
	if err != nil {
		t.Fatalf("list events page reverse descending: %v", err)
	}
	if len(revDescResult.Events) == 0 {
		t.Fatal("expected events from reverse descending query")
	}
}

func TestListEventsPageAfterSeq(t *testing.T) {
	store := openTestEventsStore(t)
	campaignID := "camp-after-seq"

	for i := 0; i < 5; i++ {
		evt := testEvent(campaignID, event.Type("campaign.created"), "")
		evt.Timestamp = time.Date(2026, 2, 3, 12, i, 0, 0, time.UTC)
		if _, err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("append event %d: %v", i+1, err)
		}
	}

	result, err := store.ListEventsPage(context.Background(), storage.ListEventsPageRequest{
		CampaignID: campaignID,
		AfterSeq:   3,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list events page: %v", err)
	}
	if len(result.Events) != 2 {
		t.Fatalf("events = %d, want %d", len(result.Events), 2)
	}
	if result.Events[0].Seq != 4 || result.Events[1].Seq != 5 {
		t.Fatalf("event seqs = [%d,%d], want [%d,%d]", result.Events[0].Seq, result.Events[1].Seq, 4, 5)
	}
	if result.TotalCount != 2 {
		t.Fatalf("total count = %d, want %d", result.TotalCount, 2)
	}
	if result.HasNextPage {
		t.Fatalf("has next page = true, want false")
	}
}

func TestBuildListEventsPageSQLPlanBuildsExpectedClauses(t *testing.T) {
	plan := buildListEventsPageSQLPlan(storage.ListEventsPageRequest{
		CampaignID:    "camp-plan",
		AfterSeq:      5,
		PageSize:      25,
		CursorSeq:     10,
		CursorDir:     "bwd",
		CursorReverse: true,
		Descending:    true,
		FilterClause:  "session_id = ?",
		FilterParams:  []any{"sess-1"},
	})

	if plan.whereClause != "campaign_id = ? AND seq > ? AND seq < ? AND session_id = ?" {
		t.Fatalf("where clause = %q", plan.whereClause)
	}
	if plan.orderClause != "ORDER BY seq ASC" {
		t.Fatalf("order clause = %q, want %q", plan.orderClause, "ORDER BY seq ASC")
	}
	if plan.limitClause != "LIMIT 26" {
		t.Fatalf("limit clause = %q, want %q", plan.limitClause, "LIMIT 26")
	}
	if len(plan.params) != 4 {
		t.Fatalf("params = %d, want %d", len(plan.params), 4)
	}
	if plan.params[0] != "camp-plan" || plan.params[1] != uint64(5) || plan.params[2] != uint64(10) || plan.params[3] != "sess-1" {
		t.Fatalf("params = %v", plan.params)
	}
	if plan.countWhereClause != "campaign_id = ? AND seq > ? AND session_id = ?" {
		t.Fatalf("count where clause = %q", plan.countWhereClause)
	}
	if len(plan.countParams) != 3 {
		t.Fatalf("count params = %d, want %d", len(plan.countParams), 3)
	}
}

func TestVerifyEventIntegrity(t *testing.T) {
	store := openTestEventsStore(t)
	campaignID := "camp-verify"

	for i := 0; i < 5; i++ {
		evt := testEvent(campaignID, event.Type("campaign.created"), "")
		evt.Timestamp = time.Date(2026, 2, 3, 12, i, 0, 0, time.UTC)
		if _, err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("append event %d: %v", i+1, err)
		}
	}

	if err := store.VerifyEventIntegrity(context.Background()); err != nil {
		t.Fatalf("verify event integrity: %v", err)
	}
}

func TestVerifyEventIntegrityHandlesSubMillisecondTimestamps(t *testing.T) {
	store := openTestEventsStore(t)
	campaignID := "camp-verify-ms"

	evt := testEvent(campaignID, event.Type("campaign.created"), "")
	evt.Timestamp = time.Date(2026, 2, 3, 12, 0, 0, 123456789, time.UTC)
	if _, err := store.AppendEvent(context.Background(), evt); err != nil {
		t.Fatalf("append event: %v", err)
	}

	if err := store.VerifyEventIntegrity(context.Background()); err != nil {
		t.Fatalf("verify event integrity: %v", err)
	}
}

func TestGetEventNotFound(t *testing.T) {
	store := openTestEventsStore(t)

	_, err := store.GetEventByHash(context.Background(), "nonexistent-hash")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for hash, got %v", err)
	}

	// Append one event first to ensure the campaign has a seq tracker
	evt := testEvent("camp-nf", event.Type("campaign.created"), "")
	if _, err := store.AppendEvent(context.Background(), evt); err != nil {
		t.Fatalf("append event: %v", err)
	}

	_, err = store.GetEventBySeq(context.Background(), "camp-nf", 999)
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for seq, got %v", err)
	}
}

func TestAppendEventMultipleCampaigns(t *testing.T) {
	store := openTestEventsStore(t)

	// Each campaign gets independent sequence numbers
	for _, campID := range []string{"camp-a", "camp-b"} {
		for i := 0; i < 3; i++ {
			evt := testEvent(campID, event.Type("campaign.created"), "")
			evt.Timestamp = time.Date(2026, 2, 3, 12, i, 0, 0, time.UTC)
			stored, err := store.AppendEvent(context.Background(), evt)
			if err != nil {
				t.Fatalf("append %s event %d: %v", campID, i+1, err)
			}
			expected := uint64(i + 1)
			if stored.Seq != expected {
				t.Fatalf("expected seq %d for %s, got %d", expected, campID, stored.Seq)
			}
		}
	}

	// Verify integrity across both campaigns
	if err := store.VerifyEventIntegrity(context.Background()); err != nil {
		t.Fatalf("verify integrity: %v", err)
	}

	// Verify independent latest seq
	for _, campID := range []string{"camp-a", "camp-b"} {
		seq, err := store.GetLatestEventSeq(context.Background(), campID)
		if err != nil {
			t.Fatalf("get latest seq %s: %v", campID, err)
		}
		if seq != 3 {
			t.Fatalf("expected latest seq 3 for %s, got %d", campID, seq)
		}
	}
}

func TestAppendEventFieldRoundTrip(t *testing.T) {
	store := openTestEventsStore(t)

	evt := event.Event{
		CampaignID:    "camp-fields",
		Timestamp:     time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.character_state_patched"),
		SessionID:     "sess-1",
		RequestID:     "req-1",
		InvocationID:  "inv-1",
		ActorType:     event.ActorTypeParticipant,
		ActorID:       "part-1",
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","hp_after":5}`),
	}

	stored, err := store.AppendEvent(context.Background(), evt)
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	got, err := store.GetEventBySeq(context.Background(), "camp-fields", 1)
	if err != nil {
		t.Fatalf("get event: %v", err)
	}

	checks := []struct {
		name     string
		expected string
		actual   string
	}{
		{"CampaignID", stored.CampaignID, got.CampaignID},
		{"SessionID", stored.SessionID, got.SessionID},
		{"RequestID", stored.RequestID, got.RequestID},
		{"InvocationID", stored.InvocationID, got.InvocationID},
		{"ActorID", stored.ActorID, got.ActorID},
		{"EntityType", stored.EntityType, got.EntityType},
		{"EntityID", stored.EntityID, got.EntityID},
		{"CorrelationID", stored.CorrelationID, got.CorrelationID},
		{"CausationID", stored.CausationID, got.CausationID},
	}
	for _, c := range checks {
		if c.expected != c.actual {
			t.Fatalf("%s: expected %q, got %q", c.name, c.expected, c.actual)
		}
	}
	if string(got.PayloadJSON) != `{"character_id":"char-1","hp_after":5}` {
		t.Fatalf("expected payload to round-trip, got %s", string(got.PayloadJSON))
	}
	if fmt.Sprintf("%d", got.Seq) != fmt.Sprintf("%d", stored.Seq) {
		t.Fatalf("expected seq to match")
	}
}
