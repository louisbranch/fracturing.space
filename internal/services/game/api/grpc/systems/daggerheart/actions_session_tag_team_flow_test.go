package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

func TestSessionTagTeamFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionTagTeamFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingDifficulty(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingFirst(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingSecond(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
		First: &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_SameParticipant(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "strength"},
		SelectedCharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
