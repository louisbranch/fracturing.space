package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"google.golang.org/grpc/codes"
)

func TestDeleteAdversary_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.DeleteAdversary(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteAdversary_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteAdversary_MissingAdversaryID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteAdversary_NotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: "nonexistent",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestDeleteAdversary_Success(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	deleteResp, err := svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		Reason:      "Test cleanup",
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleteResp.Adversary.Name != "Goblin" {
		t.Errorf("expected deleted adversary name = Goblin, got %q", deleteResp.Adversary.Name)
	}

	_, err = svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: createResp.Adversary.Id,
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestDeleteAdversary_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryTestService()
	engine := &dynamicDomainEngine{store: svc.stores.Event}
	svc.stores.Write.Executor = engine

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	engine.calls = 0

	_, err = svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		Reason:      " cleanup ",
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if engine.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", engine.calls)
	}
	if engine.lastCommand.Type != command.Type("sys.daggerheart.adversary.delete") {
		t.Fatalf("command type = %s, want %s", engine.lastCommand.Type, "sys.daggerheart.adversary.delete")
	}

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		Reason      string `json:"reason"`
	}
	if err := json.Unmarshal(engine.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.AdversaryID != createResp.Adversary.Id {
		t.Fatalf("adversary_id = %s, want %s", payload.AdversaryID, createResp.Adversary.Id)
	}
	if payload.Reason != "cleanup" {
		t.Fatalf("reason = %s, want %s", payload.Reason, "cleanup")
	}
}
