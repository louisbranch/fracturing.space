package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

// --- ApplyRollOutcome tests ---
// --- ApplyReactionOutcome tests ---

func TestApplyReactionOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "s1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestApplyReactionOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		RollSeq: 1,
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := workflowtransport.WithCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyReactionOutcome(ctx, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	configureNoopDomain(svc)

	rollEvent := newRollEvent(t, "req-react-outcome-legacy").
		withRollKind(pb.RollKind_ROLL_KIND_REACTION).
		withHopeFear(false).
		withResults(map[string]any{"d20": 12}).
		appendTo(eventStore)

	ctx := testSessionCtx("camp-1", "sess-1", "req-react-outcome-legacy")
	resp, err := svc.ApplyReactionOutcome(ctx, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyReactionOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("expected character_id char-1, got %s", resp.CharacterId)
	}
	if resp.Result.GetOutcome() != pb.Outcome_SUCCESS_WITH_HOPE {
		t.Fatalf("expected outcome SUCCESS_WITH_HOPE, got %s", resp.Result.GetOutcome())
	}
	if !resp.Result.GetSuccess() {
		t.Fatal("expected reaction success")
	}
}
