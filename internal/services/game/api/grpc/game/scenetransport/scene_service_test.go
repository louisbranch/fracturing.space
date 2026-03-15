package scenetransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- test helpers (local copies; not exported from root package) ---

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	// Simulate the ErrorConversionUnaryInterceptor: handlers may return
	// domain errors that the interceptor would convert to gRPC status.
	statusErr, ok := status.FromError(err)
	if !ok {
		err = grpcerror.HandleDomainError(err)
		statusErr, ok = status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %T", err)
		}
	}
	if statusErr.Code() != want {
		t.Fatalf("status code = %v, want %v (message: %s)", statusErr.Code(), want, statusErr.Message())
	}
}

func activeCampaignStore(campaignID string) *gametest.FakeCampaignStore {
	store := gametest.NewFakeCampaignStore()
	store.Campaigns[campaignID] = gametest.ActiveCampaignRecord(campaignID)
	return store
}

func sessionManagerParticipantStore(campaignID string) *gametest.FakeParticipantStore {
	store := gametest.NewFakeParticipantStore()
	store.Participants[campaignID] = map[string]storage.ParticipantRecord{
		"manager-1": gametest.ManagerParticipantRecord(campaignID, "manager-1"),
	}
	return store
}

type fakeDomainEngine struct {
	result      engine.Result
	lastCommand command.Command
}

func (f *fakeDomainEngine) Execute(_ context.Context, cmd command.Command) (engine.Result, error) {
	f.lastCommand = cmd
	return f.result, nil
}

func emptyDeps() Deps {
	return Deps{}
}

func depsWithAuth(campaignStore storage.CampaignStore, participantStore storage.ParticipantStore) Deps {
	return Deps{
		Auth: authz.PolicyDeps{
			Participant: participantStore,
		},
		Campaign: campaignStore,
	}
}

// --- GetScene ---

func TestGetScene_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.GetScene(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetScene_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.GetScene(context.Background(), &statev1.GetSceneRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetScene_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.GetScene(context.Background(), &statev1.GetSceneRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetScene_CampaignNotFound(t *testing.T) {
	svc := NewService(Deps{Campaign: gametest.NewFakeCampaignStore()})
	_, err := svc.GetScene(context.Background(), &statev1.GetSceneRequest{
		CampaignId: "nonexistent",
		SceneId:    "sc-1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetScene_ReturnsScene(t *testing.T) {
	campaignStore := activeCampaignStore("c1")
	participantStore := sessionManagerParticipantStore("c1")
	sceneStore := &fakeSceneStoreForService{
		scenes: map[string]storage.SceneRecord{
			"c1:sc-1": {
				CampaignID: "c1",
				SceneID:    "sc-1",
				SessionID:  "s-1",
				Name:       "Battle",
				Active:     true,
				CreatedAt:  time.Unix(1000, 0),
				UpdatedAt:  time.Unix(1000, 0),
			},
		},
	}
	sceneCharStore := &fakeSceneCharStoreForService{}

	svc := NewService(Deps{
		Auth:           authz.PolicyDeps{Participant: participantStore},
		Campaign:       campaignStore,
		Scene:          sceneStore,
		SceneCharacter: sceneCharStore,
	})
	resp, err := svc.GetScene(gametest.ContextWithParticipantID("manager-1"), &statev1.GetSceneRequest{
		CampaignId: "c1",
		SceneId:    "sc-1",
	})
	if err != nil {
		t.Fatalf("get scene: %v", err)
	}
	if resp.GetScene().GetName() != "Battle" {
		t.Errorf("name = %q, want %q", resp.GetScene().GetName(), "Battle")
	}
	if !resp.GetScene().GetActive() {
		t.Error("expected active")
	}
}

// --- ListScenes ---

func TestListScenes_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ListScenes(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListScenes_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ListScenes(context.Background(), &statev1.ListScenesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListScenes_MissingSessionId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ListScenes(context.Background(), &statev1.ListScenesRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListScenes_ReturnsEmpty(t *testing.T) {
	campaignStore := activeCampaignStore("c1")
	participantStore := sessionManagerParticipantStore("c1")
	sceneStore := &fakeSceneStoreForService{
		scenes: map[string]storage.SceneRecord{},
	}

	svc := NewService(Deps{
		Auth:     authz.PolicyDeps{Participant: participantStore},
		Campaign: campaignStore,
		Scene:    sceneStore,
	})
	resp, err := svc.ListScenes(gametest.ContextWithParticipantID("manager-1"), &statev1.ListScenesRequest{
		CampaignId: "c1",
		SessionId:  "s-1",
	})
	if err != nil {
		t.Fatalf("list scenes: %v", err)
	}
	if len(resp.GetScenes()) != 0 {
		t.Errorf("expected empty, got %d", len(resp.GetScenes()))
	}
}

// --- Write RPCs nil/empty checks ---

func TestCreateScene_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.CreateScene(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateScene_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.CreateScene(context.Background(), &statev1.CreateSceneRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateScene_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.UpdateScene(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateScene_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.UpdateScene(context.Background(), &statev1.UpdateSceneRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndScene_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.EndScene(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndScene_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.EndScene(context.Background(), &statev1.EndSceneRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAddCharacterToScene_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AddCharacterToScene(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRemoveCharacterFromScene_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.RemoveCharacterFromScene(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestTransferCharacter_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.TransferCharacter(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestTransitionScene_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.TransitionScene(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestOpenSceneGate_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.OpenSceneGate(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveSceneGate_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ResolveSceneGate(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSceneGate_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AbandonSceneGate(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSceneSpotlight_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.SetSceneSpotlight(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClearSceneSpotlight_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ClearSceneSpotlight(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- Write RPCs missing campaign ID ---

func TestAddCharacterToScene_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AddCharacterToScene(context.Background(), &statev1.AddCharacterToSceneRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRemoveCharacterFromScene_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.RemoveCharacterFromScene(context.Background(), &statev1.RemoveCharacterFromSceneRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestTransferCharacter_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.TransferCharacter(context.Background(), &statev1.TransferCharacterRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestTransitionScene_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.TransitionScene(context.Background(), &statev1.TransitionSceneRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestOpenSceneGate_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.OpenSceneGate(context.Background(), &statev1.OpenSceneGateRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveSceneGate_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ResolveSceneGate(context.Background(), &statev1.ResolveSceneGateRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSceneGate_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AbandonSceneGate(context.Background(), &statev1.AbandonSceneGateRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSceneSpotlight_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.SetSceneSpotlight(context.Background(), &statev1.SetSceneSpotlightRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClearSceneSpotlight_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ClearSceneSpotlight(context.Background(), &statev1.ClearSceneSpotlightRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- Write RPCs application-level validation (through service layer) ---

func TestCreateScene_MissingSessionId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.CreateScene(context.Background(), &statev1.CreateSceneRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateScene_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.UpdateScene(context.Background(), &statev1.UpdateSceneRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndScene_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.EndScene(context.Background(), &statev1.EndSceneRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAddCharacterToScene_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AddCharacterToScene(context.Background(), &statev1.AddCharacterToSceneRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAddCharacterToScene_MissingCharacterId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AddCharacterToScene(context.Background(), &statev1.AddCharacterToSceneRequest{
		CampaignId: "c1", SceneId: "sc-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRemoveCharacterFromScene_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.RemoveCharacterFromScene(context.Background(), &statev1.RemoveCharacterFromSceneRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRemoveCharacterFromScene_MissingCharacterId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.RemoveCharacterFromScene(context.Background(), &statev1.RemoveCharacterFromSceneRequest{
		CampaignId: "c1", SceneId: "sc-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestTransferCharacter_MissingSourceSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.TransferCharacter(context.Background(), &statev1.TransferCharacterRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestTransferCharacter_MissingTargetSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.TransferCharacter(context.Background(), &statev1.TransferCharacterRequest{
		CampaignId: "c1", SourceSceneId: "sc-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestTransferCharacter_MissingCharacterId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.TransferCharacter(context.Background(), &statev1.TransferCharacterRequest{
		CampaignId: "c1", SourceSceneId: "sc-1", TargetSceneId: "sc-2",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestTransitionScene_MissingSourceSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.TransitionScene(context.Background(), &statev1.TransitionSceneRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestTransitionScene_UsesSourceSceneSessionID(t *testing.T) {
	campaignStore := activeCampaignStore("c1")
	participantStore := sessionManagerParticipantStore("c1")
	sceneStore := &fakeSceneStoreForService{
		scenes: map[string]storage.SceneRecord{
			"c1:sc-1": {
				CampaignID: "c1",
				SceneID:    "sc-1",
				SessionID:  "sess-1",
				Name:       "Room A",
				Active:     true,
				CreatedAt:  time.Unix(1000, 0),
				UpdatedAt:  time.Unix(1000, 0),
			},
		},
	}
	domain := &fakeDomainEngine{}

	svc := NewService(Deps{
		Auth:     authz.PolicyDeps{Participant: participantStore},
		Campaign: campaignStore,
		Scene:    sceneStore,
		Write: domainwriteexec.WritePath{
			Executor: domain,
		},
	})

	_, _ = svc.TransitionScene(gametest.ContextWithParticipantID("manager-1"), &statev1.TransitionSceneRequest{
		CampaignId:    "c1",
		SourceSceneId: "sc-1",
		Name:          "Room B",
	})

	if domain.lastCommand.Type != handler.CommandTypeSceneTransition {
		t.Fatalf("command type = %q, want %q", domain.lastCommand.Type, handler.CommandTypeSceneTransition)
	}
	if domain.lastCommand.SessionID != "sess-1" {
		t.Fatalf("command session id = %q, want %q", domain.lastCommand.SessionID, "sess-1")
	}
}

func TestOpenSceneGate_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.OpenSceneGate(context.Background(), &statev1.OpenSceneGateRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestOpenSceneGate_MissingGateType(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.OpenSceneGate(context.Background(), &statev1.OpenSceneGateRequest{
		CampaignId: "c1", SceneId: "sc-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveSceneGate_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ResolveSceneGate(context.Background(), &statev1.ResolveSceneGateRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveSceneGate_MissingGateId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ResolveSceneGate(context.Background(), &statev1.ResolveSceneGateRequest{
		CampaignId: "c1", SceneId: "sc-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSceneGate_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AbandonSceneGate(context.Background(), &statev1.AbandonSceneGateRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSceneGate_MissingGateId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AbandonSceneGate(context.Background(), &statev1.AbandonSceneGateRequest{
		CampaignId: "c1", SceneId: "sc-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSceneSpotlight_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.SetSceneSpotlight(context.Background(), &statev1.SetSceneSpotlightRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSceneSpotlight_InvalidType(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.SetSceneSpotlight(context.Background(), &statev1.SetSceneSpotlightRequest{
		CampaignId: "c1", SceneId: "sc-1", Type: "invalid",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClearSceneSpotlight_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ClearSceneSpotlight(context.Background(), &statev1.ClearSceneSpotlightRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ListScenes with results ---

func TestListScenes_ReturnsScenes(t *testing.T) {
	campaignStore := activeCampaignStore("c1")
	participantStore := sessionManagerParticipantStore("c1")
	sceneStore := &fakeSceneStoreForService{
		scenes: map[string]storage.SceneRecord{
			"c1:sc-1": {
				CampaignID: "c1",
				SceneID:    "sc-1",
				SessionID:  "s-1",
				Name:       "Battle",
				Active:     true,
				CreatedAt:  time.Unix(1000, 0),
				UpdatedAt:  time.Unix(1000, 0),
			},
			"c1:sc-2": {
				CampaignID: "c1",
				SceneID:    "sc-2",
				SessionID:  "s-1",
				Name:       "Tavern",
				Active:     true,
				CreatedAt:  time.Unix(2000, 0),
				UpdatedAt:  time.Unix(2000, 0),
			},
		},
	}

	svc := NewService(Deps{
		Auth:     authz.PolicyDeps{Participant: participantStore},
		Campaign: campaignStore,
		Scene:    sceneStore,
	})
	resp, err := svc.ListScenes(gametest.ContextWithParticipantID("manager-1"), &statev1.ListScenesRequest{
		CampaignId: "c1",
		SessionId:  "s-1",
	})
	if err != nil {
		t.Fatalf("list scenes: %v", err)
	}
	if len(resp.GetScenes()) != 2 {
		t.Fatalf("scene count = %d, want 2", len(resp.GetScenes()))
	}
	// Scenes should have been converted to proto.
	for _, sc := range resp.GetScenes() {
		if sc.GetName() == "" {
			t.Error("expected scene name to be set")
		}
	}
}

// --- SceneToProto ---

func TestSceneToProto_WithCharacters(t *testing.T) {
	now := time.Now()
	ended := now.Add(time.Hour)
	rec := storage.SceneRecord{
		SceneID:     "sc-1",
		SessionID:   "s-1",
		Name:        "Battle",
		Description: "A fierce battle",
		Active:      false,
		CreatedAt:   now,
		UpdatedAt:   now,
		EndedAt:     &ended,
	}
	chars := []storage.SceneCharacterRecord{
		{CharacterID: "char-1"},
		{CharacterID: "char-2"},
	}
	pb := SceneToProto(rec, chars)
	if pb.GetName() != "Battle" {
		t.Errorf("name = %q, want %q", pb.GetName(), "Battle")
	}
	if len(pb.GetCharacterIds()) != 2 {
		t.Errorf("character count = %d, want 2", len(pb.GetCharacterIds()))
	}
	if pb.GetEndedAt() == nil {
		t.Error("expected ended_at")
	}
}

func TestSceneToProto_NoCharacters(t *testing.T) {
	rec := storage.SceneRecord{
		SceneID:   "sc-1",
		SessionID: "s-1",
		Name:      "Tavern",
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	pb := SceneToProto(rec, nil)
	if len(pb.GetCharacterIds()) != 0 {
		t.Errorf("character_ids = %v, want empty", pb.GetCharacterIds())
	}
	if pb.GetEndedAt() != nil {
		t.Error("expected nil ended_at")
	}
}

// --- Minimal fake scene stores for service tests ---

type fakeSceneStoreForService struct {
	storage.SceneStore
	scenes map[string]storage.SceneRecord
}

func (s *fakeSceneStoreForService) GetScene(_ context.Context, campaignID, sceneID string) (storage.SceneRecord, error) {
	key := campaignID + ":" + sceneID
	rec, ok := s.scenes[key]
	if !ok {
		return storage.SceneRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

func (s *fakeSceneStoreForService) ListScenes(_ context.Context, campaignID, sessionID string, pageSize int, pageToken string) (storage.ScenePage, error) {
	var scenes []storage.SceneRecord
	for _, rec := range s.scenes {
		if rec.CampaignID == campaignID && rec.SessionID == sessionID {
			scenes = append(scenes, rec)
		}
	}
	return storage.ScenePage{Scenes: scenes}, nil
}

type fakeSceneCharStoreForService struct {
	storage.SceneCharacterStore
}

func (s *fakeSceneCharStoreForService) ListSceneCharacters(_ context.Context, _, _ string) ([]storage.SceneCharacterRecord, error) {
	return nil, nil
}
