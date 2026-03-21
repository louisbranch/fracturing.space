package scenetransport

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
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
