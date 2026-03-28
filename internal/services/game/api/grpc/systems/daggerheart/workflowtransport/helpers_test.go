package workflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
)

func TestOutcomeCodeHelpers(t *testing.T) {
	tests := []struct {
		code        string
		wantFlavor  string
		wantSuccess bool
		wantKnown   bool
	}{
		{code: pb.Outcome_SUCCESS_WITH_HOPE.String(), wantFlavor: "HOPE", wantSuccess: true, wantKnown: true},
		{code: pb.Outcome_FAILURE_WITH_FEAR.String(), wantFlavor: "FEAR", wantSuccess: false, wantKnown: true},
		{code: "invalid", wantFlavor: "", wantSuccess: false, wantKnown: false},
	}

	for _, tc := range tests {
		if got := OutcomeFlavorFromCode(tc.code); got != tc.wantFlavor {
			t.Fatalf("OutcomeFlavorFromCode(%q) = %q, want %q", tc.code, got, tc.wantFlavor)
		}
		success, known := OutcomeSuccessFromCode(tc.code)
		if success != tc.wantSuccess || known != tc.wantKnown {
			t.Fatalf("OutcomeSuccessFromCode(%q) = (%v,%v), want (%v,%v)", tc.code, success, known, tc.wantSuccess, tc.wantKnown)
		}
	}

	if got := OutcomeCodeToProto("invalid"); got != pb.Outcome_OUTCOME_UNSPECIFIED {
		t.Fatalf("OutcomeCodeToProto(invalid) = %v, want unspecified", got)
	}
}

func TestWithCampaignSessionMetadata(t *testing.T) {
	ctx := WithCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	if got := grpcmeta.CampaignIDFromContext(ctx); got != "camp-1" {
		t.Fatalf("campaign id = %q, want camp-1", got)
	}
	if got := grpcmeta.SessionIDFromContext(ctx); got != "sess-1" {
		t.Fatalf("session id = %q, want sess-1", got)
	}
}

func TestNormalizeTargets(t *testing.T) {
	got := NormalizeTargets([]string{"  a ", "", "b", "a", "c"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NormalizeTargets() = %v, want %v", got, want)
		}
	}
}
