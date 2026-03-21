//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// runCampaignToolsTests exercises campaign-related gRPC operations.
func runCampaignToolsTests(t *testing.T, suite *integrationSuite) {
	t.Helper()

	t.Run("participant create", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaign := campaignResp.GetCampaign()
		if campaign.GetId() == "" {
			t.Fatal("campaign id is empty")
		}

		participantResp, err := suite.participant.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
			CampaignId: campaign.GetId(),
			Name:       "Test Player",
			Role:       statev1.ParticipantRole_PLAYER,
			Controller: statev1.Controller_CONTROLLER_HUMAN,
		})
		if err != nil {
			t.Fatalf("create participant: %v", err)
		}
		p := participantResp.GetParticipant()
		if p.GetId() == "" {
			t.Fatal("participant id is empty")
		}
		if p.GetCreatedAt() == nil {
			t.Fatal("participant created_at is nil")
		}
		if p.GetUpdatedAt() == nil {
			t.Fatal("participant updated_at is nil")
		}
		if p.GetUpdatedAt().AsTime().Before(p.GetCreatedAt().AsTime()) {
			t.Fatalf("expected updated_at after created_at: %v < %v", p.GetUpdatedAt().AsTime(), p.GetCreatedAt().AsTime())
		}
	})

	t.Run("character create", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()

		pcResp, err := suite.character.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       "Test PC",
			Kind:       statev1.CharacterKind_PC,
			Notes:      "A brave warrior",
		})
		if err != nil {
			t.Fatalf("create PC character: %v", err)
		}
		pc := pcResp.GetCharacter()
		if pc.GetId() == "" {
			t.Fatal("character id is empty")
		}
		if pc.GetNotes() != "A brave warrior" {
			t.Fatalf("expected notes 'A brave warrior', got %q", pc.GetNotes())
		}
		if pc.GetCreatedAt() == nil {
			t.Fatal("character created_at is nil")
		}
		if pc.GetUpdatedAt() == nil {
			t.Fatal("character updated_at is nil")
		}
		if pc.GetUpdatedAt().AsTime().Before(pc.GetCreatedAt().AsTime()) {
			t.Fatalf("expected updated_at after created_at: %v < %v", pc.GetUpdatedAt().AsTime(), pc.GetCreatedAt().AsTime())
		}

		npcResp, err := suite.character.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       "Test NPC",
			Kind:       statev1.CharacterKind_NPC,
		})
		if err != nil {
			t.Fatalf("create NPC character: %v", err)
		}
		npc := npcResp.GetCharacter()
		if npc.GetId() == "" {
			t.Fatal("NPC character id is empty")
		}
		if npc.GetNotes() != "" {
			t.Fatalf("expected empty notes for NPC, got %q", npc.GetNotes())
		}
	})

	t.Run("character control set", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()

		characterResp, err := suite.character.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       "Test Character",
			Kind:       statev1.CharacterKind_PC,
		})
		if err != nil {
			t.Fatalf("create character: %v", err)
		}
		characterID := characterResp.GetCharacter().GetId()

		// Set to GM control (empty participant_id)
		gmControlResp, err := suite.character.SetDefaultControl(ctx, &statev1.SetDefaultControlRequest{
			CampaignId:    campaignID,
			CharacterId:   characterID,
			ParticipantId: wrapperspb.String(""),
		})
		if err != nil {
			t.Fatalf("set GM control: %v", err)
		}
		if gmControlResp.GetParticipantId().GetValue() != "" {
			t.Fatalf("expected empty participant id, got %q", gmControlResp.GetParticipantId().GetValue())
		}

		// Create a participant for player control
		participantResp, err := suite.participant.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
			CampaignId: campaignID,
			Name:       "Test Player",
			Role:       statev1.ParticipantRole_PLAYER,
		})
		if err != nil {
			t.Fatalf("create participant: %v", err)
		}
		participantID := participantResp.GetParticipant().GetId()

		// Set to participant control
		playerControlResp, err := suite.character.SetDefaultControl(ctx, &statev1.SetDefaultControlRequest{
			CampaignId:    campaignID,
			CharacterId:   characterID,
			ParticipantId: wrapperspb.String(participantID),
		})
		if err != nil {
			t.Fatalf("set participant control: %v", err)
		}
		if playerControlResp.GetParticipantId().GetValue() != participantID {
			t.Fatalf("expected participant id %q, got %q", participantID, playerControlResp.GetParticipantId().GetValue())
		}
	})

	t.Run("campaign lifecycle", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()
		ctx = suite.ctx(ctx)

		campaignResp, err := suite.campaign.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HUMAN,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := campaignResp.GetCampaign().GetId()
		ownerPID := campaignResp.GetOwnerParticipant().GetId()

		participantResp, err := suite.participant.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
			CampaignId: campaignID,
			Name:       "Lifecycle Player",
			Role:       statev1.ParticipantRole_PLAYER,
			Controller: statev1.Controller_CONTROLLER_HUMAN,
		})
		if err != nil {
			t.Fatalf("create participant: %v", err)
		}
		participantID := participantResp.GetParticipant().GetId()

		characterResp, err := suite.character.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       "Lifecycle Character",
			Kind:       statev1.CharacterKind_PC,
		})
		if err != nil {
			t.Fatalf("create character: %v", err)
		}
		characterID := characterResp.GetCharacter().GetId()

		_, err = suite.character.SetDefaultControl(ctx, &statev1.SetDefaultControlRequest{
			CampaignId:    campaignID,
			CharacterId:   characterID,
			ParticipantId: wrapperspb.String(participantID),
		})
		if err != nil {
			t.Fatalf("set character control: %v", err)
		}
		ensureDaggerheartCreationReadiness(t, ctx, suite.character, campaignID, characterID)

		_ = ensureSessionStartReadiness(t, ctx, suite.participant, suite.character, campaignID, ownerPID, characterID)

		sessionResp, err := suite.session.StartSession(ctx, &statev1.StartSessionRequest{CampaignId: campaignID})
		if err != nil {
			t.Fatalf("start session: %v", err)
		}
		sessionID := sessionResp.GetSession().GetId()

		_, err = suite.session.EndSession(ctx, &statev1.EndSessionRequest{CampaignId: campaignID, SessionId: sessionID})
		if err != nil {
			t.Fatalf("end session: %v", err)
		}

		endResp, err := suite.campaign.EndCampaign(ctx, &statev1.EndCampaignRequest{CampaignId: campaignID})
		if err != nil {
			t.Fatalf("end campaign: %v", err)
		}
		if endResp.GetCampaign().GetStatus() != statev1.CampaignStatus_COMPLETED {
			t.Fatalf("expected status COMPLETED, got %v", endResp.GetCampaign().GetStatus())
		}
		if endResp.GetCampaign().GetCompletedAt() == nil {
			t.Fatal("campaign completed_at is nil")
		}

		archiveResp, err := suite.campaign.ArchiveCampaign(ctx, &statev1.ArchiveCampaignRequest{CampaignId: campaignID})
		if err != nil {
			t.Fatalf("archive campaign: %v", err)
		}
		if archiveResp.GetCampaign().GetStatus() != statev1.CampaignStatus_ARCHIVED {
			t.Fatalf("expected status ARCHIVED, got %v", archiveResp.GetCampaign().GetStatus())
		}
		if archiveResp.GetCampaign().GetArchivedAt() == nil {
			t.Fatal("campaign archived_at is nil")
		}

		restoreResp, err := suite.campaign.RestoreCampaign(ctx, &statev1.RestoreCampaignRequest{CampaignId: campaignID})
		if err != nil {
			t.Fatalf("restore campaign: %v", err)
		}
		if restoreResp.GetCampaign().GetStatus() != statev1.CampaignStatus_DRAFT {
			t.Fatalf("expected status DRAFT, got %v", restoreResp.GetCampaign().GetStatus())
		}
		if restoreResp.GetCampaign().GetCompletedAt() != nil {
			t.Fatal("expected completed_at cleared after restore")
		}
		if restoreResp.GetCampaign().GetArchivedAt() != nil {
			t.Fatal("expected archived_at cleared after restore")
		}
	})
}

// campaignStatusString converts proto campaign status to a readable name.
func campaignStatusString(s statev1.CampaignStatus) string {
	return strings.TrimPrefix(s.String(), "CAMPAIGN_STATUS_")
}
