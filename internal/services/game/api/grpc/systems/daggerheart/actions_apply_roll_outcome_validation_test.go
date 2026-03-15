package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"google.golang.org/grpc/codes"
)

func TestApplyRollOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyRollOutcome(context.Background(), &pb.ApplyRollOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRollOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyRollOutcome(context.Background(), &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := workflowtransport.WithCampaignSessionMetadata(context.Background(), "camp-1", "")
	_, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := workflowtransport.WithCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	svc.stores.Write.Executor = nil

	rollEvent := newRollEvent(t, "req-roll-outcome-required").appendTo(eventStore)

	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-outcome-required")
	_, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	assertStatusCode(t, err, codes.Internal)
}
