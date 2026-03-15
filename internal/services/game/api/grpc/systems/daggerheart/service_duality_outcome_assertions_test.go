package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
)

// assertOutcomeResponse validates duality outcome response fields against expectations.
func assertOutcomeResponse(t *testing.T, response *pb.DualityOutcomeResponse, request daggerheartdomain.OutcomeRequest) {
	t.Helper()

	if response == nil {
		t.Fatal("DualityOutcome response is nil")
	}

	result, err := daggerheartdomain.EvaluateOutcome(request)
	if err != nil {
		t.Fatalf("EvaluateOutcome returned error: %v", err)
	}

	if response.GetHope() != int32(result.Hope) || response.GetFear() != int32(result.Fear) {
		t.Fatalf("DualityOutcome dice = (%d, %d), want (%d, %d)", response.GetHope(), response.GetFear(), result.Hope, result.Fear)
	}
	if response.GetModifier() != int32(result.Modifier) {
		t.Fatalf("DualityOutcome modifier = %d, want %d", response.GetModifier(), result.Modifier)
	}
	if response.Total != int32(result.Total) {
		t.Fatalf("DualityOutcome total = %d, want %d", response.Total, result.Total)
	}
	if response.IsCrit != result.IsCrit {
		t.Fatalf("DualityOutcome is_crit = %t, want %t", response.IsCrit, result.IsCrit)
	}
	if response.MeetsDifficulty != result.MeetsDifficulty {
		t.Fatalf("DualityOutcome meets_difficulty = %t, want %t", response.MeetsDifficulty, result.MeetsDifficulty)
	}
	if response.Outcome != wantOutcomeProto(result.Outcome) {
		t.Fatalf("DualityOutcome outcome = %v, want %v", response.Outcome, wantOutcomeProto(result.Outcome))
	}
	if request.Difficulty != nil && response.Difficulty == nil {
		t.Fatal("DualityOutcome difficulty is nil, want value")
	}
	if request.Difficulty != nil && response.Difficulty != nil && *response.Difficulty != int32(*request.Difficulty) {
		t.Fatalf("DualityOutcome difficulty = %d, want %d", *response.Difficulty, *request.Difficulty)
	}
}
