//go:build integration

package integration

import (
	"context"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func runMutationEventGuardrailTests(t *testing.T, suite *integrationSuite, grpcAddr string, authAddr string) {
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

	eventClient := statev1.NewEventServiceClient(conn)
	inviteClient := statev1.NewInviteServiceClient(conn)

	authConn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial auth gRPC: %v", err)
	}
	defer authConn.Close()

	authClient := authv1.NewAuthServiceClient(authConn)

	t.Run("campaign mutations emit events", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctxWithUser := suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctxWithUser, &statev1.CreateCampaignRequest{
			Name:   "Guardrail Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()
		ownerPID := campaignResp.GetOwnerParticipant().GetId()
		if campaignID == "" {
			t.Fatal("campaign id is empty")
		}
		if ownerPID == "" {
			t.Fatal("owner participant id is empty")
		}

		lastSeq := requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, 0, "campaign.created")

		_, err = suite.participant.CreateParticipant(ctxWithUser, &statev1.CreateParticipantRequest{
			CampaignId: campaignID,
			Name:       "Guardrail Player",
			Role:       statev1.ParticipantRole_PLAYER,
			Controller: statev1.Controller_CONTROLLER_HUMAN,
		})
		if err != nil {
			t.Fatalf("create participant: %v", err)
		}
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "participant.joined")

		characterResp, err := suite.character.CreateCharacter(ctxWithUser, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       "Guardrail Hero",
			Kind:       statev1.CharacterKind_PC,
		})
		if err != nil {
			t.Fatalf("create character: %v", err)
		}
		characterID := characterResp.GetCharacter().GetId()
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "character.created")

		_, err = suite.character.SetDefaultControl(ctxWithUser, &statev1.SetDefaultControlRequest{
			CampaignId:    campaignID,
			CharacterId:   characterID,
			ParticipantId: wrapperspb.String(ownerPID),
		})
		if err != nil {
			t.Fatalf("set character control: %v", err)
		}
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "character.updated")

		ensureDaggerheartCreationReadiness(t, ctxWithUser, suite.character, campaignID, characterID)
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "sys.daggerheart.character_profile_replaced")

		_ = ensureSessionStartReadiness(t, ctxWithUser, suite.participant, suite.character, campaignID, ownerPID, characterID)

		sessionResp, err := suite.session.StartSession(ctxWithUser, &statev1.StartSessionRequest{
			CampaignId: campaignID,
			Name:       "Guardrail Session",
		})
		if err != nil {
			t.Fatalf("start session: %v", err)
		}
		sessionID := sessionResp.GetSession().GetId()
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "campaign.updated", "session.started")

		_, err = suite.session.EndSession(ctxWithUser, &statev1.EndSessionRequest{
			CampaignId: campaignID,
			SessionId:  sessionID,
		})
		if err != nil {
			t.Fatalf("end session: %v", err)
		}
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "session.ended")

		_, err = suite.campaign.EndCampaign(ctxWithUser, &statev1.EndCampaignRequest{CampaignId: campaignID})
		if err != nil {
			t.Fatalf("end campaign: %v", err)
		}
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "campaign.updated")

		_, err = suite.campaign.ArchiveCampaign(ctxWithUser, &statev1.ArchiveCampaignRequest{CampaignId: campaignID})
		if err != nil {
			t.Fatalf("archive campaign: %v", err)
		}
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "campaign.updated")

		_, err = suite.campaign.RestoreCampaign(ctxWithUser, &statev1.RestoreCampaignRequest{CampaignId: campaignID})
		if err != nil {
			t.Fatalf("restore campaign: %v", err)
		}
		_ = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "campaign.updated")
	})

	t.Run("invite claim emits events", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctxWithUser := suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctxWithUser, &statev1.CreateCampaignRequest{
			Name:   "Invite Guardrail Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()
		ownerPID := campaignResp.GetOwnerParticipant().GetId()

		lastSeq := requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, 0, "campaign.created")

		participantResp, err := suite.participant.CreateParticipant(ctxWithUser, &statev1.CreateParticipantRequest{
			CampaignId: campaignID,
			Name:       "Invite Guardrail Player",
			Role:       statev1.ParticipantRole_PLAYER,
			Controller: statev1.Controller_CONTROLLER_HUMAN,
		})
		if err != nil {
			t.Fatalf("create participant: %v", err)
		}
		participantID := participantResp.GetParticipant().GetId()
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "participant.joined")

		ownerCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(grpcmeta.ParticipantIDHeader, ownerPID))
		inviteResp, err := inviteClient.CreateInvite(ownerCtx, &statev1.CreateInviteRequest{
			CampaignId:    campaignID,
			ParticipantId: participantID,
		})
		if err != nil {
			t.Fatalf("create invite: %v", err)
		}
		if inviteResp == nil || inviteResp.Invite == nil {
			t.Fatal("create invite returned nil invite")
		}
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "invite.created")

		claimerID := createAuthUser(t, authAddr, "invite-claimer")

		grantResp, err := authClient.IssueJoinGrant(ctx, &authv1.IssueJoinGrantRequest{
			UserId:        claimerID,
			CampaignId:    campaignID,
			InviteId:      inviteResp.Invite.Id,
			ParticipantId: participantID,
		})
		if err != nil {
			t.Fatalf("issue join grant: %v", err)
		}
		if grantResp == nil || grantResp.JoinGrant == "" {
			t.Fatal("issue join grant returned empty grant")
		}

		claimCtx := withUserID(ctx, claimerID)
		_, err = inviteClient.ClaimInvite(claimCtx, &statev1.ClaimInviteRequest{
			CampaignId: campaignID,
			InviteId:   inviteResp.Invite.Id,
			JoinGrant:  grantResp.JoinGrant,
		})
		if err != nil {
			t.Fatalf("claim invite: %v", err)
		}
		requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "participant.bound", "invite.claimed")
	})

	t.Run("campaign fork emits event", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctxWithUser := suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctxWithUser, &statev1.CreateCampaignRequest{
			Name:   "Fork Guardrail Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()

		forkResp, err := suite.fork.ForkCampaign(ctxWithUser, &statev1.ForkCampaignRequest{
			SourceCampaignId: campaignID,
			NewCampaignName:  "Guardrail Fork",
			CopyParticipants: true,
		})
		if err != nil {
			t.Fatalf("fork campaign: %v", err)
		}
		forkedID := forkResp.GetCampaign().GetId()

		requireEventTypesAfterSeq(t, ctxWithUser, eventClient, forkedID, 0, "campaign.created", "campaign.forked")
	})
}
