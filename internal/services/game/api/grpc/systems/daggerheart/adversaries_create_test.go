package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"google.golang.org/grpc/codes"
)

func TestCreateAdversary_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateAdversary_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.CreateAdversary(context.Background(), adversaryCreateRequest(testAdversaryEntryGoblinID))
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateAdversary_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		SessionId:        testAdversarySessionID,
		SceneId:          testAdversarySceneID,
		AdversaryEntryId: testAdversaryEntryGoblinID,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateAdversary_MissingAdversaryEntryID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1",
		SessionId:  testAdversarySessionID,
		SceneId:    testAdversarySceneID,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateAdversary_CampaignNotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId:       "nonexistent",
		SessionId:        testAdversarySessionID,
		SceneId:          testAdversarySceneID,
		AdversaryEntryId: testAdversaryEntryGoblinID,
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestCreateAdversary_NonDaggerheartCampaign(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId:       "camp-non-dh",
		SessionId:        testAdversarySessionID,
		SceneId:          testAdversarySceneID,
		AdversaryEntryId: testAdversaryEntryGoblinID,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateAdversary_Success(t *testing.T) {
	svc := newAdversaryTestService()
	req := adversaryCreateRequest(testAdversaryEntryGoblinID)
	req.Notes = "A test goblin"
	resp, err := svc.CreateAdversary(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Adversary == nil {
		t.Fatal("expected adversary in response")
	}
	if resp.Adversary.Name != "Goblin" {
		t.Errorf("name = %q, want Goblin", resp.Adversary.Name)
	}
	if resp.Adversary.Kind != "bruiser" {
		t.Errorf("kind = %q, want bruiser", resp.Adversary.Kind)
	}
	if resp.Adversary.AdversaryEntryId != testAdversaryEntryGoblinID {
		t.Errorf("adversary_entry_id = %q, want %q", resp.Adversary.AdversaryEntryId, testAdversaryEntryGoblinID)
	}
	if resp.Adversary.Id == "" {
		t.Error("expected non-empty adversary ID")
	}
}

func TestCreateAdversary_WithSession(t *testing.T) {
	svc := newAdversaryTestService()
	resp, err := svc.CreateAdversary(context.Background(), adversaryCreateRequest(testAdversaryEntryGoblinID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Adversary.SessionId != testAdversarySessionID {
		t.Errorf("expected session_id = %s, got %q", testAdversarySessionID, resp.Adversary.SessionId)
	}
}

func TestCreateAdversary_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryTestService()
	engine := &dynamicDomainEngine{store: svc.stores.Event}
	svc.stores.Write.Executor = engine

	req := adversaryCreateRequest(testAdversaryEntryGoblinID)
	req.Notes = " note "
	_, err := svc.CreateAdversary(context.Background(), req)
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	if engine.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", engine.calls)
	}
	if engine.lastCommand.Type != command.Type("sys.daggerheart.adversary.create") {
		t.Fatalf("command type = %s, want %s", engine.lastCommand.Type, "sys.daggerheart.adversary.create")
	}
	if engine.lastCommand.SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", engine.lastCommand.SystemID, daggerheart.SystemID)
	}
	if engine.lastCommand.SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", engine.lastCommand.SystemVersion, daggerheart.SystemVersion)
	}

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Notes       string `json:"notes"`
	}
	if err := json.Unmarshal(engine.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.AdversaryID == "" {
		t.Fatal("expected adversary_id in command payload")
	}
	if payload.Name != "Goblin" {
		t.Fatalf("name = %s, want %s", payload.Name, "Goblin")
	}
	if payload.Kind != "bruiser" {
		t.Fatalf("kind = %s, want %s", payload.Kind, "bruiser")
	}
	if payload.Notes != "note" {
		t.Fatalf("notes = %s, want %s", payload.Notes, "note")
	}
}

func TestCreateAdversary_SessionNotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId:       "camp-1",
		SessionId:        "nonexistent",
		SceneId:          testAdversarySceneID,
		AdversaryEntryId: testAdversaryEntryGoblinID,
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestCreateAdversary_MissingSceneID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId:       "camp-1",
		SessionId:        testAdversarySessionID,
		AdversaryEntryId: testAdversaryEntryGoblinID,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
