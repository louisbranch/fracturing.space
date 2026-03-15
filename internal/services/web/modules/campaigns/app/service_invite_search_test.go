package app

import (
	"context"
	"testing"
)

func TestSearchInviteUsersForwardsRawUsernameStyleQuery(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
		inviteSearchResults:   []InviteUserSearchResult{{UserID: "user-2", Username: "alice"}},
	}
	svc := newService(gateway)

	results, err := svc.SearchInviteUsers(context.Background(), "c1", SearchInviteUsersInput{
		ViewerUserID: "user-1",
		Query:        "  @Al!  ",
	})
	if err != nil {
		t.Fatalf("SearchInviteUsers() error = %v", err)
	}
	if len(results) != 1 || results[0].Username != "alice" {
		t.Fatalf("results = %#v", results)
	}
	if gateway.searchInviteCalls != 1 {
		t.Fatalf("search invite calls = %d, want 1", gateway.searchInviteCalls)
	}
	if got := gateway.lastSearchInviteUsersInput.Query; got != "@Al!" {
		t.Fatalf("forwarded query = %q, want %q", got, "@Al!")
	}
	if got := gateway.lastSearchInviteUsersInput.ViewerUserID; got != "user-1" {
		t.Fatalf("viewer user id = %q, want %q", got, "user-1")
	}
	if got := gateway.lastSearchInviteUsersInput.Limit; got != 8 {
		t.Fatalf("limit = %d, want 8 default", got)
	}
}

func TestSearchInviteUsersDoesNotApplyLengthGateBeforeGateway(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{
		authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true},
	}
	svc := newService(gateway)

	results, err := svc.SearchInviteUsers(context.Background(), "c1", SearchInviteUsersInput{
		ViewerUserID: "user-1",
		Query:        "@a",
	})
	if err != nil {
		t.Fatalf("SearchInviteUsers() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
	if gateway.searchInviteCalls != 1 {
		t.Fatalf("search invite calls = %d, want 1", gateway.searchInviteCalls)
	}
	if got := gateway.lastSearchInviteUsersInput.Query; got != "@a" {
		t.Fatalf("forwarded query = %q, want %q", got, "@a")
	}
}
