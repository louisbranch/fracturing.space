package game

import (
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNormalizeListEventsRequestTokenCursorFields(t *testing.T) {
	cursor := pagination.NewPrevPageCursor(
		[]pagination.CursorValue{pagination.UintValue("seq", 42)},
		true,
		"type = \"session.started\"|after_seq=7",
		"seq desc",
	)
	token, err := pagination.Encode(cursor)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	normalized, err := normalizeListEventsRequest(&campaignv1.ListEventsRequest{
		CampaignId: "camp-1",
		OrderBy:    "seq desc",
		Filter:     "type = \"session.started\"",
		AfterSeq:   7,
		PageToken:  token,
	})
	if err != nil {
		t.Fatalf("normalize list events request: %v", err)
	}

	if normalized.cursorSeq != 42 {
		t.Fatalf("cursor seq = %d, want %d", normalized.cursorSeq, 42)
	}
	if normalized.cursorDir != string(cursor.Dir) {
		t.Fatalf("cursor dir = %q, want %q", normalized.cursorDir, cursor.Dir)
	}
	if !normalized.cursorReverse {
		t.Fatalf("cursor reverse = false, want true")
	}
	if !normalized.descending {
		t.Fatalf("descending = false, want true")
	}
}

func TestNormalizeListEventsRequestRejectsTokenWithoutSeqValue(t *testing.T) {
	cursor := pagination.NewCursor(
		[]pagination.CursorValue{pagination.StringValue("type", "session.started")},
		pagination.DirectionForward,
		false,
		"",
		"seq",
	)
	token, err := pagination.Encode(cursor)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	_, err = normalizeListEventsRequest(&campaignv1.ListEventsRequest{
		CampaignId: "camp-1",
		PageToken:  token,
	})
	if err == nil {
		t.Fatal("expected invalid page token error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}
