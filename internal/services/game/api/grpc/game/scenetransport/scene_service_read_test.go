package scenetransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

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
				Open:       true,
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
	resp, err := svc.GetScene(requestctx.WithParticipantID("manager-1"), &statev1.GetSceneRequest{
		CampaignId: "c1",
		SceneId:    "sc-1",
	})
	if err != nil {
		t.Fatalf("get scene: %v", err)
	}
	if resp.GetScene().GetName() != "Battle" {
		t.Errorf("name = %q, want %q", resp.GetScene().GetName(), "Battle")
	}
	if !resp.GetScene().GetOpen() {
		t.Error("expected open")
	}
}

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
	resp, err := svc.ListScenes(requestctx.WithParticipantID("manager-1"), &statev1.ListScenesRequest{
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
				Open:       true,
				CreatedAt:  time.Unix(1000, 0),
				UpdatedAt:  time.Unix(1000, 0),
			},
			"c1:sc-2": {
				CampaignID: "c1",
				SceneID:    "sc-2",
				SessionID:  "s-1",
				Name:       "Tavern",
				Open:       true,
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
	resp, err := svc.ListScenes(requestctx.WithParticipantID("manager-1"), &statev1.ListScenesRequest{
		CampaignId: "c1",
		SessionId:  "s-1",
	})
	if err != nil {
		t.Fatalf("list scenes: %v", err)
	}
	if len(resp.GetScenes()) != 2 {
		t.Fatalf("scene count = %d, want 2", len(resp.GetScenes()))
	}
	for _, sc := range resp.GetScenes() {
		if sc.GetName() == "" {
			t.Error("expected scene name to be set")
		}
	}
}
