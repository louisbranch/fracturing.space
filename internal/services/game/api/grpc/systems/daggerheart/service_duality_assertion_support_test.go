package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
)

func wantOutcomeProto(outcome daggerheartdomain.Outcome) pb.Outcome {
	switch outcome {
	case daggerheartdomain.OutcomeRollWithHope:
		return pb.Outcome_ROLL_WITH_HOPE
	case daggerheartdomain.OutcomeRollWithFear:
		return pb.Outcome_ROLL_WITH_FEAR
	case daggerheartdomain.OutcomeSuccessWithHope:
		return pb.Outcome_SUCCESS_WITH_HOPE
	case daggerheartdomain.OutcomeSuccessWithFear:
		return pb.Outcome_SUCCESS_WITH_FEAR
	case daggerheartdomain.OutcomeFailureWithHope:
		return pb.Outcome_FAILURE_WITH_HOPE
	case daggerheartdomain.OutcomeFailureWithFear:
		return pb.Outcome_FAILURE_WITH_FEAR
	case daggerheartdomain.OutcomeCriticalSuccess:
		return pb.Outcome_CRITICAL_SUCCESS
	default:
		return pb.Outcome_OUTCOME_UNSPECIFIED
	}
}

// structInt extracts a numeric value from a map payload for tests.
func structInt(t *testing.T, data map[string]any, key string) int {
	t.Helper()
	value, ok := data[key]
	if !ok {
		t.Fatalf("step data missing %q", key)
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		t.Fatalf("step data %q has type %T", key, value)
	}
	return 0
}
