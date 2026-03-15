package mechanicstransport

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerActionRollRejectsNilRequest(t *testing.T) {
	_, err := newTestHandler(42).ActionRoll(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestHandlerActionRollReturnsDeterministicResponse(t *testing.T) {
	handler := newTestHandler(99)
	difficulty := int32(10)

	resp, err := handler.ActionRoll(context.Background(), &pb.ActionRollRequest{
		Modifier:   2,
		Difficulty: &difficulty,
	})
	if err != nil {
		t.Fatalf("ActionRoll: %v", err)
	}

	expected, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Modifier:   2,
		Difficulty: intPointer(&difficulty),
		Seed:       99,
	})
	if err != nil {
		t.Fatalf("RollAction: %v", err)
	}

	if resp.GetTotal() != int32(expected.Total) || resp.GetOutcome() != outcomeToProto(expected.Outcome) {
		t.Fatalf("ActionRoll mismatch: resp=%v expected=%+v", resp, expected)
	}
	if resp.GetRng().GetSeedUsed() != 99 || resp.GetRng().GetSeedSource() != random.SeedSourceServer {
		t.Fatalf("unexpected rng metadata: %+v", resp.GetRng())
	}
}

func TestHandlerActionRollSeedFailure(t *testing.T) {
	handler := NewHandler(func() (int64, error) { return 0, errors.New("boom") })
	_, err := handler.ActionRoll(context.Background(), &pb.ActionRollRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestHandlerDualityOutcomeAndExplain(t *testing.T) {
	handler := newTestHandler(42)
	difficulty := int32(10)

	outcomeResp, err := handler.DualityOutcome(context.Background(), &pb.DualityOutcomeRequest{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: &difficulty,
	})
	if err != nil {
		t.Fatalf("DualityOutcome: %v", err)
	}
	if outcomeResp.GetOutcome() == pb.Outcome_OUTCOME_UNSPECIFIED {
		t.Fatal("expected concrete outcome")
	}

	explainResp, err := handler.DualityExplain(context.Background(), &pb.DualityExplainRequest{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: &difficulty,
		RequestId:  stringPointer("trace-123"),
	})
	if err != nil {
		t.Fatalf("DualityExplain: %v", err)
	}
	if len(explainResp.GetSteps()) == 0 || explainResp.GetRulesVersion() == "" {
		t.Fatalf("unexpected explain response: %+v", explainResp)
	}
}

func TestHandlerDualityProbabilityAndRulesVersion(t *testing.T) {
	handler := newTestHandler(42)

	probabilityResp, err := handler.DualityProbability(context.Background(), &pb.DualityProbabilityRequest{
		Modifier:   0,
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("DualityProbability: %v", err)
	}
	if probabilityResp.GetTotalOutcomes() == 0 || len(probabilityResp.GetOutcomeCounts()) == 0 {
		t.Fatalf("unexpected probability response: %+v", probabilityResp)
	}

	rulesResp, err := handler.RulesVersion(context.Background(), &pb.RulesVersionRequest{})
	if err != nil {
		t.Fatalf("RulesVersion: %v", err)
	}
	if rulesResp.GetRulesVersion() == "" || len(rulesResp.GetOutcomes()) == 0 {
		t.Fatalf("unexpected rules version response: %+v", rulesResp)
	}
}

func TestHandlerRollDice(t *testing.T) {
	handler := newTestHandler(13)

	_, err := handler.RollDice(context.Background(), &pb.RollDiceRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)

	resp, err := handler.RollDice(context.Background(), &pb.RollDiceRequest{
		Dice: []*pb.DiceSpec{
			{Sides: 6, Count: 2},
			{Sides: 8, Count: 1},
		},
		Rng: &commonv1.RngRequest{
			RollMode: commonv1.RollMode_LIVE,
		},
	})
	if err != nil {
		t.Fatalf("RollDice: %v", err)
	}

	expected, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 6, Count: 2}, {Sides: 8, Count: 1}},
		Seed: 13,
	})
	if err != nil {
		t.Fatalf("dice.RollDice: %v", err)
	}
	if resp.GetTotal() != int32(expected.Total) || len(resp.GetRolls()) != len(expected.Rolls) {
		t.Fatalf("unexpected roll response: %+v expected=%+v", resp, expected)
	}
}

func newTestHandler(seed int64) *Handler {
	return NewHandler(func() (int64, error) { return seed, nil })
}

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

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	if got := status.Code(err); got != want {
		t.Fatalf("status code = %v, want %v (err=%v)", got, want, err)
	}
}
