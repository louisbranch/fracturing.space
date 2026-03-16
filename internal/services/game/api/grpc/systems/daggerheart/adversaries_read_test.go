package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/adversarytransport"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestGetAdversary_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetAdversary_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetAdversary_MissingAdversaryID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetAdversary_NotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: "nonexistent",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetAdversary_NonDaggerheartCampaign(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-non-dh", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestGetAdversary_Success(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), adversaryCreateRequest(testAdversaryEntryGoblinID))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	getResp, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: createResp.Adversary.Id,
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getResp.Adversary.Name != "Goblin" {
		t.Errorf("name = %q, want Goblin", getResp.Adversary.Name)
	}
}

func TestListAdversaries_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.ListAdversaries(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListAdversaries_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.ListAdversaries(context.Background(), &pb.DaggerheartListAdversariesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListAdversaries_EmptyResult(t *testing.T) {
	svc := newAdversaryTestService()
	resp, err := svc.ListAdversaries(context.Background(), &pb.DaggerheartListAdversariesRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Adversaries) != 0 {
		t.Errorf("expected 0 adversaries, got %d", len(resp.Adversaries))
	}
}

func TestListAdversaries_WithResults(t *testing.T) {
	svc := newAdversaryTestService()

	for _, entryID := range []string{testAdversaryEntryGoblinID, testAdversaryEntryOrcID} {
		_, err := svc.CreateAdversary(context.Background(), adversaryCreateRequest(entryID))
		if err != nil {
			t.Fatalf("create %s: %v", entryID, err)
		}
	}

	resp, err := svc.ListAdversaries(context.Background(), &pb.DaggerheartListAdversariesRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Adversaries) != 2 {
		t.Errorf("expected 2 adversaries, got %d", len(resp.Adversaries))
	}
}

func TestListAdversaries_FilterBySession(t *testing.T) {
	svc := newAdversaryTestService()

	req := adversaryCreateRequest(testAdversaryEntryGoblinID)
	_, err := svc.CreateAdversary(context.Background(), req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	altReq := adversaryCreateRequest(testAdversaryEntryOrcID)
	altReq.SessionId = testAdversaryAltSessionID
	altReq.SceneId = testAdversaryAltSceneID
	_, err = svc.CreateAdversary(context.Background(), altReq)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := svc.ListAdversaries(context.Background(), &pb.DaggerheartListAdversariesRequest{
		CampaignId: "camp-1",
		SessionId:  wrapperspb.String(testAdversarySessionID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Adversaries) != 1 {
		t.Errorf("expected 1 adversary, got %d", len(resp.Adversaries))
	}
}

func TestLoadAdversaryForSession_NotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := adversarytransport.LoadAdversaryForSession(context.Background(), svc.stores.Daggerheart, "camp-1", "sess-1", "nonexistent")
	assertStatusCode(t, err, codes.NotFound)
}

func TestLoadAdversaryForSession_WrongSession(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), adversaryCreateRequest(testAdversaryEntryGoblinID))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = adversarytransport.LoadAdversaryForSession(context.Background(), svc.stores.Daggerheart, "camp-1", "other-session", createResp.Adversary.Id)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestLoadAdversaryForSession_Success(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), adversaryCreateRequest(testAdversaryEntryGoblinID))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	a, err := adversarytransport.LoadAdversaryForSession(context.Background(), svc.stores.Daggerheart, "camp-1", "sess-1", createResp.Adversary.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if a.Name != "Goblin" {
		t.Errorf("name = %q, want Goblin", a.Name)
	}
}
