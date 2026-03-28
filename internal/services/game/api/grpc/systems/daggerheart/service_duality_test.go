package daggerheart

import (
	"context"
	"reflect"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestDualityOutcomeRejectsNilRequest(t *testing.T) {
	server := newTestService(42)

	_, err := server.DualityOutcome(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestDualityOutcomeRejectsInvalidDice(t *testing.T) {
	server := newTestService(42)

	_, err := server.DualityOutcome(context.Background(), &pb.DualityOutcomeRequest{
		Hope: 0,
		Fear: 12,
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestDualityOutcomeRejectsNegativeDifficulty(t *testing.T) {
	server := newTestService(42)

	negative := int32(-1)
	_, err := server.DualityOutcome(context.Background(), &pb.DualityOutcomeRequest{
		Hope:       6,
		Fear:       5,
		Difficulty: &negative,
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestDualityOutcomeReturnsResults(t *testing.T) {
	server := newTestService(42)

	difficulty := int32(10)
	response, err := server.DualityOutcome(context.Background(), &pb.DualityOutcomeRequest{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: &difficulty,
	})
	if err != nil {
		t.Fatalf("DualityOutcome returned error: %v", err)
	}
	assertOutcomeResponse(t, response, daggerheartdomain.OutcomeRequest{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: intPointer(&difficulty),
	})
}

func TestDualityExplainRejectsNilRequest(t *testing.T) {
	server := newTestService(42)

	_, err := server.DualityExplain(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestDualityExplainReturnsExplanation(t *testing.T) {
	server := newTestService(42)

	difficulty := int32(10)
	response, err := server.DualityExplain(context.Background(), &pb.DualityExplainRequest{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: &difficulty,
		RequestId:  stringPointer("trace-123"),
	})
	if err != nil {
		t.Fatalf("DualityExplain returned error: %v", err)
	}
	assertExplainResponse(t, response, daggerheartdomain.OutcomeRequest{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: intPointer(&difficulty),
	})
}

func TestMechanicsOutcomeConsistency(t *testing.T) {
	server := newTestService(42)

	difficulty := int32(12)
	modifier := int32(-1)
	seed := uint64(100)

	actionResponse, err := server.ActionRoll(context.Background(), &pb.ActionRollRequest{
		Modifier:   modifier,
		Difficulty: &difficulty,
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("ActionRoll returned error: %v", err)
	}

	outcomeResponse, err := server.DualityOutcome(context.Background(), &pb.DualityOutcomeRequest{
		Hope:       actionResponse.GetHope(),
		Fear:       actionResponse.GetFear(),
		Modifier:   modifier,
		Difficulty: &difficulty,
	})
	if err != nil {
		t.Fatalf("DualityOutcome returned error: %v", err)
	}

	explainResponse, err := server.DualityExplain(context.Background(), &pb.DualityExplainRequest{
		Hope:       actionResponse.GetHope(),
		Fear:       actionResponse.GetFear(),
		Modifier:   modifier,
		Difficulty: &difficulty,
	})
	if err != nil {
		t.Fatalf("DualityExplain returned error: %v", err)
	}

	if outcomeResponse.GetOutcome() != actionResponse.GetOutcome() {
		t.Fatalf("outcome mismatch: action %v, outcome %v", actionResponse.GetOutcome(), outcomeResponse.GetOutcome())
	}
	if explainResponse.GetOutcome() != actionResponse.GetOutcome() {
		t.Fatalf("outcome mismatch: action %v, explain %v", actionResponse.GetOutcome(), explainResponse.GetOutcome())
	}
	if outcomeResponse.GetTotal() != actionResponse.GetTotal() {
		t.Fatalf("total mismatch: action %d, outcome %d", actionResponse.GetTotal(), outcomeResponse.GetTotal())
	}
	if explainResponse.GetTotal() != actionResponse.GetTotal() {
		t.Fatalf("total mismatch: action %d, explain %d", actionResponse.GetTotal(), explainResponse.GetTotal())
	}
	if outcomeResponse.GetIsCrit() != actionResponse.GetIsCrit() {
		t.Fatalf("crit mismatch: action %t, outcome %t", actionResponse.GetIsCrit(), outcomeResponse.GetIsCrit())
	}
	if explainResponse.GetIsCrit() != actionResponse.GetIsCrit() {
		t.Fatalf("crit mismatch: action %t, explain %t", actionResponse.GetIsCrit(), explainResponse.GetIsCrit())
	}
	if outcomeResponse.GetMeetsDifficulty() != actionResponse.GetMeetsDifficulty() {
		t.Fatalf("meets difficulty mismatch: action %t, outcome %t", actionResponse.GetMeetsDifficulty(), outcomeResponse.GetMeetsDifficulty())
	}
	if explainResponse.GetMeetsDifficulty() != actionResponse.GetMeetsDifficulty() {
		t.Fatalf("meets difficulty mismatch: action %t, explain %t", actionResponse.GetMeetsDifficulty(), explainResponse.GetMeetsDifficulty())
	}
}

func TestDualityProbabilityRejectsNilRequest(t *testing.T) {
	server := newTestService(42)

	_, err := server.DualityProbability(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestDualityProbabilityRejectsNegativeDifficulty(t *testing.T) {
	server := newTestService(42)

	negative := int32(-1)
	_, err := server.DualityProbability(context.Background(), &pb.DualityProbabilityRequest{
		Modifier:   1,
		Difficulty: negative,
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestDualityProbabilityReturnsCounts(t *testing.T) {
	server := newTestService(42)

	response, err := server.DualityProbability(context.Background(), &pb.DualityProbabilityRequest{
		Modifier:   0,
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}
	assertProbabilityResponse(t, response, daggerheartdomain.ProbabilityRequest{Modifier: 0, Difficulty: 10})
}

func TestRulesVersionRejectsNilRequest(t *testing.T) {
	server := newTestService(42)

	_, err := server.RulesVersion(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestRulesVersionReturnsMetadata(t *testing.T) {
	server := newTestService(42)

	response, err := server.RulesVersion(context.Background(), &pb.RulesVersionRequest{})
	if err != nil {
		t.Fatalf("RulesVersion returned error: %v", err)
	}
	if response == nil {
		t.Fatal("RulesVersion response is nil")
	}

	metadata := daggerheartdomain.RulesVersion()
	if response.System != metadata.System {
		t.Fatalf("RulesVersion system = %q, want %q", response.System, metadata.System)
	}
	if response.Module != metadata.Module {
		t.Fatalf("RulesVersion module = %q, want %q", response.Module, metadata.Module)
	}
	if response.RulesVersion != metadata.RulesVersion {
		t.Fatalf("RulesVersion rules_version = %q, want %q", response.RulesVersion, metadata.RulesVersion)
	}
	if response.DiceModel != metadata.DiceModel {
		t.Fatalf("RulesVersion dice_model = %q, want %q", response.DiceModel, metadata.DiceModel)
	}
	if response.TotalFormula != metadata.TotalFormula {
		t.Fatalf("RulesVersion total_formula = %q, want %q", response.TotalFormula, metadata.TotalFormula)
	}
	if response.CritRule != metadata.CritRule {
		t.Fatalf("RulesVersion crit_rule = %q, want %q", response.CritRule, metadata.CritRule)
	}
	if response.DifficultyRule != metadata.DifficultyRule {
		t.Fatalf("RulesVersion difficulty_rule = %q, want %q", response.DifficultyRule, metadata.DifficultyRule)
	}

	expectedOutcomes := make([]pb.Outcome, 0, len(metadata.Outcomes))
	for _, outcome := range metadata.Outcomes {
		expectedOutcomes = append(expectedOutcomes, wantOutcomeProto(outcome))
	}
	if !reflect.DeepEqual(response.Outcomes, expectedOutcomes) {
		t.Fatalf("RulesVersion outcomes = %v, want %v", response.Outcomes, expectedOutcomes)
	}
}
