package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

func TestSessionGroupActionFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionGroupActionFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingLeader(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingLeaderTrait(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingDifficulty(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1", LeaderTrait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingSupporters(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1", LeaderTrait: "agility", Difficulty: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- SessionTagTeamFlow tests ---
