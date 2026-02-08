//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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

	campaignClient := statev1.NewCampaignServiceClient(conn)
	sessionClient := statev1.NewSessionServiceClient(conn)
	participantClient := statev1.NewParticipantServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:   "Lock Test",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createResp == nil || createResp.Campaign == nil || createResp.Campaign.Id == "" {
		t.Fatal("expected campaign response")
	}
	participantsResp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: createResp.Campaign.Id,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participantsResp.Participants) == 0 {
		t.Fatal("expected owner participant")
	}
	ownerID := participantsResp.Participants[0].Id
	participantCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(grpcmeta.ParticipantIDHeader, ownerID))
	startResp, err := sessionClient.StartSession(ctx, &statev1.StartSessionRequest{
		CampaignId: createResp.Campaign.Id,
		Name:       "Session 1",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startResp == nil || startResp.Session == nil || startResp.Session.Id == "" {
		t.Fatal("expected session response")
	}
	_, err = participantClient.CreateParticipant(participantCtx, &statev1.CreateParticipantRequest{
		CampaignId:  createResp.Campaign.Id,
		DisplayName: "Player One",
		Role:        statev1.ParticipantRole_PLAYER,
		Controller:  statev1.Controller_CONTROLLER_HUMAN,
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

	endResp, err := sessionClient.EndSession(ctx, &statev1.EndSessionRequest{
		CampaignId: createResp.Campaign.Id,
		SessionId:  startResp.Session.Id,
	})
	if err != nil {
		t.Fatalf("end session: %v", err)
	}
	if endResp == nil || endResp.Session == nil {
		t.Fatal("expected end session response")
	}
	if endResp.Session.Status != statev1.SessionStatus_SESSION_ENDED {
		t.Fatalf("expected ended status, got %v", endResp.Session.Status)
	}
	if endResp.Session.EndedAt == nil {
		t.Fatal("expected ended_at to be set")
	}
	createParticipantResp, err := participantClient.CreateParticipant(participantCtx, &statev1.CreateParticipantRequest{
		CampaignId:  createResp.Campaign.Id,
		DisplayName: "Player One",
		Role:        statev1.ParticipantRole_PLAYER,
		Controller:  statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create participant after end session: %v", err)
	}
	if createParticipantResp == nil || createParticipantResp.Participant == nil || createParticipantResp.Participant.Id == "" {
		t.Fatal("expected participant response after end session")
	}
	_, err = participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: createResp.Campaign.Id,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
}
