package daggerheart

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

const (
	outcomeFlavorHope = "HOPE"
	outcomeFlavorFear = "FEAR"
)

func outcomeFlavorFromCode(code string) string {
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

func outcomeSuccessFromCode(code string) (bool, bool) {
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

func outcomeCodeToProto(code string) pb.Outcome {
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
