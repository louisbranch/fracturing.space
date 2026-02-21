package game

import (
	"context"
	"sync"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestListEvents_NilRequest(t *testing.T) {
	svc := NewEventService(Stores{})
	_, err := svc.ListEvents(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestNormalizeListEventsRequestDefaultsAndScope(t *testing.T) {
	req, err := normalizeListEventsRequest(&campaignv1.ListEventsRequest{
		CampaignId: " c1 ",
		Filter:     " type = \"session.started\" ",
		AfterSeq:   7,
	})
	if err != nil {
		t.Fatalf("normalize list events request: %v", err)
	}
	if req.campaignID != "c1" {
		t.Fatalf("campaign id = %q, want %q", req.campaignID, "c1")
	}
	if req.pageSize != defaultListEventsPageSize {
		t.Fatalf("page size = %d, want %d", req.pageSize, defaultListEventsPageSize)
	}
	if req.orderBy != "seq" {
		t.Fatalf("order by = %q, want %q", req.orderBy, "seq")
	}
	if req.paginationScope != "type = \"session.started\"|after_seq=7" {
		t.Fatalf("pagination scope = %q, want %q", req.paginationScope, "type = \"session.started\"|after_seq=7")
	}
}

func TestListEvents_MissingCampaignId(t *testing.T) {
	eventStore := newFakeEventStore()
	svc := NewEventService(Stores{Event: eventStore})
	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_InvalidOrderBy(t *testing.T) {
	eventStore := newFakeEventStore()
	svc := NewEventService(Stores{Event: eventStore})
	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		OrderBy:    "invalid",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_InvalidFilter(t *testing.T) {
	eventStore := newFakeEventStore()
	svc := NewEventService(Stores{Event: eventStore})
	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		Filter:     "invalid filter syntax ===",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_InvalidPageToken(t *testing.T) {
	eventStore := newFakeEventStore()
	svc := NewEventService(Stores{Event: eventStore})
	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageToken:  "not-valid-base64!!!",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_TokenWithChangedFilter(t *testing.T) {
	eventStore := newFakeEventStore()
	svc := NewEventService(Stores{Event: eventStore})

	// Create a token with one filter
	tokenCursor := pagination.NewCursor(
		[]pagination.CursorValue{pagination.UintValue("seq", 10)},
		pagination.DirectionForward,
		false,
		"type=session.started",
		"seq",
	)
	token, err := pagination.Encode(tokenCursor)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	// Try to use it with a different filter
	_, err = svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		Filter:     "type=session.ended",
		PageToken:  token,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_TokenWithChangedOrderBy(t *testing.T) {
	eventStore := newFakeEventStore()
	svc := NewEventService(Stores{Event: eventStore})

	// Create a token with ASC order
	tokenCursor := pagination.NewCursor(
		[]pagination.CursorValue{pagination.UintValue("seq", 10)},
		pagination.DirectionForward,
		false,
		"",
		"seq",
	)
	token, err := pagination.Encode(tokenCursor)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	// Try to use it with DESC order
	_, err = svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		OrderBy:    "seq desc",
		PageToken:  token,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_TokenWithChangedAfterSeq(t *testing.T) {
	eventStore := newFakeEventStore()
	now := time.Now().UTC()
	eventStore.events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: event.Type("e1"), Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: event.Type("e2"), Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: event.Type("e3"), Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: event.Type("e4"), Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: event.Type("e5"), Timestamp: now},
	}
	svc := NewEventService(Stores{Event: eventStore})

	firstResp, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageSize:   2,
		AfterSeq:   2,
	})
	if err != nil {
		t.Fatalf("first list events: %v", err)
	}
	if firstResp.GetNextPageToken() == "" {
		t.Fatalf("expected next page token")
	}

	_, err = svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageSize:   2,
		AfterSeq:   1,
		PageToken:  firstResp.GetNextPageToken(),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_EmptyResult(t *testing.T) {
	eventStore := newFakeEventStore()
	svc := NewEventService(Stores{Event: eventStore})

	resp, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(resp.Events))
	}
	if resp.NextPageToken != "" {
		t.Errorf("expected no next page token, got %q", resp.NextPageToken)
	}
}

func TestListEvents_AfterSeqFiltersResults(t *testing.T) {
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	eventStore.events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: event.Type("e1"), Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: event.Type("e2"), Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: event.Type("e3"), Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: event.Type("e4"), Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: event.Type("e5"), Timestamp: now},
	}

	svc := NewEventService(Stores{Event: eventStore})
	resp, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		AfterSeq:   3,
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("events = %d, want %d", len(resp.Events), 2)
	}
	if resp.Events[0].Seq != 4 || resp.Events[1].Seq != 5 {
		t.Fatalf("event seqs = [%d,%d], want [%d,%d]", resp.Events[0].Seq, resp.Events[1].Seq, 4, 5)
	}
}

func TestListEvents_ASC_Pagination(t *testing.T) {
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	// Add 5 events
	eventStore.events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: event.Type("e1"), Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: event.Type("e2"), Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: event.Type("e3"), Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: event.Type("e4"), Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: event.Type("e5"), Timestamp: now},
	}

	svc := NewEventService(Stores{Event: eventStore})

	// Page 1: get first 2 events (ASC order)
	resp, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageSize:   2,
	})
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("page 1: expected 2 events, got %d", len(resp.Events))
	}
	if resp.Events[0].Seq != 1 || resp.Events[1].Seq != 2 {
		t.Errorf("page 1: expected seqs [1,2], got [%d,%d]", resp.Events[0].Seq, resp.Events[1].Seq)
	}
	if resp.NextPageToken == "" {
		t.Error("page 1: expected next page token")
	}
	if resp.PreviousPageToken != "" {
		t.Error("page 1: expected no previous page token")
	}

	// Page 2: use next token
	resp2, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageSize:   2,
		PageToken:  resp.NextPageToken,
	})
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if len(resp2.Events) != 2 {
		t.Fatalf("page 2: expected 2 events, got %d", len(resp2.Events))
	}
	if resp2.Events[0].Seq != 3 || resp2.Events[1].Seq != 4 {
		t.Errorf("page 2: expected seqs [3,4], got [%d,%d]", resp2.Events[0].Seq, resp2.Events[1].Seq)
	}
	if resp2.NextPageToken == "" {
		t.Error("page 2: expected next page token")
	}
	if resp2.PreviousPageToken == "" {
		t.Error("page 2: expected previous page token")
	}

	// Go back to page 1 using previous token
	respBack, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageSize:   2,
		PageToken:  resp2.PreviousPageToken,
	})
	if err != nil {
		t.Fatalf("back to page 1 error: %v", err)
	}
	if len(respBack.Events) != 2 {
		t.Fatalf("back: expected 2 events, got %d", len(respBack.Events))
	}
	if respBack.Events[0].Seq != 1 || respBack.Events[1].Seq != 2 {
		t.Errorf("back: expected seqs [1,2], got [%d,%d]", respBack.Events[0].Seq, respBack.Events[1].Seq)
	}
}

func TestAppendEvent_UsesDomainEngineForActionEvents(t *testing.T) {
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		eventType   string
		commandType command.Type
	}{
		{
			name:        "note added",
			eventType:   "action.note_added",
			commandType: command.Type("action.note.add"),
		},
		{
			name:        "roll resolved",
			eventType:   "action.roll_resolved",
			commandType: command.Type("action.roll.resolve"),
		},
		{
			name:        "outcome applied",
			eventType:   "action.outcome_applied",
			commandType: command.Type("action.outcome.apply"),
		},
		{
			name:        "outcome rejected",
			eventType:   "action.outcome_rejected",
			commandType: command.Type("action.outcome.reject"),
		},
	}

	results := make(map[command.Type]engine.Result, len(cases))
	for _, tc := range cases {
		results[tc.commandType] = engine.Result{
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type(tc.eventType),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "c1",
				PayloadJSON: []byte("{}"),
			}),
		}
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: results}
	svc := NewEventService(Stores{Event: eventStore, Domain: domain})

	for _, tc := range cases {
		_, err := svc.AppendEvent(context.Background(), &campaignv1.AppendEventRequest{
			CampaignId:  "c1",
			Type:        tc.eventType,
			ActorType:   "system",
			EntityType:  "campaign",
			EntityId:    "c1",
			PayloadJson: []byte("{}"),
		})
		if err != nil {
			t.Fatalf("AppendEvent %s returned error: %v", tc.name, err)
		}
	}

	if domain.calls != len(cases) {
		t.Fatalf("expected domain to be called %d times, got %d", len(cases), domain.calls)
	}
	if len(domain.commands) != len(cases) {
		t.Fatalf("expected %d domain commands, got %d", len(cases), len(domain.commands))
	}
	for i, tc := range cases {
		if domain.commands[i].Type != tc.commandType {
			t.Fatalf("command type = %s, want %s", domain.commands[i].Type, tc.commandType)
		}
	}
}

func TestAppendEvent_RejectsUnmappedTypeWithDomain(t *testing.T) {
	eventStore := newFakeEventStore()
	domain := &fakeDomainEngine{store: eventStore}
	svc := NewEventService(Stores{Event: eventStore, Domain: domain})

	_, err := svc.AppendEvent(context.Background(), &campaignv1.AppendEventRequest{
		CampaignId:  "c1",
		Type:        "campaign.created",
		ActorType:   "system",
		EntityType:  "campaign",
		EntityId:    "c1",
		PayloadJson: []byte("{}"),
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if domain.calls != 0 {
		t.Fatalf("expected domain not to be called, got %d", domain.calls)
	}
}

func TestListEvents_DESC_Pagination(t *testing.T) {
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	// Add 5 events
	eventStore.events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: event.Type("e1"), Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: event.Type("e2"), Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: event.Type("e3"), Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: event.Type("e4"), Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: event.Type("e5"), Timestamp: now},
	}

	svc := NewEventService(Stores{Event: eventStore})

	// Page 1: get first 2 events (DESC order, so highest seqs first)
	resp, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageSize:   2,
		OrderBy:    "seq desc",
	})
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("page 1: expected 2 events, got %d", len(resp.Events))
	}
	if resp.Events[0].Seq != 5 || resp.Events[1].Seq != 4 {
		t.Errorf("page 1 DESC: expected seqs [5,4], got [%d,%d]", resp.Events[0].Seq, resp.Events[1].Seq)
	}
	if resp.NextPageToken == "" {
		t.Error("page 1: expected next page token")
	}
	if resp.PreviousPageToken != "" {
		t.Error("page 1: expected no previous page token")
	}

	// Page 2: use next token (should get seqs 3, 2)
	resp2, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageSize:   2,
		OrderBy:    "seq desc",
		PageToken:  resp.NextPageToken,
	})
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if len(resp2.Events) != 2 {
		t.Fatalf("page 2: expected 2 events, got %d", len(resp2.Events))
	}
	if resp2.Events[0].Seq != 3 || resp2.Events[1].Seq != 2 {
		t.Errorf("page 2 DESC: expected seqs [3,2], got [%d,%d]", resp2.Events[0].Seq, resp2.Events[1].Seq)
	}
	if resp2.NextPageToken == "" {
		t.Error("page 2: expected next page token")
	}
	if resp2.PreviousPageToken == "" {
		t.Error("page 2: expected previous page token")
	}

	// Go back to page 1 using previous token
	respBack, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageSize:   2,
		OrderBy:    "seq desc",
		PageToken:  resp2.PreviousPageToken,
	})
	if err != nil {
		t.Fatalf("back to page 1 error: %v", err)
	}
	if len(respBack.Events) != 2 {
		t.Fatalf("back: expected 2 events, got %d", len(respBack.Events))
	}
	if respBack.Events[0].Seq != 5 || respBack.Events[1].Seq != 4 {
		t.Errorf("back DESC: expected seqs [5,4], got [%d,%d]", respBack.Events[0].Seq, respBack.Events[1].Seq)
	}
}

func TestListEvents_TokenWithInvalidDirection(t *testing.T) {
	eventStore := newFakeEventStore()
	svc := NewEventService(Stores{Event: eventStore})

	// Manually create a token with invalid direction
	invalidCursor := pagination.Cursor{
		Values:     []pagination.CursorValue{pagination.UintValue("seq", 10)},
		Dir:        pagination.Direction("invalid"),
		FilterHash: "",
	}
	token, err := pagination.Encode(invalidCursor)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	_, err = svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageToken:  token,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSubscribeCampaignUpdates_MissingCampaignID(t *testing.T) {
	svc := NewEventService(Stores{Event: newFakeEventStore()})
	stream := &fakeCampaignUpdateStream{ctx: context.Background()}

	err := svc.SubscribeCampaignUpdates(&campaignv1.SubscribeCampaignUpdatesRequest{}, stream)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSubscribeCampaignUpdates_StreamsCommittedAndProjectionUpdates(t *testing.T) {
	eventStore := newFakeEventStore()
	now := time.Now().UTC()
	eventStore.events["camp-1"] = []event.Event{
		{
			CampaignID: "camp-1",
			Seq:        2,
			Type:       event.Type("character.updated"),
			Timestamp:  now,
			EntityType: "character",
			EntityID:   "char-1",
		},
	}

	svc := NewEventService(Stores{Event: eventStore})
	ctx, cancel := context.WithCancel(context.Background())
	stream := &fakeCampaignUpdateStream{ctx: ctx}
	stream.onSend = func() {
		if len(stream.updates) >= 2 {
			cancel()
		}
	}
	defer cancel()

	err := svc.SubscribeCampaignUpdates(&campaignv1.SubscribeCampaignUpdatesRequest{
		CampaignId: "camp-1",
		AfterSeq:   1,
	}, stream)
	if err != nil {
		t.Fatalf("subscribe campaign updates: %v", err)
	}

	if len(stream.updates) != 2 {
		t.Fatalf("updates = %d, want %d", len(stream.updates), 2)
	}

	committed := stream.updates[0]
	if committed.GetCampaignId() != "camp-1" {
		t.Fatalf("committed campaign id = %q, want %q", committed.GetCampaignId(), "camp-1")
	}
	if committed.GetSeq() != 2 {
		t.Fatalf("committed seq = %d, want %d", committed.GetSeq(), 2)
	}
	if committed.GetEventCommitted() == nil {
		t.Fatalf("expected committed update kind")
	}

	applied := stream.updates[1]
	if applied.GetProjectionApplied() == nil {
		t.Fatalf("expected projection_applied update kind")
	}
	if applied.GetProjectionApplied().GetSourceSeq() != 2 {
		t.Fatalf("projection source seq = %d, want %d", applied.GetProjectionApplied().GetSourceSeq(), 2)
	}
	if len(applied.GetProjectionApplied().GetScopes()) == 0 {
		t.Fatalf("projection scopes = empty, want non-empty")
	}
}

type fakeCampaignUpdateStream struct {
	ctx     context.Context
	mu      sync.Mutex
	updates []*campaignv1.CampaignUpdate
	onSend  func()
}

func (f *fakeCampaignUpdateStream) Send(update *campaignv1.CampaignUpdate) error {
	f.mu.Lock()
	f.updates = append(f.updates, update)
	hook := f.onSend
	f.mu.Unlock()
	if hook != nil {
		hook()
	}
	return nil
}

func (f *fakeCampaignUpdateStream) SetHeader(metadata.MD) error { return nil }

func (f *fakeCampaignUpdateStream) SendHeader(metadata.MD) error { return nil }

func (f *fakeCampaignUpdateStream) SetTrailer(metadata.MD) {}

func (f *fakeCampaignUpdateStream) Context() context.Context {
	if f.ctx == nil {
		return context.Background()
	}
	return f.ctx
}

func (f *fakeCampaignUpdateStream) SendMsg(any) error { return nil }

func (f *fakeCampaignUpdateStream) RecvMsg(any) error { return nil }
