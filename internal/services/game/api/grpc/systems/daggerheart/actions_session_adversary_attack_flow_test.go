package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

func TestSessionAdversaryAttackFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryAttackFlow_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingTargetId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingDamage(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", AdversaryId: "adv-1", TargetId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingDamageType(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", AdversaryId: "adv-1", TargetId: "char-1",
		Damage: &pb.DaggerheartAttackDamageSpec{},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- SessionGroupActionFlow tests ---
