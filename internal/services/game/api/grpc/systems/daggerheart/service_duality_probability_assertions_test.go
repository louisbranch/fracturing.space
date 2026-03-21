package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
)

// assertProbabilityResponse validates duality probability response fields against expectations.
func assertProbabilityResponse(t *testing.T, response *pb.DualityProbabilityResponse, request daggerheartdomain.ProbabilityRequest) {
	t.Helper()

	if response == nil {
		t.Fatal("DualityProbability response is nil")
	}

	result, err := daggerheartdomain.DualityProbability(request)
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}

	if response.TotalOutcomes != int32(result.TotalOutcomes) {
		t.Fatalf("DualityProbability total_outcomes = %d, want %d", response.TotalOutcomes, result.TotalOutcomes)
	}
	if response.CritCount != int32(result.CritCount) {
		t.Fatalf("DualityProbability crit_count = %d, want %d", response.CritCount, result.CritCount)
	}
	if response.SuccessCount != int32(result.SuccessCount) {
		t.Fatalf("DualityProbability success_count = %d, want %d", response.SuccessCount, result.SuccessCount)
	}
	if response.FailureCount != int32(result.FailureCount) {
		t.Fatalf("DualityProbability failure_count = %d, want %d", response.FailureCount, result.FailureCount)
	}
	if len(response.GetOutcomeCounts()) != len(result.OutcomeCounts) {
		t.Fatalf("DualityProbability outcome count len = %d, want %d", len(response.GetOutcomeCounts()), len(result.OutcomeCounts))
	}

	for i, count := range response.GetOutcomeCounts() {
		want := result.OutcomeCounts[i]
		if count.Outcome != wantOutcomeProto(want.Outcome) {
			t.Fatalf("DualityProbability outcome[%d] = %v, want %v", i, count.Outcome, wantOutcomeProto(want.Outcome))
		}
		if count.Count != int32(want.Count) {
			t.Fatalf("DualityProbability count[%d] = %d, want %d", i, count.Count, want.Count)
		}
	}
}
