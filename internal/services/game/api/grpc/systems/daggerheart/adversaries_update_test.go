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
		Notes:       wrapperspb.String("New note"),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateAdversary_MissingAdversaryID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId: "camp-1",
		Notes:      wrapperspb.String("New note"),
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

	createResp, err := svc.CreateAdversary(context.Background(), adversaryCreateRequest(testAdversaryEntryGoblinID))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updateResp, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		SceneId:     testAdversaryAltSceneID,
		Notes:       wrapperspb.String("Upgraded"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updateResp.Adversary.Notes != "Upgraded" {
		t.Errorf("notes = %q, want Upgraded", updateResp.Adversary.Notes)
	}
	if updateResp.Adversary.SceneId != testAdversaryAltSceneID {
		t.Errorf("scene_id = %q, want %q", updateResp.Adversary.SceneId, testAdversaryAltSceneID)
	}
}

func TestUpdateAdversary_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryTestService()
	engine := &dynamicDomainEngine{store: svc.stores.Event}
	svc.stores.Write.Executor = engine

	createResp, err := svc.CreateAdversary(context.Background(), adversaryCreateRequest(testAdversaryEntryGoblinID))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	engine.calls = 0

	_, err = svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		Notes:       wrapperspb.String("Upgraded"),
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
		Notes       string `json:"notes"`
	}
	if err := json.Unmarshal(engine.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.AdversaryID != createResp.Adversary.Id {
		t.Fatalf("adversary_id = %s, want %s", payload.AdversaryID, createResp.Adversary.Id)
	}
	if payload.Notes != "Upgraded" {
		t.Fatalf("notes = %s, want %s", payload.Notes, "Upgraded")
	}
}

func TestUpdateAdversary_NotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: "nonexistent",
		Notes:       wrapperspb.String("New note"),
	})
	assertStatusCode(t, err, codes.NotFound)
}
