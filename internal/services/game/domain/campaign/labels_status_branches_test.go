package campaign

import "testing"

func TestNormalizeIntent_CoversAllBranches(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Intent
	}{
		{name: "blank defaults", input: "   ", want: IntentStandard},
		{name: "standard", input: "STANDARD", want: IntentStandard},
		{name: "starter", input: "CAMPAIGN_INTENT_STARTER", want: IntentStarter},
		{name: "sandbox", input: "sandbox", want: IntentSandbox},
		{name: "unknown defaults", input: "unknown", want: IntentStandard},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeIntent(tc.input); got != tc.want {
				t.Fatalf("NormalizeIntent(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeAccessPolicy_CoversAllBranches(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  AccessPolicy
	}{
		{name: "blank defaults", input: "", want: AccessPolicyPrivate},
		{name: "private", input: "PRIVATE", want: AccessPolicyPrivate},
		{name: "restricted", input: "CAMPAIGN_ACCESS_POLICY_RESTRICTED", want: AccessPolicyRestricted},
		{name: "public", input: "public", want: AccessPolicyPublic},
		{name: "unknown defaults", input: "not-real", want: AccessPolicyPrivate},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeAccessPolicy(tc.input); got != tc.want {
				t.Fatalf("NormalizeAccessPolicy(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeStatusLabel_CoversAllBranches(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Status
		ok    bool
	}{
		{name: "blank", input: "", want: StatusUnspecified, ok: false},
		{name: "draft", input: "CAMPAIGN_STATUS_DRAFT", want: StatusDraft, ok: true},
		{name: "active", input: "ACTIVE", want: StatusActive, ok: true},
		{name: "completed", input: "completed", want: StatusCompleted, ok: true},
		{name: "archived", input: "ARCHIVED", want: StatusArchived, ok: true},
		{name: "invalid", input: "never", want: StatusUnspecified, ok: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := normalizeStatusLabel(tc.input)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("normalizeStatusLabel(%q) = (%q,%v), want (%q,%v)", tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestIsStatusTransitionAllowed_CoversAllFromStates(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		{name: "draft->active", from: StatusDraft, to: StatusActive, want: true},
		{name: "draft->archived denied", from: StatusDraft, to: StatusArchived, want: false},
		{name: "active->completed", from: StatusActive, to: StatusCompleted, want: true},
		{name: "active->archived", from: StatusActive, to: StatusArchived, want: true},
		{name: "active->draft denied", from: StatusActive, to: StatusDraft, want: false},
		{name: "completed->archived", from: StatusCompleted, to: StatusArchived, want: true},
		{name: "completed->draft denied", from: StatusCompleted, to: StatusDraft, want: false},
		{name: "archived->draft", from: StatusArchived, to: StatusDraft, want: true},
		{name: "archived->active denied", from: StatusArchived, to: StatusActive, want: false},
		{name: "unspecified denied", from: StatusUnspecified, to: StatusDraft, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isStatusTransitionAllowed(tc.from, tc.to); got != tc.want {
				t.Fatalf("isStatusTransitionAllowed(%q,%q) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestStatusCommandTarget_UnknownCommand(t *testing.T) {
	if _, ok := statusCommandTarget("campaign.unknown"); ok {
		t.Fatal("expected unknown command to have no lifecycle target")
	}
}
