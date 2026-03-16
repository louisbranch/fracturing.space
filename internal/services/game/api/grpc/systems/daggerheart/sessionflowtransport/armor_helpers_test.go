package sessionflowtransport

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerRollArmorFeatureDieRejectsInvalidSides(t *testing.T) {
	t.Parallel()

	handler := NewHandler(Dependencies{})
	_, err := handler.rollArmorFeatureDie(nil, 0)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerRollArmorFeatureDieRejectsOutOfRangeReplaySeed(t *testing.T) {
	t.Parallel()

	handler := NewHandler(Dependencies{
		SeedFunc: func() (int64, error) {
			t.Fatal("seed func should not run for replay seeds")
			return 0, nil
		},
	})
	overflow := ^uint64(0) >> 1
	overflow++
	_, err := handler.rollArmorFeatureDie(&commonv1.RngRequest{
		RollMode: commonv1.RollMode_REPLAY,
		Seed:     &overflow,
	}, 6)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerRollArmorFeatureDieUsesReplaySeed(t *testing.T) {
	t.Parallel()

	seed := uint64(11)
	handler := NewHandler(Dependencies{
		SeedFunc: func() (int64, error) {
			t.Fatal("seed func should not run for replay seeds")
			return 0, nil
		},
	})
	got, err := handler.rollArmorFeatureDie(&commonv1.RngRequest{
		RollMode: commonv1.RollMode_REPLAY,
		Seed:     &seed,
	}, 4)
	if err != nil {
		t.Fatalf("rollArmorFeatureDie returned error: %v", err)
	}

	want, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 4, Count: 1}},
		Seed: int64(seed),
	})
	if err != nil {
		t.Fatalf("dice.RollDice returned error: %v", err)
	}
	if got != want.Total {
		t.Fatalf("result = %d, want %d", got, want.Total)
	}
}
