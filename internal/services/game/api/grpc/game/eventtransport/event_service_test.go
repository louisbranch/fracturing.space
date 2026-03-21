package eventtransport

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestListEvents_EmptyResult(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	authzCtx := gametest.ContextWithAdminOverride("events-test")
	svc := NewService(Deps{Event: eventStore})

	resp, err := svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
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
	eventStore := gametest.NewFakeEventStore()
	authzCtx := gametest.ContextWithAdminOverride("events-test")
	now := time.Now().UTC()

	eventStore.Events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: event.Type("e1"), Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: event.Type("e2"), Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: event.Type("e3"), Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: event.Type("e4"), Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: event.Type("e5"), Timestamp: now},
	}

	svc := NewService(Deps{Event: eventStore})
	resp, err := svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
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
	eventStore := gametest.NewFakeEventStore()
	authzCtx := gametest.ContextWithAdminOverride("events-test")
	now := time.Now().UTC()

	// Add 5 events
	eventStore.Events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: event.Type("e1"), Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: event.Type("e2"), Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: event.Type("e3"), Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: event.Type("e4"), Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: event.Type("e5"), Timestamp: now},
	}

	svc := NewService(Deps{Event: eventStore})

	// Page 1: get first 2 events (ASC order)
	resp, err := svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
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
	resp2, err := svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
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
	respBack, err := svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
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
	eventStore := gametest.NewFakeEventStore()
	authzCtx := gametest.ContextWithAdminOverride("events-test")
	now := time.Now().UTC()

	// Add 5 events
	eventStore.Events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: event.Type("e1"), Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: event.Type("e2"), Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: event.Type("e3"), Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: event.Type("e4"), Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: event.Type("e5"), Timestamp: now},
	}

	svc := NewService(Deps{Event: eventStore})

	// Page 1: get first 2 events (DESC order, so highest seqs first)
	resp, err := svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
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
	resp2, err := svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
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
	respBack, err := svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
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
