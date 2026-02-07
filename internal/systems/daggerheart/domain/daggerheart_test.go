package domain

import (
	"errors"
	"testing"
)

func TestRollAction(t *testing.T) {
	diff := func(d int) *int {
		return &d
	}

	tcs := []struct {
		wantOutcome         Outcome
		seed                int64
		modifier            int
		difficulty          *int
		wantHope            int
		wantFear            int
		wantTotal           int
		wantIsCrit          bool
		wantMeetsDifficulty bool
	}{
		{
			wantOutcome:         OutcomeCriticalSuccess,
			seed:                0,
			modifier:            0,
			difficulty:          nil,
			wantHope:            7,
			wantFear:            7,
			wantTotal:           14,
			wantIsCrit:          true,
			wantMeetsDifficulty: false,
		},
		{
			wantOutcome:         OutcomeRollWithHope,
			seed:                1,
			modifier:            0,
			difficulty:          nil,
			wantHope:            6,
			wantFear:            4,
			wantTotal:           10,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			wantOutcome:         OutcomeRollWithFear,
			seed:                3,
			modifier:            0,
			difficulty:          nil,
			wantHope:            5,
			wantFear:            6,
			wantTotal:           11,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			wantOutcome:         OutcomeSuccessWithHope,
			seed:                1,
			modifier:            0,
			difficulty:          diff(9),
			wantHope:            6,
			wantFear:            4,
			wantTotal:           10,
			wantIsCrit:          false,
			wantMeetsDifficulty: true,
		},
		{
			wantOutcome:         OutcomeSuccessWithFear,
			seed:                3,
			modifier:            0,
			difficulty:          diff(10),
			wantHope:            5,
			wantFear:            6,
			wantTotal:           11,
			wantIsCrit:          false,
			wantMeetsDifficulty: true,
		},
		{
			wantOutcome:         OutcomeFailureWithHope,
			seed:                1,
			modifier:            -1,
			difficulty:          diff(10),
			wantHope:            6,
			wantFear:            4,
			wantTotal:           9,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			wantOutcome:         OutcomeFailureWithFear,
			seed:                3,
			modifier:            -2,
			difficulty:          diff(10),
			wantHope:            5,
			wantFear:            6,
			wantTotal:           9,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
	}

	for _, tc := range tcs {
		result, err := RollAction(ActionRequest{
			Modifier:   tc.modifier,
			Difficulty: tc.difficulty,
			Seed:       tc.seed,
		})
		if err != nil {
			t.Fatalf("RollAction returned error: %v", err)
		}
		if result.Hope != tc.wantHope || result.Fear != tc.wantFear || result.Total != tc.wantTotal || result.Outcome != tc.wantOutcome || result.IsCrit != tc.wantIsCrit || result.MeetsDifficulty != tc.wantMeetsDifficulty {
			t.Errorf("RollAction(%d, %v) = (%d, %d, %d, %v, %t, %t), want (%d, %d, %d, %v, %t, %t)", tc.modifier, tc.difficulty, result.Hope, result.Fear, result.Total, result.Outcome, result.IsCrit, result.MeetsDifficulty, tc.wantHope, tc.wantFear, tc.wantTotal, tc.wantOutcome, tc.wantIsCrit, tc.wantMeetsDifficulty)
		}
	}
}

func TestRollActionRejectsNegativeDifficulty(t *testing.T) {
	difficulty := -1
	_, err := RollAction(ActionRequest{
		Modifier:   0,
		Difficulty: &difficulty,
		Seed:       0,
	})
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("RollAction error = %v, want %v", err, ErrInvalidDifficulty)
	}
}

func TestEvaluateOutcome(t *testing.T) {
	tcs := []struct {
		name                string
		request             OutcomeRequest
		wantOutcome         Outcome
		wantTotal           int
		wantIsCrit          bool
		wantMeetsDifficulty bool
	}{
		{
			name: "critical overrides difficulty",
			request: OutcomeRequest{
				Hope:       7,
				Fear:       7,
				Modifier:   0,
				Difficulty: intPtr(12),
			},
			wantOutcome:         OutcomeCriticalSuccess,
			wantTotal:           14,
			wantIsCrit:          true,
			wantMeetsDifficulty: true,
		},
		{
			name: "success with hope",
			request: OutcomeRequest{
				Hope:       10,
				Fear:       4,
				Modifier:   1,
				Difficulty: intPtr(10),
			},
			wantOutcome:         OutcomeSuccessWithHope,
			wantTotal:           15,
			wantIsCrit:          false,
			wantMeetsDifficulty: true,
		},
		{
			name: "failure with fear",
			request: OutcomeRequest{
				Hope:       2,
				Fear:       8,
				Modifier:   0,
				Difficulty: intPtr(11),
			},
			wantOutcome:         OutcomeFailureWithFear,
			wantTotal:           10,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			name: "roll with hope",
			request: OutcomeRequest{
				Hope:     9,
				Fear:     2,
				Modifier: 0,
			},
			wantOutcome:         OutcomeRollWithHope,
			wantTotal:           11,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
		{
			name: "roll with fear",
			request: OutcomeRequest{
				Hope:     3,
				Fear:     11,
				Modifier: 0,
			},
			wantOutcome:         OutcomeRollWithFear,
			wantTotal:           14,
			wantIsCrit:          false,
			wantMeetsDifficulty: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EvaluateOutcome(tc.request)
			if err != nil {
				t.Fatalf("EvaluateOutcome returned error: %v", err)
			}
			if result.Total != tc.wantTotal || result.Outcome != tc.wantOutcome || result.IsCrit != tc.wantIsCrit || result.MeetsDifficulty != tc.wantMeetsDifficulty {
				t.Fatalf("EvaluateOutcome = (%d, %v, %t, %t), want (%d, %v, %t, %t)", result.Total, result.Outcome, result.IsCrit, result.MeetsDifficulty, tc.wantTotal, tc.wantOutcome, tc.wantIsCrit, tc.wantMeetsDifficulty)
			}
		})
	}
}

func TestEvaluateOutcomeRejectsInvalidDice(t *testing.T) {
	_, err := EvaluateOutcome(OutcomeRequest{Hope: 0, Fear: 12})
	if !errors.Is(err, ErrInvalidDualityDie) {
		t.Fatalf("EvaluateOutcome error = %v, want %v", err, ErrInvalidDualityDie)
	}
}

func TestEvaluateOutcomeRejectsNegativeDifficulty(t *testing.T) {
	_, err := EvaluateOutcome(OutcomeRequest{Hope: 6, Fear: 5, Difficulty: intPtr(-1)})
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("EvaluateOutcome error = %v, want %v", err, ErrInvalidDifficulty)
	}
}

func TestExplainOutcomeProvidesSteps(t *testing.T) {
	difficulty := 10
	request := OutcomeRequest{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: &difficulty,
	}

	result, err := ExplainOutcome(request)
	if err != nil {
		t.Fatalf("ExplainOutcome returned error: %v", err)
	}

	expected, err := EvaluateOutcome(request)
	if err != nil {
		t.Fatalf("EvaluateOutcome returned error: %v", err)
	}
	if result.Outcome != expected.Outcome || result.Total != expected.Total || result.IsCrit != expected.IsCrit || result.MeetsDifficulty != expected.MeetsDifficulty {
		t.Fatalf("ExplainOutcome result mismatch with EvaluateOutcome")
	}
	if result.RulesVersion != RulesVersion().RulesVersion {
		t.Fatalf("ExplainOutcome rules version = %q, want %q", result.RulesVersion, RulesVersion().RulesVersion)
	}
	if result.Intermediates.BaseTotal != 14 {
		t.Fatalf("ExplainOutcome base_total = %d, want 14", result.Intermediates.BaseTotal)
	}
	if result.Intermediates.Total != 15 {
		t.Fatalf("ExplainOutcome total = %d, want 15", result.Intermediates.Total)
	}
	if !result.Intermediates.HopeGtFear || result.Intermediates.FearGtHope {
		t.Fatalf("ExplainOutcome hope/fear comparison mismatch")
	}
	if len(result.Steps) != 5 {
		t.Fatalf("ExplainOutcome steps = %d, want 5", len(result.Steps))
	}

	wantCodes := []string{"SUM_DICE", "APPLY_MODIFIER", "CHECK_CRIT", "CHECK_DIFFICULTY", "SELECT_OUTCOME"}
	for i, code := range wantCodes {
		if result.Steps[i].Code != code {
			t.Fatalf("ExplainOutcome step %d code = %q, want %q", i, result.Steps[i].Code, code)
		}
	}
	if got := structInt(t, result.Steps[0].Data, "base_total"); got != 14 {
		t.Fatalf("ExplainOutcome step SUM_DICE base_total = %d, want 14", got)
	}
}

func TestDualityProbabilityCounts(t *testing.T) {
	result, err := DualityProbability(ProbabilityRequest{Modifier: 0, Difficulty: 10})
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}
	if result.TotalOutcomes != 144 {
		t.Fatalf("total outcomes = %d, want 144", result.TotalOutcomes)
	}
	if result.CritCount != 12 {
		t.Fatalf("crit count = %d, want 12", result.CritCount)
	}
	if result.SuccessCount+result.FailureCount != result.TotalOutcomes {
		t.Fatalf("success+failure = %d, want %d", result.SuccessCount+result.FailureCount, result.TotalOutcomes)
	}
	countSum := 0
	for _, count := range result.OutcomeCounts {
		countSum += count.Count
	}
	if countSum != result.TotalOutcomes {
		t.Fatalf("outcome count sum = %d, want %d", countSum, result.TotalOutcomes)
	}
}

func TestDualityProbabilityCritsConstant(t *testing.T) {
	first, err := DualityProbability(ProbabilityRequest{Modifier: -2, Difficulty: 8})
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}
	second, err := DualityProbability(ProbabilityRequest{Modifier: 5, Difficulty: 18})
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}
	if first.CritCount != second.CritCount {
		t.Fatalf("crit count changed: %d vs %d", first.CritCount, second.CritCount)
	}
}

func TestDualityProbabilityRejectsNegativeDifficulty(t *testing.T) {
	_, err := DualityProbability(ProbabilityRequest{Modifier: 0, Difficulty: -1})
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("DualityProbability error = %v, want %v", err, ErrInvalidDifficulty)
	}
}

func intPtr(value int) *int {
	return &value
}

func TestOutcomeString(t *testing.T) {
	tests := []struct {
		outcome Outcome
		want    string
	}{
		{OutcomeUnspecified, "Unspecified"},
		{OutcomeRollWithHope, "Roll with hope"},
		{OutcomeRollWithFear, "Roll with fear"},
		{OutcomeSuccessWithHope, "Success with hope"},
		{OutcomeSuccessWithFear, "Success with fear"},
		{OutcomeFailureWithHope, "Failure with hope"},
		{OutcomeFailureWithFear, "Failure with fear"},
		{OutcomeCriticalSuccess, "Critical success"},
		{Outcome(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.outcome.String()
			if got != tt.want {
				t.Fatalf("Outcome(%d).String() = %q, want %q", tt.outcome, got, tt.want)
			}
		})
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
