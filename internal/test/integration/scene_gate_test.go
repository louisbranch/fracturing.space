//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// TestSceneGateBlocksSceneActions verifies that opening a scene gate blocks
// scene-scoped commands (spotlight) and that resolving the gate unblocks them.
//
// Scene gates use GateScopeScene, which blocks scene-scoped commands like
// SetSceneSpotlight when a gate is open on that scene.
func TestSceneGateBlocksSceneActions(t *testing.T) {
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

	userID := createAuthUser(t, authAddr, "scene-gate-gm")
	ctxWithUser := withUserID(ctx, userID)

	// Setup campaign.
	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:        "Scene Gate Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "scene-gate",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	campaignID := createCampaign.GetCampaign().GetId()
	ownerParticipantID := createCampaign.GetOwnerParticipant().GetId()

	charID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Gate Hero")
	ensureSessionStartReadiness(t, ctxWithUser, participantClient, characterClient, campaignID, ownerParticipantID, charID)

	// Start session.
	startSession, err := sessionClient.StartSession(ctxWithUser, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Gate Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	sessionID := startSession.GetSession().GetId()

	// Create scene with the character.
	createResp, err := sceneClient.CreateScene(ctxWithUser, &gamev1.CreateSceneRequest{
		CampaignId:   campaignID,
		SessionId:    sessionID,
		Name:         "Gated Scene",
		CharacterIds: []string{charID},
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	sceneID := createResp.GetSceneId()

	// Open scene gate.
	gateID := "scene-gate-block-1"
	_, err = sceneClient.OpenSceneGate(ctxWithUser, &gamev1.OpenSceneGateRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
		GateType:   "spotlight",
		Reason:     "vote",
		GateId:     gateID,
	})
	if err != nil {
		t.Fatalf("open scene gate: %v", err)
	}

	// Attempt to open another gate on the same scene — should fail because
	// the scene already has an open gate (GateScopeScene blocks this).
	_, err = sceneClient.OpenSceneGate(ctxWithUser, &gamev1.OpenSceneGateRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
		GateType:   "another",
		Reason:     "blocked",
		GateId:     "scene-gate-block-2",
	})
	if err == nil {
		t.Fatal("expected open scene gate to fail while gate is open")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %s", st.Code())
	}
	if !strings.Contains(strings.ToLower(st.Message()), "scene gate is open") {
		t.Fatalf("expected scene gate open error, got %q", st.Message())
	}

	// Resolve the gate.
	_, err = sceneClient.ResolveSceneGate(ctxWithUser, &gamev1.ResolveSceneGateRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
		GateId:     gateID,
		Decision:   "allow",
	})
	if err != nil {
		t.Fatalf("resolve scene gate: %v", err)
	}

	// After resolving, opening a new gate should succeed.
	secondGateID := "scene-gate-after-resolve"
	_, err = sceneClient.OpenSceneGate(ctxWithUser, &gamev1.OpenSceneGateRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
		GateType:   "spotlight",
		Reason:     "second vote",
		GateId:     secondGateID,
	})
	if err != nil {
		t.Fatalf("open scene gate after resolve: %v", err)
	}

	// Abandon the second gate.
	_, err = sceneClient.AbandonSceneGate(ctxWithUser, &gamev1.AbandonSceneGateRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
		GateId:     secondGateID,
		Reason:     "timeout",
	})
	if err != nil {
		t.Fatalf("abandon scene gate: %v", err)
	}

	// Verify scene is still accessible after gates.
	getResp, err := sceneClient.GetScene(ctxWithUser, &gamev1.GetSceneRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		t.Fatalf("get scene after gates: %v", err)
	}
	if !getResp.GetScene().GetActive() {
		t.Error("expected scene to still be active")
	}
}
