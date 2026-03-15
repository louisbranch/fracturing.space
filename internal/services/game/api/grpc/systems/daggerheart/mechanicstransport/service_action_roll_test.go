package mechanicstransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
	"google.golang.org/grpc/codes"
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
