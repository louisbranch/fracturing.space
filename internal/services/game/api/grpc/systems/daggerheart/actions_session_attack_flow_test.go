package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

// --- SessionAttackFlow tests ---

func TestSessionAttackFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAttackFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingTrait(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingTargetId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingDamage(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1", Trait: "agility", TargetId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingDamageType(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1", Trait: "agility", TargetId: "adv-1",
		Damage: &pb.DaggerheartAttackDamageSpec{},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
