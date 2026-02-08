package game

import (
	"context"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/cursor"
	"google.golang.org/grpc/codes"
)

func TestListEvents_NilRequest(t *testing.T) {
	svc := NewEventService(Stores{})
	_, err := svc.ListEvents(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_MissingEventStore(t *testing.T) {
	svc := NewEventService(Stores{})
	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
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
	tokenCursor := cursor.NewForwardCursor(10, "type=session.started", "seq")
	token, err := cursor.Encode(tokenCursor)
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
	tokenCursor := cursor.NewForwardCursor(10, "", "seq")
	token, err := cursor.Encode(tokenCursor)
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

func TestListEvents_ASC_Pagination(t *testing.T) {
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	// Add 5 events
	eventStore.events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: "e1", Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: "e2", Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: "e3", Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: "e4", Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: "e5", Timestamp: now},
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

func TestListEvents_DESC_Pagination(t *testing.T) {
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	// Add 5 events
	eventStore.events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: "e1", Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: "e2", Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: "e3", Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: "e4", Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: "e5", Timestamp: now},
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
	invalidCursor := cursor.Cursor{
		Seq:        10,
		Dir:        "invalid",
		FilterHash: "",
	}
	token, err := cursor.Encode(invalidCursor)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	_, err = svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageToken:  token,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
