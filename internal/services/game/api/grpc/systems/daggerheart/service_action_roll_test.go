package daggerheart

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestActionRollRejectsNilRequest(t *testing.T) {
	server := newTestService(42)

	_, err := server.ActionRoll(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestActionRollRejectsNegativeDifficulty(t *testing.T) {
	server := newTestService(42)

	negative := int32(-1)
	_, err := server.ActionRoll(context.Background(), &pb.ActionRollRequest{Difficulty: &negative})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
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
	grpcassert.StatusCode(t, err, codes.Internal)
}
