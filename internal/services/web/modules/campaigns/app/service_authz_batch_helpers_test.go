package app

import "testing"

func TestBuildAuthorizationChecksByIDTracksDuplicates(t *testing.T) {
	t.Parallel()

	ids := []string{"", "char-1", "char-2", "char-1", "  ", "char-2", "char-3"}
	checks, indexes := buildAuthorizationChecksByID(
		len(ids),
		func(idx int) string { return ids[idx] },
		func(checkID string, _ int) AuthorizationCheck {
			return AuthorizationCheck{CheckID: checkID}
		},
	)

	if len(checks) != 3 {
		t.Fatalf("len(checks) = %d, want 3", len(checks))
	}
	if checks[0].CheckID != "char-1" || checks[1].CheckID != "char-2" || checks[2].CheckID != "char-3" {
		t.Fatalf("checks = %#v", checks)
	}
	if got := indexes["char-1"]; len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("indexes[char-1] = %#v", got)
	}
	if got := indexes["char-2"]; len(got) != 2 || got[0] != 2 || got[1] != 5 {
		t.Fatalf("indexes[char-2] = %#v", got)
	}
	if got := indexes["char-3"]; len(got) != 1 || got[0] != 6 {
		t.Fatalf("indexes[char-3] = %#v", got)
	}
}

func TestResolvedDecisionCheckIDUsesRequestOrderFallback(t *testing.T) {
	t.Parallel()

	checks := []AuthorizationCheck{{CheckID: "char-1"}, {CheckID: "char-2"}}
	if got := resolvedDecisionCheckID(AuthorizationDecision{CheckID: "char-x"}, 0, checks); got != "char-x" {
		t.Fatalf("resolved explicit check id = %q, want %q", got, "char-x")
	}
	if got := resolvedDecisionCheckID(AuthorizationDecision{}, 1, checks); got != "char-2" {
		t.Fatalf("resolved fallback check id = %q, want %q", got, "char-2")
	}
	if got := resolvedDecisionCheckID(AuthorizationDecision{}, 5, checks); got != "" {
		t.Fatalf("resolved out-of-range fallback = %q, want empty", got)
	}
}

func TestApplyAuthorizationDecisionsAppliesToAllMappedIndexes(t *testing.T) {
	t.Parallel()

	checks := []AuthorizationCheck{{CheckID: "char-1"}, {CheckID: "char-2"}}
	indexes := map[string][]int{
		"char-1": {0, 2},
		"char-2": {1},
	}
	decisions := []AuthorizationDecision{
		{Allowed: true, Evaluated: true},
		{CheckID: "char-2", Allowed: false, Evaluated: true},
	}

	applied := map[int]int{}
	applyAuthorizationDecisions(checks, indexes, decisions, func(idx int, _ AuthorizationDecision) {
		applied[idx]++
	})

	if applied[0] != 1 || applied[2] != 1 || applied[1] != 1 {
		t.Fatalf("applied indexes = %#v", applied)
	}
}

func TestAllowedByCheckIDUsesResolvedIDsAndEvaluatedAllowedOnly(t *testing.T) {
	t.Parallel()

	checks := []AuthorizationCheck{{CheckID: "read"}, {CheckID: "write"}, {CheckID: "admin"}}
	decisions := []AuthorizationDecision{
		{Allowed: true, Evaluated: true},
		{CheckID: "write", Allowed: false, Evaluated: true},
		{CheckID: "admin", Allowed: true, Evaluated: false},
	}
	allowed := allowedByCheckID(checks, decisions)

	if !allowed["read"] {
		t.Fatalf("read should be allowed: %#v", allowed)
	}
	if allowed["write"] {
		t.Fatalf("write should be false: %#v", allowed)
	}
	if allowed["admin"] {
		t.Fatalf("admin should be false when not evaluated: %#v", allowed)
	}
}
