package eventtransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

func TestListEvents_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
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
	eventStore := gametest.NewFakeEventStore()
	svc := NewService(Deps{Event: eventStore})
	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_RequiresCampaignReadPolicy(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	participantStore := gametest.NewFakeParticipantStore()
	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: participantStore},
		Event:       eventStore,
		Participant: participantStore,
	})

	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListEvents_InvalidOrderBy(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	svc := NewService(Deps{Event: eventStore})
	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		OrderBy:    "invalid",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_InvalidFilter(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	svc := NewService(Deps{Event: eventStore})
	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		Filter:     "invalid filter syntax ===",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_InvalidPageToken(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	svc := NewService(Deps{Event: eventStore})
	_, err := svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageToken:  "not-valid-base64!!!",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_TokenWithChangedFilter(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	svc := NewService(Deps{Event: eventStore})

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

	_, err = svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		Filter:     "type=session.ended",
		PageToken:  token,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_TokenWithChangedOrderBy(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	svc := NewService(Deps{Event: eventStore})

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

	_, err = svc.ListEvents(context.Background(), &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		OrderBy:    "seq desc",
		PageToken:  token,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_TokenWithChangedAfterSeq(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	authzCtx := requestctx.WithAdminOverride("events-test")
	now := time.Now().UTC()
	eventStore.Events["c1"] = []event.Event{
		{CampaignID: "c1", Seq: 1, Type: event.Type("e1"), Timestamp: now},
		{CampaignID: "c1", Seq: 2, Type: event.Type("e2"), Timestamp: now},
		{CampaignID: "c1", Seq: 3, Type: event.Type("e3"), Timestamp: now},
		{CampaignID: "c1", Seq: 4, Type: event.Type("e4"), Timestamp: now},
		{CampaignID: "c1", Seq: 5, Type: event.Type("e5"), Timestamp: now},
	}
	svc := NewService(Deps{Event: eventStore})

	firstResp, err := svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
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

	_, err = svc.ListEvents(authzCtx, &campaignv1.ListEventsRequest{
		CampaignId: "c1",
		PageSize:   2,
		AfterSeq:   1,
		PageToken:  firstResp.GetNextPageToken(),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListEvents_TokenWithInvalidDirection(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	svc := NewService(Deps{Event: eventStore})

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
