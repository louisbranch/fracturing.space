//go:build integration

package integration

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func runMutationEventGuardrailTests(t *testing.T, suite *integrationSuite, grpcAddr string) {
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

		_, err = suite.character.UpdateCharacter(ctxWithUser, &statev1.UpdateCharacterRequest{
			CampaignId:         campaignID,
			CharacterId:        characterID,
			OwnerParticipantId: wrapperspb.String(ownerPID),
		})
		if err != nil {
			t.Fatalf("set character owner: %v", err)
		}
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "character.updated")

		ensureDaggerheartCreationReadiness(t, ctxWithUser, suite.character, campaignID, characterID)
		lastSeq = requireEventTypesAfterSeq(t, ctxWithUser, eventClient, campaignID, lastSeq, "sys.daggerheart.character_profile_replaced")

		_ = ensureSessionStartReadiness(t, ctxWithUser, suite.participant, suite.character, campaignID, ownerPID, characterID)

		sessionResp := startSessionWithDefaultControllers(t, ctxWithUser, suite.session, suite.character, campaignID, "Guardrail Session")
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

	// Invite lifecycle events (invite.created, invite.claimed) are now managed
	// by the standalone invite service and no longer appear in the game event
	// journal. The game service only emits participant.bound via BindParticipant.

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
