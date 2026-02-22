package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/codes"
)

func TestSessionAdversaryActionCheck_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryActionCheck_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		SessionId: "sess-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryActionCheck_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryActionCheck_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryActionCheck_RequiresDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	svc.stores.Domain = nil
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck returned error: %v", err)
	}
}

func TestSessionAdversaryActionCheck_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := grpcmeta.WithRequestID(context.Background(), "req-adv-action-success")
	resp, err := svc.SessionAdversaryActionCheck(ctx, &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
}

func TestSessionAdversaryActionCheck_DoesNotRequireDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	resp, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
}

// --- SessionAdversaryAttackFlow tests ---
