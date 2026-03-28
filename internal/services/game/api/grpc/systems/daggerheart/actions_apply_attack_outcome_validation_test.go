package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestApplyAttackOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "s1",
		RollSeq:   1,
		Targets:   []string{"char-1"},
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestApplyAttackOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1, Targets: []string{"adv-1"},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		RollSeq: 1, Targets: []string{"adv-1"},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := workflowtransport.WithCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", Targets: []string{"adv-1"},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingTargets(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := workflowtransport.WithCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "s1",
		RollSeq:   1,
		Targets:   []string{"char-1"},
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryAttackOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1, Targets: []string{"char-1"},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		RollSeq: 1, Targets: []string{"char-1"},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := workflowtransport.WithCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", Targets: []string{"char-1"},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingTargets(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := workflowtransport.WithCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}
