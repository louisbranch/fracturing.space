//go:build integration
// +build integration

package integration

import (
	"context"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestSessionGateBlocksDaggerheartActions(t *testing.T) {
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
	sessionClient := gamev1.NewSessionServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	userID := createAuthUser(t, authAddr, "gate-gm")
	ctxWithUser := withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:               "Gate Actions Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "gate",
		CreatorDisplayName: "Gate GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	characterID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Gate Hero")
	patchDaggerheartProfile(t, ctxWithUser, characterClient, campaignID, characterID)

	startSession, err := sessionClient.StartSession(ctxWithUser, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Gate Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()
	sessionCtx := withSessionID(ctx, sessionID)

	gateID := "gate-block-1"
	openResp, err := sessionClient.OpenSessionGate(ctxWithUser, &gamev1.OpenSessionGateRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		GateType:   "spotlight",
		Reason:     "vote",
		GateId:     gateID,
	})
	if err != nil {
		t.Fatalf("open session gate: %v", err)
	}
	if openResp.GetGate() == nil {
		t.Fatal("expected gate in response")
	}

	_, err = daggerheartClient.ApplyConditions(sessionCtx, &daggerheartv1.DaggerheartApplyConditionsRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		Add:         []daggerheartv1.DaggerheartCondition{daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
		Source:      "test",
	})
	if err == nil {
		t.Fatal("expected apply conditions to fail with gate open")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition, got %s", st.Code())
	}
	if !strings.Contains(st.Message(), gateID) {
		t.Fatalf("expected gate id in error, got %q", st.Message())
	}

	_, err = sessionClient.ResolveSessionGate(ctxWithUser, &gamev1.ResolveSessionGateRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		GateId:     gateID,
		Decision:   "allow",
	})
	if err != nil {
		t.Fatalf("resolve session gate: %v", err)
	}

	applyResp, err := daggerheartClient.ApplyConditions(sessionCtx, &daggerheartv1.DaggerheartApplyConditionsRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		Add:         []daggerheartv1.DaggerheartCondition{daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
		Source:      "test",
	})
	if err != nil {
		t.Fatalf("apply conditions after resolve: %v", err)
	}
	if applyResp.GetState() == nil {
		t.Fatal("expected state after resolve")
	}
}
