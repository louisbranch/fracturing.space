package gametools

import (
	"context"
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
)

func TestSceneEndUsesResolvedCampaignAndShapesResult(t *testing.T) {
	t.Parallel()

	client := &sceneClientStub{}
	session := NewDirectSession(Clients{Scene: client}, SessionContext{
		CampaignID:    "camp-1",
		ParticipantID: "part-1",
	})

	result, err := session.sceneEnd(context.Background(), []byte(`{"scene_id":"scene-1","reason":"wrap up"}`))
	if err != nil {
		t.Fatalf("sceneEnd() error = %v", err)
	}

	payload := decodeToolOutput[sceneEndResult](t, result.Output)
	if payload.SceneID != "scene-1" || !payload.Ended {
		t.Fatalf("scene end payload = %#v", payload)
	}
	if client.lastEndRequest.GetCampaignId() != "camp-1" || client.lastEndRequest.GetSceneId() != "scene-1" || client.lastEndRequest.GetReason() != "wrap up" {
		t.Fatalf("end request = %#v", client.lastEndRequest)
	}
	if got := client.lastMetadata.Get(grpcmeta.ParticipantIDHeader); len(got) != 1 || got[0] != "part-1" {
		t.Fatalf("participant metadata = %#v", got)
	}
}

func TestSceneAddAndRemoveCharacterShapeRequestsAndErrors(t *testing.T) {
	t.Parallel()

	t.Run("add character success", func(t *testing.T) {
		t.Parallel()

		client := &sceneClientStub{}
		session := NewDirectSession(Clients{Scene: client}, SessionContext{
			CampaignID:    "camp-1",
			ParticipantID: "part-1",
		})

		result, err := session.sceneAddCharacter(context.Background(), []byte(`{"scene_id":"scene-1","character_id":"char-1"}`))
		if err != nil {
			t.Fatalf("sceneAddCharacter() error = %v", err)
		}

		payload := decodeToolOutput[sceneAddCharacterResult](t, result.Output)
		if payload.SceneID != "scene-1" || payload.CharacterID != "char-1" || !payload.Added {
			t.Fatalf("scene add payload = %#v", payload)
		}
		if client.lastAddRequest.GetCampaignId() != "camp-1" || client.lastAddRequest.GetSceneId() != "scene-1" || client.lastAddRequest.GetCharacterId() != "char-1" {
			t.Fatalf("add request = %#v", client.lastAddRequest)
		}
		if got := client.lastMetadata.Get(grpcmeta.ParticipantIDHeader); len(got) != 1 || got[0] != "part-1" {
			t.Fatalf("participant metadata = %#v", got)
		}
	})

	t.Run("add character client errors are wrapped", func(t *testing.T) {
		t.Parallel()

		session := NewDirectSession(Clients{Scene: &sceneClientStub{addErr: errors.New("boom")}}, SessionContext{CampaignID: "camp-1"})
		_, err := session.sceneAddCharacter(context.Background(), []byte(`{"scene_id":"scene-1","character_id":"char-1"}`))
		if err == nil || err.Error() != "add character to scene failed: boom" {
			t.Fatalf("sceneAddCharacter() error = %v", err)
		}
	})

	t.Run("remove character success", func(t *testing.T) {
		t.Parallel()

		client := &sceneClientStub{}
		session := NewDirectSession(Clients{Scene: client}, SessionContext{CampaignID: "camp-1"})

		result, err := session.sceneRemoveCharacter(context.Background(), []byte(`{"scene_id":"scene-1","character_id":"char-1"}`))
		if err != nil {
			t.Fatalf("sceneRemoveCharacter() error = %v", err)
		}

		payload := decodeToolOutput[sceneRemoveCharacterResult](t, result.Output)
		if payload.SceneID != "scene-1" || payload.CharacterID != "char-1" || !payload.Removed {
			t.Fatalf("scene remove payload = %#v", payload)
		}
		if client.lastRemoveRequest.GetCampaignId() != "camp-1" || client.lastRemoveRequest.GetSceneId() != "scene-1" || client.lastRemoveRequest.GetCharacterId() != "char-1" {
			t.Fatalf("remove request = %#v", client.lastRemoveRequest)
		}
	})

	t.Run("remove character client errors are wrapped", func(t *testing.T) {
		t.Parallel()

		session := NewDirectSession(Clients{Scene: &sceneClientStub{removeErr: errors.New("boom")}}, SessionContext{CampaignID: "camp-1"})
		_, err := session.sceneRemoveCharacter(context.Background(), []byte(`{"scene_id":"scene-1","character_id":"char-1"}`))
		if err == nil || err.Error() != "remove character from scene failed: boom" {
			t.Fatalf("sceneRemoveCharacter() error = %v", err)
		}
	})
}

func TestSceneEndClientErrorsAreWrapped(t *testing.T) {
	t.Parallel()

	session := NewDirectSession(Clients{Scene: &sceneClientStub{endErr: errors.New("boom")}}, SessionContext{CampaignID: "camp-1"})
	_, err := session.sceneEnd(context.Background(), []byte(`{"scene_id":"scene-1"}`))
	if err == nil || err.Error() != "end scene failed: boom" {
		t.Fatalf("sceneEnd() error = %v", err)
	}
}
