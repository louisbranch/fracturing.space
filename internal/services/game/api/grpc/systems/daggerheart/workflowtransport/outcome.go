package workflowtransport

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

const (
	outcomeFlavorHope = "HOPE"
	outcomeFlavorFear = "FEAR"
)

// OutcomeFlavorFromCode normalizes a protobuf outcome string into the
// Daggerheart HOPE/FEAR flavor vocabulary.
func OutcomeFlavorFromCode(code string) string {
	switch strings.TrimSpace(code) {
	case pb.Outcome_ROLL_WITH_HOPE.String(),
		pb.Outcome_SUCCESS_WITH_HOPE.String(),
		pb.Outcome_FAILURE_WITH_HOPE.String(),
		pb.Outcome_CRITICAL_SUCCESS.String():
		return outcomeFlavorHope
	case pb.Outcome_ROLL_WITH_FEAR.String(),
		pb.Outcome_SUCCESS_WITH_FEAR.String(),
		pb.Outcome_FAILURE_WITH_FEAR.String():
		return outcomeFlavorFear
	default:
		return ""
	}
}

// OutcomeSuccessFromCode reports whether a protobuf outcome string is known and
// whether it counts as a success.
func OutcomeSuccessFromCode(code string) (bool, bool) {
	switch strings.TrimSpace(code) {
	case pb.Outcome_SUCCESS_WITH_HOPE.String(),
		pb.Outcome_SUCCESS_WITH_FEAR.String(),
		pb.Outcome_CRITICAL_SUCCESS.String():
		return true, true
	case pb.Outcome_FAILURE_WITH_HOPE.String(),
		pb.Outcome_FAILURE_WITH_FEAR.String(),
		pb.Outcome_ROLL_WITH_HOPE.String(),
		pb.Outcome_ROLL_WITH_FEAR.String():
		return false, true
	default:
		return false, false
	}
}

// OutcomeCodeToProto maps a stored outcome string back to protobuf.
func OutcomeCodeToProto(code string) pb.Outcome {
	switch strings.TrimSpace(code) {
	case pb.Outcome_ROLL_WITH_HOPE.String():
		return pb.Outcome_ROLL_WITH_HOPE
	case pb.Outcome_ROLL_WITH_FEAR.String():
		return pb.Outcome_ROLL_WITH_FEAR
	case pb.Outcome_SUCCESS_WITH_HOPE.String():
		return pb.Outcome_SUCCESS_WITH_HOPE
	case pb.Outcome_SUCCESS_WITH_FEAR.String():
		return pb.Outcome_SUCCESS_WITH_FEAR
	case pb.Outcome_FAILURE_WITH_HOPE.String():
		return pb.Outcome_FAILURE_WITH_HOPE
	case pb.Outcome_FAILURE_WITH_FEAR.String():
		return pb.Outcome_FAILURE_WITH_FEAR
	case pb.Outcome_CRITICAL_SUCCESS.String():
		return pb.Outcome_CRITICAL_SUCCESS
	default:
		return pb.Outcome_OUTCOME_UNSPECIFIED
	}
}
