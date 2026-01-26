//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func runSessionLockTests(t *testing.T, grpcAddr string) {
	t.Helper()

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	campaignClient := campaignv1.NewCampaignServiceClient(conn)
	sessionClient := sessionv1.NewSessionServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createResp, err := campaignClient.CreateCampaign(ctx, &campaignv1.CreateCampaignRequest{
		Name:   "Lock Test",
		GmMode: campaignv1.GmMode_HUMAN,
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createResp == nil || createResp.Campaign == nil || createResp.Campaign.Id == "" {
		t.Fatal("expected campaign response")
	}

	startResp, err := sessionClient.StartSession(ctx, &sessionv1.StartSessionRequest{
		CampaignId: createResp.Campaign.Id,
		Name:       "Session 1",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startResp == nil || startResp.Session == nil || startResp.Session.Id == "" {
		t.Fatal("expected session response")
	}

	_, err = campaignClient.CreateParticipant(ctx, &campaignv1.CreateParticipantRequest{
		CampaignId:  createResp.Campaign.Id,
		DisplayName: "Player One",
		Role:        campaignv1.ParticipantRole_PLAYER,
		Controller:  campaignv1.Controller_CONTROLLER_HUMAN,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition, got %v", st.Code())
	}
	if !strings.Contains(st.Message(), "campaign has an active session") {
		t.Fatalf("expected active session message, got %q", st.Message())
	}
	expectedSessionID := "active_session_id=" + startResp.Session.Id
	if !strings.Contains(st.Message(), expectedSessionID) {
		t.Fatalf("expected active session id in message, got %q", st.Message())
	}

	_, err = campaignClient.ListParticipants(ctx, &campaignv1.ListParticipantsRequest{
		CampaignId: createResp.Campaign.Id,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
}
