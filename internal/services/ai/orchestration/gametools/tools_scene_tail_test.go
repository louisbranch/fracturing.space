package gametools

import (
	"context"
	"errors"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
)

func TestSceneUpdateShapesRequestAndWrapsErrors(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		client := &sceneClientStub{}
		session := NewDirectSession(Clients{Scene: client}, SessionContext{
			CampaignID:    "camp-1",
			ParticipantID: "part-1",
		})

		result, err := session.sceneUpdate(context.Background(), []byte(`{"scene_id":"scene-1","name":"Storm Gate","description":"The gate groans open"}`))
		if err != nil {
			t.Fatalf("sceneUpdate() error = %v", err)
		}

		payload := decodeToolOutput[sceneUpdateResult](t, result.Output)
		if payload.SceneID != "scene-1" || !payload.Updated {
			t.Fatalf("scene update payload = %#v", payload)
		}
		if client.lastUpdateRequest.GetCampaignId() != "camp-1" || client.lastUpdateRequest.GetSceneId() != "scene-1" || client.lastUpdateRequest.GetName() != "Storm Gate" || client.lastUpdateRequest.GetDescription() != "The gate groans open" {
			t.Fatalf("update request = %#v", client.lastUpdateRequest)
		}
		if got := client.lastMetadata.Get(grpcmeta.ParticipantIDHeader); len(got) != 1 || got[0] != "part-1" {
			t.Fatalf("participant metadata = %#v", got)
		}
	})

	t.Run("wrapped error", func(t *testing.T) {
		t.Parallel()

		session := NewDirectSession(Clients{Scene: &sceneClientStub{updateErr: errors.New("boom")}}, SessionContext{CampaignID: "camp-1"})
		_, err := session.sceneUpdate(context.Background(), []byte(`{"scene_id":"scene-1"}`))
		if err == nil || err.Error() != "update scene failed: boom" {
			t.Fatalf("sceneUpdate() error = %v", err)
		}
	})
}

func TestDaggerheartRuntimeAdapterAccessorsExposeSessionClients(t *testing.T) {
	t.Parallel()

	characters := &characterClientStub{}
	sessions := &sessionClientStub{}
	snapshots := &snapshotClientStub{}
	daggerheart := &daggerheartClientStub{}
	session := NewDirectSession(Clients{
		Interaction: interactionClientStub{
			response: &statev1.GetInteractionStateResponse{
				State: &statev1.InteractionState{
					ActiveScene: &statev1.InteractionScene{SceneId: "scene-from-state"},
				},
			},
		},
		Character:   characters,
		Session:     sessions,
		Snapshot:    snapshots,
		Daggerheart: daggerheart,
	}, SessionContext{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})

	adapter := daggerheartRuntimeAdapter{session: session}
	if adapter.CharacterClient() != characters {
		t.Fatalf("CharacterClient() = %#v, want %#v", adapter.CharacterClient(), characters)
	}
	if adapter.SessionClient() != sessions {
		t.Fatalf("SessionClient() = %#v, want %#v", adapter.SessionClient(), sessions)
	}
	if adapter.SnapshotClient() != snapshots {
		t.Fatalf("SnapshotClient() = %#v, want %#v", adapter.SnapshotClient(), snapshots)
	}
	if adapter.DaggerheartClient() != daggerheart {
		t.Fatalf("DaggerheartClient() = %#v, want %#v", adapter.DaggerheartClient(), daggerheart)
	}

	sceneID, err := adapter.ResolveSceneID(context.Background(), "camp-1", "scene-explicit")
	if err != nil {
		t.Fatalf("ResolveSceneID() error = %v", err)
	}
	if sceneID != "scene-explicit" {
		t.Fatalf("ResolveSceneID() = %q, want scene-explicit", sceneID)
	}
}

type snapshotClientStub struct {
	statev1.SnapshotServiceClient
}

type daggerheartClientStub struct {
	pb.DaggerheartServiceClient
}
