package daggerheart

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"google.golang.org/grpc/codes"
)

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
