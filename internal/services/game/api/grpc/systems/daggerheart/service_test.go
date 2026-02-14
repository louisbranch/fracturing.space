package daggerheart

import (
	"context"
	"errors"
	"reflect"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/random"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestActionRollRejectsNilRequest(t *testing.T) {
	server := newTestService(42)

	_, err := server.ActionRoll(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestActionRollRejectsNegativeDifficulty(t *testing.T) {
	server := newTestService(42)

	negative := int32(-1)
	_, err := server.ActionRoll(context.Background(), &pb.ActionRollRequest{Difficulty: &negative})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestActionRollWithDifficulty(t *testing.T) {
	seed := int64(99)
	server := newTestService(seed)

	difficulty := int32(10)
	modifier := int32(2)
	response, err := server.ActionRoll(context.Background(), &pb.ActionRollRequest{
		Modifier:   modifier,
		Difficulty: &difficulty,
	})
	if err != nil {
		t.Fatalf("ActionRoll returned error: %v", err)
	}
	assertResponseMatches(t, response, seed, random.SeedSourceServer, commonv1.RollMode_LIVE, modifier, &difficulty)
}

func TestActionRollWithoutDifficulty(t *testing.T) {
	seed := int64(55)
	server := newTestService(seed)

	modifier := int32(-1)
	response, err := server.ActionRoll(context.Background(), &pb.ActionRollRequest{Modifier: modifier})
	if err != nil {
		t.Fatalf("ActionRoll returned error: %v", err)
	}
	if response.Difficulty != nil {
		t.Fatalf("ActionRoll difficulty = %v, want nil", *response.Difficulty)
	}
	assertResponseMatches(t, response, seed, random.SeedSourceServer, commonv1.RollMode_LIVE, modifier, nil)
}

func TestActionRollAcceptsReplaySeed(t *testing.T) {
	seed := uint64(101)
	server := newTestService(55)

	response, err := server.ActionRoll(context.Background(), &pb.ActionRollRequest{
		Modifier: 1,
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("ActionRoll returned error: %v", err)
	}
	assertResponseMatches(t, response, int64(seed), random.SeedSourceClient, commonv1.RollMode_REPLAY, 1, nil)
}

func TestActionRollWithAdvantage(t *testing.T) {
	seed := uint64(77)
	server := newTestService(11)

	response, err := server.ActionRoll(context.Background(), &pb.ActionRollRequest{
		Modifier:  1,
		Advantage: 1,
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("ActionRoll returned error: %v", err)
	}
	if response.GetAdvantageDie() == 0 {
		t.Fatal("expected advantage_die to be set")
	}
}

func TestActionRollSeedFailure(t *testing.T) {
	server := &DaggerheartService{
		seedFunc: func() (int64, error) {
			return 0, errors.New("seed failure")
		},
	}

	_, err := server.ActionRoll(context.Background(), &pb.ActionRollRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestDualityOutcomeRejectsNilRequest(t *testing.T) {
	server := newTestService(42)

	_, err := server.DualityOutcome(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDualityOutcomeRejectsInvalidDice(t *testing.T) {
	server := newTestService(42)

	_, err := server.DualityOutcome(context.Background(), &pb.DualityOutcomeRequest{
		Hope: 0,
		Fear: 12,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDualityOutcomeRejectsNegativeDifficulty(t *testing.T) {
	server := newTestService(42)

	negative := int32(-1)
	_, err := server.DualityOutcome(context.Background(), &pb.DualityOutcomeRequest{
		Hope:       6,
		Fear:       5,
		Difficulty: &negative,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
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
	assertStatusCode(t, err, codes.InvalidArgument)
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
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDualityProbabilityRejectsNegativeDifficulty(t *testing.T) {
	server := newTestService(42)

	negative := int32(-1)
	_, err := server.DualityProbability(context.Background(), &pb.DualityProbabilityRequest{
		Modifier:   1,
		Difficulty: negative,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
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
	assertStatusCode(t, err, codes.InvalidArgument)
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
		expectedOutcomes = append(expectedOutcomes, outcomeToProto(outcome))
	}
	if !reflect.DeepEqual(response.Outcomes, expectedOutcomes) {
		t.Fatalf("RulesVersion outcomes = %v, want %v", response.Outcomes, expectedOutcomes)
	}
}

func TestRollDiceRejectsNilRequest(t *testing.T) {
	server := newTestService(42)

	_, err := server.RollDice(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRollDiceRejectsMissingDice(t *testing.T) {
	server := newTestService(42)

	_, err := server.RollDice(context.Background(), &pb.RollDiceRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRollDiceRejectsInvalidDiceSpec(t *testing.T) {
	server := newTestService(42)

	_, err := server.RollDice(context.Background(), &pb.RollDiceRequest{
		Dice: []*pb.DiceSpec{{Sides: 0, Count: 1}},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRollDiceReturnsResults(t *testing.T) {
	seed := int64(13)
	server := newTestService(seed)

	response, err := server.RollDice(context.Background(), &pb.RollDiceRequest{
		Dice: []*pb.DiceSpec{
			{Sides: 6, Count: 2},
			{Sides: 8, Count: 1},
		},
	})
	if err != nil {
		t.Fatalf("RollDice returned error: %v", err)
	}
	assertRollDiceResponse(t, response, seed, random.SeedSourceServer, commonv1.RollMode_LIVE, []dice.Spec{{Sides: 6, Count: 2}, {Sides: 8, Count: 1}})
}

func TestRollDiceAcceptsReplaySeed(t *testing.T) {
	seed := uint64(21)
	server := newTestService(99)

	response, err := server.RollDice(context.Background(), &pb.RollDiceRequest{
		Dice: []*pb.DiceSpec{{Sides: 6, Count: 2}},
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("RollDice returned error: %v", err)
	}
	assertRollDiceResponse(t, response, int64(seed), random.SeedSourceClient, commonv1.RollMode_REPLAY, []dice.Spec{{Sides: 6, Count: 2}})
}

func TestRollDiceIgnoresLiveSeed(t *testing.T) {
	seed := uint64(21)
	server := newTestService(99)

	response, err := server.RollDice(context.Background(), &pb.RollDiceRequest{
		Dice: []*pb.DiceSpec{{Sides: 6, Count: 2}},
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_LIVE,
		},
	})
	if err != nil {
		t.Fatalf("RollDice returned error: %v", err)
	}
	assertRollDiceResponse(t, response, 99, random.SeedSourceServer, commonv1.RollMode_LIVE, []dice.Spec{{Sides: 6, Count: 2}})
}

func TestRollDiceSeedFailure(t *testing.T) {
	server := &DaggerheartService{
		seedFunc: func() (int64, error) {
			return 0, errors.New("seed failure")
		},
	}

	_, err := server.RollDice(context.Background(), &pb.RollDiceRequest{
		Dice: []*pb.DiceSpec{{Sides: 6, Count: 1}},
	})
	assertStatusCode(t, err, codes.Internal)
}

// assertStatusCode verifies the gRPC status code for an error.
func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}
	if statusErr.Code() != want {
		t.Fatalf("status code = %v, want %v", statusErr.Code(), want)
	}
}

// assertResponseMatches validates response fields against expectations.
func assertResponseMatches(t *testing.T, response *pb.ActionRollResponse, seed int64, seedSource string, rollMode commonv1.RollMode, modifier int32, difficulty *int32) {
	t.Helper()

	if response == nil {
		t.Fatal("ActionRoll response is nil")
	}
	if response.GetRng() == nil {
		t.Fatal("ActionRoll rng is nil")
	}
	if response.GetRng().GetSeedUsed() != uint64(seed) {
		t.Fatalf("ActionRoll seed_used = %d, want %d", response.GetRng().GetSeedUsed(), seed)
	}
	if response.GetRng().GetRngAlgo() != random.RngAlgoMathRandV1 {
		t.Fatalf("ActionRoll rng_algo = %q, want %q", response.GetRng().GetRngAlgo(), random.RngAlgoMathRandV1)
	}
	if response.GetRng().GetSeedSource() != seedSource {
		t.Fatalf("ActionRoll seed_source = %q, want %q", response.GetRng().GetSeedSource(), seedSource)
	}
	if response.GetRng().GetRollMode() != rollMode {
		t.Fatalf("ActionRoll roll_mode = %v, want %v", response.GetRng().GetRollMode(), rollMode)
	}

	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Modifier:   int(modifier),
		Difficulty: intPointer(difficulty),
		Seed:       seed,
	})
	if err != nil {
		t.Fatalf("RollAction returned error: %v", err)
	}

	if response.GetHope() != int32(result.Hope) || response.GetFear() != int32(result.Fear) {
		t.Fatalf("ActionRoll dice = (%d, %d), want (%d, %d)", response.GetHope(), response.GetFear(), result.Hope, result.Fear)
	}
	if response.GetModifier() != int32(result.Modifier) {
		t.Fatalf("ActionRoll modifier = %d, want %d", response.GetModifier(), result.Modifier)
	}
	if response.GetAdvantageDie() != int32(result.AdvantageDie) {
		t.Fatalf("ActionRoll advantage_die = %d, want %d", response.GetAdvantageDie(), result.AdvantageDie)
	}
	if response.GetAdvantageModifier() != int32(result.AdvantageModifier) {
		t.Fatalf("ActionRoll advantage_modifier = %d, want %d", response.GetAdvantageModifier(), result.AdvantageModifier)
	}
	if response.Total != int32(result.Total) {
		t.Fatalf("ActionRoll total = %d, want %d", response.Total, result.Total)
	}
	if response.IsCrit != result.IsCrit {
		t.Fatalf("ActionRoll is_crit = %t, want %t", response.IsCrit, result.IsCrit)
	}
	if response.MeetsDifficulty != result.MeetsDifficulty {
		t.Fatalf("ActionRoll meets_difficulty = %t, want %t", response.MeetsDifficulty, result.MeetsDifficulty)
	}
	if response.Outcome != outcomeToProto(result.Outcome) {
		t.Fatalf("ActionRoll outcome = %v, want %v", response.Outcome, outcomeToProto(result.Outcome))
	}
	if difficulty != nil && response.Difficulty == nil {
		t.Fatal("ActionRoll difficulty is nil, want value")
	}
	if difficulty != nil && response.Difficulty != nil && *response.Difficulty != *difficulty {
		t.Fatalf("ActionRoll difficulty = %d, want %d", *response.Difficulty, *difficulty)
	}
}

// assertOutcomeResponse validates duality outcome response fields against expectations.
func assertOutcomeResponse(t *testing.T, response *pb.DualityOutcomeResponse, request daggerheartdomain.OutcomeRequest) {
	t.Helper()

	if response == nil {
		t.Fatal("DualityOutcome response is nil")
	}

	result, err := daggerheartdomain.EvaluateOutcome(request)
	if err != nil {
		t.Fatalf("EvaluateOutcome returned error: %v", err)
	}

	if response.GetHope() != int32(result.Hope) || response.GetFear() != int32(result.Fear) {
		t.Fatalf("DualityOutcome dice = (%d, %d), want (%d, %d)", response.GetHope(), response.GetFear(), result.Hope, result.Fear)
	}
	if response.GetModifier() != int32(result.Modifier) {
		t.Fatalf("DualityOutcome modifier = %d, want %d", response.GetModifier(), result.Modifier)
	}
	if response.Total != int32(result.Total) {
		t.Fatalf("DualityOutcome total = %d, want %d", response.Total, result.Total)
	}
	if response.IsCrit != result.IsCrit {
		t.Fatalf("DualityOutcome is_crit = %t, want %t", response.IsCrit, result.IsCrit)
	}
	if response.MeetsDifficulty != result.MeetsDifficulty {
		t.Fatalf("DualityOutcome meets_difficulty = %t, want %t", response.MeetsDifficulty, result.MeetsDifficulty)
	}
	if response.Outcome != outcomeToProto(result.Outcome) {
		t.Fatalf("DualityOutcome outcome = %v, want %v", response.Outcome, outcomeToProto(result.Outcome))
	}
	if request.Difficulty != nil && response.Difficulty == nil {
		t.Fatal("DualityOutcome difficulty is nil, want value")
	}
	if request.Difficulty != nil && response.Difficulty != nil && *response.Difficulty != int32(*request.Difficulty) {
		t.Fatalf("DualityOutcome difficulty = %d, want %d", *response.Difficulty, *request.Difficulty)
	}
}

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
	if response.Outcome != outcomeToProto(result.Outcome) {
		t.Fatalf("DualityExplain outcome = %v, want %v", response.Outcome, outcomeToProto(result.Outcome))
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

// assertProbabilityResponse validates duality probability response fields against expectations.
func assertProbabilityResponse(t *testing.T, response *pb.DualityProbabilityResponse, request daggerheartdomain.ProbabilityRequest) {
	t.Helper()

	if response == nil {
		t.Fatal("DualityProbability response is nil")
	}

	result, err := daggerheartdomain.DualityProbability(request)
	if err != nil {
		t.Fatalf("DualityProbability returned error: %v", err)
	}

	if response.TotalOutcomes != int32(result.TotalOutcomes) {
		t.Fatalf("DualityProbability total_outcomes = %d, want %d", response.TotalOutcomes, result.TotalOutcomes)
	}
	if response.CritCount != int32(result.CritCount) {
		t.Fatalf("DualityProbability crit_count = %d, want %d", response.CritCount, result.CritCount)
	}
	if response.SuccessCount != int32(result.SuccessCount) {
		t.Fatalf("DualityProbability success_count = %d, want %d", response.SuccessCount, result.SuccessCount)
	}
	if response.FailureCount != int32(result.FailureCount) {
		t.Fatalf("DualityProbability failure_count = %d, want %d", response.FailureCount, result.FailureCount)
	}
	if len(response.GetOutcomeCounts()) != len(result.OutcomeCounts) {
		t.Fatalf("DualityProbability outcome count len = %d, want %d", len(response.GetOutcomeCounts()), len(result.OutcomeCounts))
	}

	for i, count := range response.GetOutcomeCounts() {
		want := result.OutcomeCounts[i]
		if count.Outcome != outcomeToProto(want.Outcome) {
			t.Fatalf("DualityProbability outcome[%d] = %v, want %v", i, count.Outcome, outcomeToProto(want.Outcome))
		}
		if count.Count != int32(want.Count) {
			t.Fatalf("DualityProbability count[%d] = %d, want %d", i, count.Count, want.Count)
		}
	}
}

// assertRollDiceResponse validates roll dice response fields against expectations.
func assertRollDiceResponse(t *testing.T, response *pb.RollDiceResponse, seed int64, seedSource string, rollMode commonv1.RollMode, specs []dice.Spec) {
	t.Helper()

	if response == nil {
		t.Fatal("RollDice response is nil")
	}
	if response.GetRng() == nil {
		t.Fatal("RollDice rng is nil")
	}
	if response.GetRng().GetSeedUsed() != uint64(seed) {
		t.Fatalf("RollDice seed_used = %d, want %d", response.GetRng().GetSeedUsed(), seed)
	}
	if response.GetRng().GetRngAlgo() != random.RngAlgoMathRandV1 {
		t.Fatalf("RollDice rng_algo = %q, want %q", response.GetRng().GetRngAlgo(), random.RngAlgoMathRandV1)
	}
	if response.GetRng().GetSeedSource() != seedSource {
		t.Fatalf("RollDice seed_source = %q, want %q", response.GetRng().GetSeedSource(), seedSource)
	}
	if response.GetRng().GetRollMode() != rollMode {
		t.Fatalf("RollDice roll_mode = %v, want %v", response.GetRng().GetRollMode(), rollMode)
	}

	result, err := dice.RollDice(dice.Request{
		Dice: specs,
		Seed: seed,
	})
	if err != nil {
		t.Fatalf("RollDice returned error: %v", err)
	}

	if len(response.GetRolls()) != len(result.Rolls) {
		t.Fatalf("RollDice roll count = %d, want %d", len(response.GetRolls()), len(result.Rolls))
	}
	if response.Total != int32(result.Total) {
		t.Fatalf("RollDice total = %d, want %d", response.Total, result.Total)
	}

	for i, roll := range response.GetRolls() {
		want := result.Rolls[i]
		if roll.GetSides() != int32(want.Sides) {
			t.Fatalf("RollDice roll[%d] sides = %d, want %d", i, roll.GetSides(), want.Sides)
		}
		if roll.GetTotal() != int32(want.Total) {
			t.Fatalf("RollDice roll[%d] total = %d, want %d", i, roll.GetTotal(), want.Total)
		}
		if len(roll.GetResults()) != len(want.Results) {
			t.Fatalf("RollDice roll[%d] results = %v, want %v", i, roll.GetResults(), want.Results)
		}
		for j, value := range roll.GetResults() {
			if value != int32(want.Results[j]) {
				t.Fatalf("RollDice roll[%d] result[%d] = %d, want %d", i, j, value, want.Results[j])
			}
		}
	}
}

// newTestService creates a handler with a fixed seed generator.
func newTestService(seed int64) *DaggerheartService {
	return &DaggerheartService{
		seedFunc: func() (int64, error) {
			return seed, nil
		},
	}
}

// intPointer converts a difficulty pointer to the duality package type.
func intPointer(value *int32) *int {
	if value == nil {
		return nil
	}

	converted := int(*value)
	return &converted
}

func stringPointer(value string) *string {
	return &value
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
