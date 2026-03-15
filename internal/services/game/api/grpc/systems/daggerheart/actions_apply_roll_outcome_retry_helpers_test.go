package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

// fearRollEvent creates a roll event builder pre-configured for FAILURE_WITH_FEAR
// with gm_move enabled for retry and complication-gate tests.
func fearRollEvent(t *testing.T, requestID string) *rollEventBuilder {
	t.Helper()
	return newRollEvent(t, requestID).
		withOutcome(pb.Outcome_FAILURE_WITH_FEAR.String()).
		withResults(map[string]any{"d20": 1}).
		withSystemData("gm_move", true)
}
