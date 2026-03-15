package sessionrolltransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
)

func outcomeToProto(outcome daggerheartdomain.Outcome) pb.Outcome {
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
