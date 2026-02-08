package integration

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func runParticipantUserLinkTests(t *testing.T, grpcAddr string) {
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
	participantClient := statev1.NewParticipantServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:   "User Link Test",
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

	firstResp, err := participantClient.CreateParticipant(participantCtx, &statev1.CreateParticipantRequest{
		CampaignId:  createResp.Campaign.Id,
		UserId:      "user-1",
		DisplayName: "Player One",
		Role:        statev1.ParticipantRole_PLAYER,
		Controller:  statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}
	if firstResp == nil || firstResp.Participant == nil || firstResp.Participant.Id == "" {
		t.Fatal("expected participant response")
	}

	secondResp, err := participantClient.CreateParticipant(participantCtx, &statev1.CreateParticipantRequest{
		CampaignId:  createResp.Campaign.Id,
		UserId:      "user-2",
		DisplayName: "Player Two",
		Role:        statev1.ParticipantRole_PLAYER,
		Controller:  statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create second participant: %v", err)
	}
	if secondResp == nil || secondResp.Participant == nil || secondResp.Participant.Id == "" {
		t.Fatal("expected second participant response")
	}

	_, err = participantClient.UpdateParticipant(participantCtx, &statev1.UpdateParticipantRequest{
		CampaignId:    createResp.Campaign.Id,
		ParticipantId: secondResp.Participant.Id,
		UserId:        wrapperspb.String("user-1"),
	})
	if err == nil {
		t.Fatal("expected duplicate user_id error")
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if statusErr.Code() != codes.AlreadyExists {
		t.Fatalf("expected already exists, got %v: %s", statusErr.Code(), statusErr.Message())
	}
}
