package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestUpdateAdversary_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateAdversary_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		AdversaryId: "adv-1",
		Name:        wrapperspb.String("New Name"),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateAdversary_MissingAdversaryID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId: "camp-1",
		Name:       wrapperspb.String("New Name"),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateAdversary_NoFieldsProvided(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateAdversary_Success(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updateResp, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		Name:        wrapperspb.String("Hobgoblin"),
		Notes:       wrapperspb.String("Upgraded"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updateResp.Adversary.Name != "Hobgoblin" {
		t.Errorf("name = %q, want Hobgoblin", updateResp.Adversary.Name)
	}
}

func TestUpdateAdversary_UsesDomainEngine(t *testing.T) {
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

	_, err = svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		Name:        wrapperspb.String("Hobgoblin"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if engine.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", engine.calls)
	}
	if engine.lastCommand.Type != command.Type("sys.daggerheart.adversary.update") {
		t.Fatalf("command type = %s, want %s", engine.lastCommand.Type, "sys.daggerheart.adversary.update")
	}

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		Name        string `json:"name"`
	}
	if err := json.Unmarshal(engine.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.AdversaryID != createResp.Adversary.Id {
		t.Fatalf("adversary_id = %s, want %s", payload.AdversaryID, createResp.Adversary.Id)
	}
	if payload.Name != "Hobgoblin" {
		t.Fatalf("name = %s, want %s", payload.Name, "Hobgoblin")
	}
}

func TestUpdateAdversary_NotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: "nonexistent",
		Name:        wrapperspb.String("New Name"),
	})
	assertStatusCode(t, err, codes.NotFound)
}
