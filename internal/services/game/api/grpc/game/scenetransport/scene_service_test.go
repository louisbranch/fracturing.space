package scenetransport

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

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
