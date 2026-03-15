package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

func TestSessionActionRoll_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionActionRoll_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		SessionId: "sess-1", CharacterId: "char-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "camp-1", CharacterId: "char-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_MissingTrait(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	assertStatusCode(t, err, codes.Internal)
}
