package scenetransport

import (
	"context"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
)

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

func TestAddCharacterToScene_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AddCharacterToScene(context.Background(), &statev1.AddCharacterToSceneRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRemoveCharacterToScene_MissingCampaignId(t *testing.T) {
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
