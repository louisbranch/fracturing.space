package domain

import (
	"errors"
	"reflect"
	"testing"
)

func TestExplainOutcome_ReturnsDeterministicExplanation(t *testing.T) {
	difficulty := 10
	result, err := ExplainOutcome(OutcomeRequest{
		Hope:       8,
		Fear:       4,
		Modifier:   1,
		Difficulty: &difficulty,
	})
	if err != nil {
		t.Fatalf("ExplainOutcome returned error: %v", err)
	}

	if result.RulesVersion != RulesVersion().RulesVersion {
		t.Fatalf("rules version = %q, want %q", result.RulesVersion, RulesVersion().RulesVersion)
	}
	if result.Intermediates.BaseTotal != 12 {
		t.Fatalf("base total = %d, want %d", result.Intermediates.BaseTotal, 12)
	}
	if result.Intermediates.Total != 13 {
		t.Fatalf("total = %d, want %d", result.Intermediates.Total, 13)
	}
	if !result.Intermediates.MeetsDifficulty {
		t.Fatal("expected meets difficulty")
	}
	if !result.Intermediates.HopeGtFear || result.Intermediates.FearGtHope {
		t.Fatalf("unexpected hope/fear comparison intermediates: %+v", result.Intermediates)
	}
	if result.Outcome != OutcomeSuccessWithHope {
		t.Fatalf("outcome = %v, want %v", result.Outcome, OutcomeSuccessWithHope)
	}

	if len(result.Steps) != 5 {
		t.Fatalf("steps = %d, want 5", len(result.Steps))
	}
	stepCodes := []string{
		result.Steps[0].Code,
		result.Steps[1].Code,
		result.Steps[2].Code,
		result.Steps[3].Code,
		result.Steps[4].Code,
	}
	if !reflect.DeepEqual(stepCodes, []string{"SUM_DICE", "APPLY_MODIFIER", "CHECK_CRIT", "CHECK_DIFFICULTY", "SELECT_OUTCOME"}) {
		t.Fatalf("step codes = %v, want deterministic sequence", stepCodes)
	}

	checkDifficulty := result.Steps[3].Data
	if present, ok := checkDifficulty["difficulty_present"].(bool); !ok || !present {
		t.Fatalf("difficulty_present = %v, want true", checkDifficulty["difficulty_present"])
	}
	if gotDifficulty, ok := checkDifficulty["difficulty"].(int); !ok || gotDifficulty != difficulty {
		t.Fatalf("difficulty = %v, want %d", checkDifficulty["difficulty"], difficulty)
	}
}

func TestExplainOutcome_InvalidDiceRejected(t *testing.T) {
	_, err := ExplainOutcome(OutcomeRequest{
		Hope: 0,
		Fear: 5,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidDualityDie) {
		t.Fatalf("error = %v, want ErrInvalidDualityDie", err)
	}
}

func TestDualityProbability_ComputesExactCounts(t *testing.T) {
	result, err := DualityProbability(ProbabilityRequest{
		Modifier:   0,
		Difficulty: 0,
	})
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}

	if result.TotalOutcomes != 144 {
		t.Fatalf("total outcomes = %d, want %d", result.TotalOutcomes, 144)
	}
	if result.CritCount != 12 {
		t.Fatalf("crit count = %d, want %d", result.CritCount, 12)
	}
	if result.SuccessCount != 144 {
		t.Fatalf("success count = %d, want %d", result.SuccessCount, 144)
	}
	if result.FailureCount != 0 {
		t.Fatalf("failure count = %d, want %d", result.FailureCount, 0)
	}

	wantCounts := []OutcomeCount{
		{Outcome: OutcomeCriticalSuccess, Count: 12},
		{Outcome: OutcomeSuccessWithHope, Count: 66},
		{Outcome: OutcomeSuccessWithFear, Count: 66},
		{Outcome: OutcomeFailureWithHope, Count: 0},
		{Outcome: OutcomeFailureWithFear, Count: 0},
	}
	if !reflect.DeepEqual(result.OutcomeCounts, wantCounts) {
		t.Fatalf("outcome counts = %+v, want %+v", result.OutcomeCounts, wantCounts)
	}
}

func TestDualityProbability_NegativeDifficultyRejected(t *testing.T) {
	_, err := DualityProbability(ProbabilityRequest{Difficulty: -1})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidDifficulty) {
		t.Fatalf("error = %v, want ErrInvalidDifficulty", err)
	}
}

func TestRollReaction_MirrorsActionRollAndSetsReactionFlags(t *testing.T) {
	difficulty := 15
	request := ReactionRequest{
		Modifier:   2,
		Difficulty: &difficulty,
		Seed:       12345,
		Advantage:  1,
	}

	reaction, err := RollReaction(request)
	if err != nil {
		t.Fatalf("RollReaction returned error: %v", err)
	}

	action, err := RollAction(ActionRequest{
		Modifier:   request.Modifier,
		Difficulty: request.Difficulty,
		Seed:       request.Seed,
		Advantage:  request.Advantage,
	})
	if err != nil {
		t.Fatalf("RollAction returned error: %v", err)
	}

	if reaction.ActionResult != action {
		t.Fatalf("reaction action result = %+v, want %+v", reaction.ActionResult, action)
	}
	if reaction.GeneratesHopeFear {
		t.Fatal("expected reaction to not generate hope/fear")
	}
	if reaction.AidAllowed {
		t.Fatal("expected reaction aid to be disallowed")
	}
	if reaction.TriggersGMMove {
		t.Fatal("expected reaction to not trigger GM move")
	}
	if !reaction.CritNegatesEffects {
		t.Fatal("expected reaction crit to negate effects")
	}
}

func TestRulesVersion_ReturnsDualityMetadata(t *testing.T) {
	got := RulesVersion()
	if got.System != "Daggerheart" {
		t.Fatalf("system = %q, want %q", got.System, "Daggerheart")
	}
	if got.Module != "Duality" {
		t.Fatalf("module = %q, want %q", got.Module, "Duality")
	}
	if got.RulesVersion != "1.0.0" {
		t.Fatalf("rules version = %q, want %q", got.RulesVersion, "1.0.0")
	}
	if got.DiceModel != "2d12" {
		t.Fatalf("dice model = %q, want %q", got.DiceModel, "2d12")
	}
	wantOutcomes := []Outcome{
		OutcomeRollWithHope,
		OutcomeRollWithFear,
		OutcomeSuccessWithHope,
		OutcomeSuccessWithFear,
		OutcomeFailureWithHope,
		OutcomeFailureWithFear,
		OutcomeCriticalSuccess,
	}
	if !reflect.DeepEqual(got.Outcomes, wantOutcomes) {
		t.Fatalf("outcomes = %v, want %v", got.Outcomes, wantOutcomes)
	}
}

func TestOutcomeString_UnknownOutcome(t *testing.T) {
	if got := Outcome(99).String(); got != "Unknown" {
		t.Fatalf("Outcome(99).String() = %q, want %q", got, "Unknown")
	}
}
