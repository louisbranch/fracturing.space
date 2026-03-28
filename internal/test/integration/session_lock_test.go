//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func runSessionLockTests(t *testing.T, grpcAddr string, authAddr string) {
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
	characterClient := statev1.NewCharacterServiceClient(conn)
	snapshotClient := statev1.NewSnapshotServiceClient(conn)
	sessionClient := statev1.NewSessionServiceClient(conn)
	participantClient := statev1.NewParticipantServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	userID := createAuthUser(t, authAddr, "session-lock-creator")
	ctxWithUser := withUserID(ctx, userID)

	createResp, err := campaignClient.CreateCampaign(ctxWithUser, &statev1.CreateCampaignRequest{
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
	participantsResp, err := participantClient.ListParticipants(ctxWithUser, &statev1.ListParticipantsRequest{
		CampaignId: createResp.Campaign.Id,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participantsResp.Participants) == 0 {
		t.Fatal("expected owner participant")
	}
	ownerID := participantsResp.Participants[0].Id
	ensureSessionStartReadiness(t, ctxWithUser, participantClient, characterClient, createResp.Campaign.Id, ownerID)
	charactersResp, err := characterClient.ListCharacters(ctxWithUser, &statev1.ListCharactersRequest{
		CampaignId: createResp.Campaign.Id,
		PageSize:   100,
	})
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if len(charactersResp.GetCharacters()) == 0 {
		t.Fatal("expected at least one character")
	}
	characterID := strings.TrimSpace(charactersResp.GetCharacters()[0].GetId())
	if characterID == "" {
		t.Fatal("expected non-empty character id")
	}
	participantCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(grpcmeta.ParticipantIDHeader, ownerID))
	startResp := startSessionWithDefaultControllers(t, participantCtx, sessionClient, characterClient, createResp.Campaign.Id, "Session 1")
	if startResp == nil || startResp.Session == nil || startResp.Session.Id == "" {
		t.Fatal("expected session response")
	}

	_, err = campaignClient.UpdateCampaign(participantCtx, &statev1.UpdateCampaignRequest{
		CampaignId: createResp.Campaign.Id,
		Name:       wrapperspb.String("Locked Campaign Rename"),
	})
	assertActiveSessionLockError(t, err, startResp.GetSession().GetId())

	_, err = participantClient.CreateParticipant(participantCtx, &statev1.CreateParticipantRequest{
		CampaignId: createResp.Campaign.Id,
		Name:       "Player One",
		Role:       statev1.ParticipantRole_PLAYER,
		Controller: statev1.Controller_CONTROLLER_HUMAN,
	})
	assertActiveSessionLockError(t, err, startResp.GetSession().GetId())

	// Invite creation is handled by the standalone invite service and is not
	// gated by the game session lock.

	_, err = characterClient.CreateCharacter(participantCtx, &statev1.CreateCharacterRequest{
		CampaignId: createResp.Campaign.Id,
		Name:       "Locked Character",
		Kind:       statev1.CharacterKind_PC,
	})
	assertActiveSessionLockError(t, err, startResp.GetSession().GetId())

	_, err = snapshotClient.PatchCharacterState(ctxWithUser, &statev1.PatchCharacterStateRequest{
		CampaignId:  createResp.Campaign.Id,
		CharacterId: characterID,
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{
				Hp:     1,
				Hope:   1,
				Stress: 1,
			},
		},
	})
	if err != nil {
		t.Fatalf("patch character state during active session should be allowed: %v", err)
	}

	endResp, err := sessionClient.EndSession(participantCtx, &statev1.EndSessionRequest{
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
		CampaignId: createResp.Campaign.Id,
		Name:       "Player One",
		Role:       statev1.ParticipantRole_PLAYER,
		Controller: statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create participant after end session: %v", err)
	}
	if createParticipantResp == nil || createParticipantResp.Participant == nil || createParticipantResp.Participant.Id == "" {
		t.Fatal("expected participant response after end session")
	}
	_, err = participantClient.ListParticipants(ctxWithUser, &statev1.ListParticipantsRequest{
		CampaignId: createResp.Campaign.Id,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
}

func assertActiveSessionLockError(t *testing.T, err error, sessionID string) {
	t.Helper()
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
	expectedSessionID := "active_session_id=" + sessionID
	if !strings.Contains(st.Message(), expectedSessionID) {
		t.Fatalf("expected active session id in message, got %q", st.Message())
	}
}
