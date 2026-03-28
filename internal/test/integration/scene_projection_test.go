//go:build integration

package integration

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// TestSceneProjectionLifecycle verifies the full scene projection lifecycle
// through gRPC: create, read, update, character management, list, and end.
func TestSceneProjectionLifecycle(t *testing.T) {
	grpcAddr, authAddr, stopServer := startGRPCServer(t)
	defer stopServer()

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	campaignClient := gamev1.NewCampaignServiceClient(conn)
	characterClient := gamev1.NewCharacterServiceClient(conn)
	participantClient := gamev1.NewParticipantServiceClient(conn)
	sessionClient := gamev1.NewSessionServiceClient(conn)
	sceneClient := gamev1.NewSceneServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	userID := createAuthUser(t, authAddr, "scene-projection-gm")
	ctxWithUser := withUserID(ctx, userID)

	// Create campaign.
	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:        "Scene Projection Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "scene-projection",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	campaignID := createCampaign.GetCampaign().GetId()
	ownerParticipantID := createCampaign.GetOwnerParticipant().GetId()

	// Create characters.
	char1ID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Scene Hero A")
	char2ID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Scene Hero B")

	ensureSessionStartReadiness(t, ctxWithUser, participantClient, characterClient, campaignID, ownerParticipantID, char1ID, char2ID)

	// Start session.
	startSession := startSessionWithDefaultControllers(t, ctxWithUser, sessionClient, characterClient, campaignID, "Scene Session")
	sessionID := startSession.GetSession().GetId()

	// --- Create scene with initial characters ---
	createResp, err := sceneClient.CreateScene(ctxWithUser, &gamev1.CreateSceneRequest{
		CampaignId:   campaignID,
		SessionId:    sessionID,
		Name:         "Tavern Entrance",
		Description:  "A dimly lit entrance.",
		CharacterIds: []string{char1ID},
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	sceneID := createResp.GetSceneId()
	if sceneID == "" {
		t.Fatal("expected non-empty scene id")
	}

	// --- GetScene: verify creation ---
	getResp, err := sceneClient.GetScene(ctxWithUser, &gamev1.GetSceneRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		t.Fatalf("get scene: %v", err)
	}
	scene := getResp.GetScene()
	if scene == nil {
		t.Fatal("expected scene in get response")
	}
	if scene.GetName() != "Tavern Entrance" {
		t.Errorf("scene name = %q, want %q", scene.GetName(), "Tavern Entrance")
	}
	if scene.GetDescription() != "A dimly lit entrance." {
		t.Errorf("scene description = %q, want %q", scene.GetDescription(), "A dimly lit entrance.")
	}
	if scene.GetSessionId() != sessionID {
		t.Errorf("scene session_id = %q, want %q", scene.GetSessionId(), sessionID)
	}
	if !scene.GetOpen() {
		t.Error("expected scene to be open")
	}
	if len(scene.GetCharacterIds()) != 1 || scene.GetCharacterIds()[0] != char1ID {
		t.Errorf("scene character_ids = %v, want [%s]", scene.GetCharacterIds(), char1ID)
	}

	// --- Update scene ---
	_, err = sceneClient.UpdateScene(ctxWithUser, &gamev1.UpdateSceneRequest{
		CampaignId:  campaignID,
		SceneId:     sceneID,
		Name:        "Tavern Interior",
		Description: "Warm firelight flickers.",
	})
	if err != nil {
		t.Fatalf("update scene: %v", err)
	}

	getResp, err = sceneClient.GetScene(ctxWithUser, &gamev1.GetSceneRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		t.Fatalf("get scene after update: %v", err)
	}
	if getResp.GetScene().GetName() != "Tavern Interior" {
		t.Errorf("updated name = %q, want %q", getResp.GetScene().GetName(), "Tavern Interior")
	}
	if getResp.GetScene().GetDescription() != "Warm firelight flickers." {
		t.Errorf("updated description = %q, want %q", getResp.GetScene().GetDescription(), "Warm firelight flickers.")
	}

	// --- Add character ---
	_, err = sceneClient.AddCharacterToScene(ctxWithUser, &gamev1.AddCharacterToSceneRequest{
		CampaignId:  campaignID,
		SceneId:     sceneID,
		CharacterId: char2ID,
	})
	if err != nil {
		t.Fatalf("add character to scene: %v", err)
	}

	getResp, err = sceneClient.GetScene(ctxWithUser, &gamev1.GetSceneRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		t.Fatalf("get scene after add character: %v", err)
	}
	if len(getResp.GetScene().GetCharacterIds()) != 2 {
		t.Errorf("character count = %d, want 2", len(getResp.GetScene().GetCharacterIds()))
	}

	// --- Remove character ---
	_, err = sceneClient.RemoveCharacterFromScene(ctxWithUser, &gamev1.RemoveCharacterFromSceneRequest{
		CampaignId:  campaignID,
		SceneId:     sceneID,
		CharacterId: char1ID,
	})
	if err != nil {
		t.Fatalf("remove character from scene: %v", err)
	}

	getResp, err = sceneClient.GetScene(ctxWithUser, &gamev1.GetSceneRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		t.Fatalf("get scene after remove character: %v", err)
	}
	charIDs := getResp.GetScene().GetCharacterIds()
	if len(charIDs) != 1 || charIDs[0] != char2ID {
		t.Errorf("character_ids after remove = %v, want [%s]", charIDs, char2ID)
	}

	// --- ListScenes ---
	listResp, err := sceneClient.ListScenes(ctxWithUser, &gamev1.ListScenesRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list scenes: %v", err)
	}
	if len(listResp.GetScenes()) != 1 {
		t.Errorf("list scenes count = %d, want 1", len(listResp.GetScenes()))
	}
	if listResp.GetScenes()[0].GetSceneId() != sceneID {
		t.Errorf("listed scene id = %q, want %q", listResp.GetScenes()[0].GetSceneId(), sceneID)
	}

	// --- End scene ---
	_, err = sceneClient.EndScene(ctxWithUser, &gamev1.EndSceneRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
		Reason:     "transition",
	})
	if err != nil {
		t.Fatalf("end scene: %v", err)
	}

	getResp, err = sceneClient.GetScene(ctxWithUser, &gamev1.GetSceneRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		t.Fatalf("get scene after end: %v", err)
	}
	if getResp.GetScene().GetOpen() {
		t.Error("expected scene to be closed after end")
	}
	if getResp.GetScene().GetEndedAt() == nil {
		t.Error("expected ended_at to be set")
	}

	// --- GetScene for non-existent scene ---
	_, err = sceneClient.GetScene(ctxWithUser, &gamev1.GetSceneRequest{
		CampaignId: campaignID,
		SceneId:    "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for non-existent scene")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %s", st.Code())
	}
}
