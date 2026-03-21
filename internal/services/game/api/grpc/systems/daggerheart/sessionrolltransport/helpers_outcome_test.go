package sessionrolltransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
)

func TestOutcomeToProto(t *testing.T) {
	tests := []struct {
		name    string
		outcome daggerheartdomain.Outcome
		want    pb.Outcome
	}{
		{name: "roll with hope", outcome: daggerheartdomain.OutcomeRollWithHope, want: pb.Outcome_ROLL_WITH_HOPE},
		{name: "roll with fear", outcome: daggerheartdomain.OutcomeRollWithFear, want: pb.Outcome_ROLL_WITH_FEAR},
		{name: "success with hope", outcome: daggerheartdomain.OutcomeSuccessWithHope, want: pb.Outcome_SUCCESS_WITH_HOPE},
		{name: "success with fear", outcome: daggerheartdomain.OutcomeSuccessWithFear, want: pb.Outcome_SUCCESS_WITH_FEAR},
		{name: "failure with hope", outcome: daggerheartdomain.OutcomeFailureWithHope, want: pb.Outcome_FAILURE_WITH_HOPE},
		{name: "failure with fear", outcome: daggerheartdomain.OutcomeFailureWithFear, want: pb.Outcome_FAILURE_WITH_FEAR},
		{name: "critical success", outcome: daggerheartdomain.OutcomeCriticalSuccess, want: pb.Outcome_CRITICAL_SUCCESS},
		{name: "unknown outcome", outcome: daggerheartdomain.Outcome(99), want: pb.Outcome_OUTCOME_UNSPECIFIED},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := outcomeToProto(tc.outcome); got != tc.want {
				t.Fatalf("outcomeToProto(%v) = %v, want %v", tc.outcome, got, tc.want)
			}
		})
	}
}
