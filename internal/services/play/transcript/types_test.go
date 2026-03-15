package transcript

import (
	"errors"
	"math"
	"testing"
)

func TestScopeValidate(t *testing.T) {
	t.Parallel()

	if err := (Scope{CampaignID: " c1 ", SessionID: " s1 "}).Validate(); err != nil {
		t.Fatalf("Validate(valid scope) error = %v", err)
	}
	if !errors.Is((Scope{CampaignID: "c1"}).Validate(), ErrInvalidScope) {
		t.Fatal("Validate(missing session) did not return ErrInvalidScope")
	}
}

func TestAppendRequestNormalizeAndValidate(t *testing.T) {
	t.Parallel()

	req := (AppendRequest{
		Scope:           Scope{CampaignID: " c1 ", SessionID: " s1 "},
		Actor:           MessageActor{ParticipantID: " p1 ", Name: " Avery "},
		Body:            " hello ",
		ClientMessageID: " cli-1 ",
	}).Normalize()
	if req.Scope.CampaignID != "c1" || req.Scope.SessionID != "s1" {
		t.Fatalf("normalized scope = %#v", req.Scope)
	}
	if req.Actor.ParticipantID != "p1" || req.Actor.Name != "Avery" {
		t.Fatalf("normalized actor = %#v", req.Actor)
	}
	if req.Body != "hello" || req.ClientMessageID != "cli-1" {
		t.Fatalf("normalized request = %#v", req)
	}
	if !errors.Is((AppendRequest{Scope: Scope{CampaignID: "c1", SessionID: "s1"}}).Validate(), ErrEmptyBody) {
		t.Fatal("Validate(empty body) did not return ErrEmptyBody")
	}
}

func TestHistoryBeforeQueryNormalize(t *testing.T) {
	t.Parallel()

	query := (HistoryBeforeQuery{
		Scope: Scope{CampaignID: " c1 ", SessionID: " s1 "},
		Limit: MaxHistoryLimit + 10,
	}).Normalize()
	if query.Scope.CampaignID != "c1" || query.Scope.SessionID != "s1" {
		t.Fatalf("normalized scope = %#v", query.Scope)
	}
	if query.BeforeSequenceID != math.MaxInt64 {
		t.Fatalf("BeforeSequenceID = %d, want %d", query.BeforeSequenceID, int64(math.MaxInt64))
	}
	if query.Limit != MaxHistoryLimit {
		t.Fatalf("Limit = %d, want %d", query.Limit, MaxHistoryLimit)
	}
}
