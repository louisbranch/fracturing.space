package eventtransport

import "testing"

func TestProjectionScopesForEventType(t *testing.T) {
	scopes := projectionScopesForEventType("character.updated")
	if len(scopes) != 2 {
		t.Fatalf("character scopes len = %d, want %d", len(scopes), 2)
	}
	if scopes[0] != "campaign_characters" || scopes[1] != "campaign_summary" {
		t.Fatalf("character scopes = %#v, want [campaign_characters campaign_summary]", scopes)
	}

	unknownScopes := projectionScopesForEventType("custom.event")
	if len(unknownScopes) != len(defaultCampaignProjectionScopes) {
		t.Fatalf("unknown scopes len = %d, want %d", len(unknownScopes), len(defaultCampaignProjectionScopes))
	}
	unknownScopes[0] = "mutated"
	if defaultCampaignProjectionScopes[0] == "mutated" {
		t.Fatal("default projection scopes were mutated via returned slice")
	}
}

func TestHasProjectionScopeIntersection(t *testing.T) {
	if hasProjectionScopeIntersection(nil, nil) {
		t.Fatal("expected false when scopes are empty")
	}
	if !hasProjectionScopeIntersection([]string{"campaign_summary"}, nil) {
		t.Fatal("expected true when filter is empty")
	}
	if hasProjectionScopeIntersection([]string{"campaign_summary"}, map[string]struct{}{"campaign_sessions": {}}) {
		t.Fatal("expected false when no filter scope matches")
	}
	if !hasProjectionScopeIntersection([]string{"campaign_sessions"}, map[string]struct{}{"campaign_sessions": {}}) {
		t.Fatal("expected true when at least one scope matches")
	}
}
