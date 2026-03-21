package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
)

// assertExplainResponse validates duality explain response fields against expectations.
func assertExplainResponse(t *testing.T, response *pb.DualityExplainResponse, request daggerheartdomain.OutcomeRequest) {
	t.Helper()

	if response == nil {
		t.Fatal("DualityExplain response is nil")
	}

	result, err := daggerheartdomain.ExplainOutcome(request)
	if err != nil {
		t.Fatalf("ExplainOutcome returned error: %v", err)
	}

	if response.GetHope() != int32(result.Hope) || response.GetFear() != int32(result.Fear) {
		t.Fatalf("DualityExplain dice = (%d, %d), want (%d, %d)", response.GetHope(), response.GetFear(), result.Hope, result.Fear)
	}
	if response.GetModifier() != int32(result.Modifier) {
		t.Fatalf("DualityExplain modifier = %d, want %d", response.GetModifier(), result.Modifier)
	}
	if response.Total != int32(result.Total) {
		t.Fatalf("DualityExplain total = %d, want %d", response.Total, result.Total)
	}
	if response.IsCrit != result.IsCrit {
		t.Fatalf("DualityExplain is_crit = %t, want %t", response.IsCrit, result.IsCrit)
	}
	if response.MeetsDifficulty != result.MeetsDifficulty {
		t.Fatalf("DualityExplain meets_difficulty = %t, want %t", response.MeetsDifficulty, result.MeetsDifficulty)
	}
	if response.Outcome != wantOutcomeProto(result.Outcome) {
		t.Fatalf("DualityExplain outcome = %v, want %v", response.Outcome, wantOutcomeProto(result.Outcome))
	}
	if response.RulesVersion != result.RulesVersion {
		t.Fatalf("DualityExplain rules_version = %q, want %q", response.RulesVersion, result.RulesVersion)
	}
	if request.Difficulty != nil && response.Difficulty == nil {
		t.Fatal("DualityExplain difficulty is nil, want value")
	}
	if request.Difficulty != nil && response.Difficulty != nil && *response.Difficulty != int32(*request.Difficulty) {
		t.Fatalf("DualityExplain difficulty = %d, want %d", *response.Difficulty, *request.Difficulty)
	}
	if response.GetIntermediates() == nil {
		t.Fatal("DualityExplain intermediates are nil")
	}
	if response.GetIntermediates().GetBaseTotal() != int32(result.Intermediates.BaseTotal) {
		t.Fatalf("DualityExplain base_total = %d, want %d", response.GetIntermediates().GetBaseTotal(), result.Intermediates.BaseTotal)
	}
	if response.GetIntermediates().GetTotal() != int32(result.Intermediates.Total) {
		t.Fatalf("DualityExplain total = %d, want %d", response.GetIntermediates().GetTotal(), result.Intermediates.Total)
	}
	if response.GetIntermediates().GetIsCrit() != result.Intermediates.IsCrit {
		t.Fatalf("DualityExplain is_crit = %t, want %t", response.GetIntermediates().GetIsCrit(), result.Intermediates.IsCrit)
	}
	if response.GetIntermediates().GetMeetsDifficulty() != result.Intermediates.MeetsDifficulty {
		t.Fatalf("DualityExplain meets_difficulty = %t, want %t", response.GetIntermediates().GetMeetsDifficulty(), result.Intermediates.MeetsDifficulty)
	}
	if response.GetIntermediates().GetHopeGtFear() != result.Intermediates.HopeGtFear {
		t.Fatalf("DualityExplain hope_gt_fear = %t, want %t", response.GetIntermediates().GetHopeGtFear(), result.Intermediates.HopeGtFear)
	}
	if response.GetIntermediates().GetFearGtHope() != result.Intermediates.FearGtHope {
		t.Fatalf("DualityExplain fear_gt_hope = %t, want %t", response.GetIntermediates().GetFearGtHope(), result.Intermediates.FearGtHope)
	}
	if len(response.GetSteps()) != len(result.Steps) {
		t.Fatalf("DualityExplain steps = %d, want %d", len(response.GetSteps()), len(result.Steps))
	}
	for i, step := range response.GetSteps() {
		if step.GetCode() != result.Steps[i].Code {
			t.Fatalf("DualityExplain step[%d] code = %q, want %q", i, step.GetCode(), result.Steps[i].Code)
		}
	}
	if len(response.GetSteps()) > 0 {
		baseTotal := structInt(t, response.GetSteps()[0].GetData().AsMap(), "base_total")
		if baseTotal != result.Intermediates.BaseTotal {
			t.Fatalf("DualityExplain step base_total = %d, want %d", baseTotal, result.Intermediates.BaseTotal)
		}
	}
}
