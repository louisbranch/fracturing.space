package mechanicstransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"google.golang.org/grpc/codes"
)

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
