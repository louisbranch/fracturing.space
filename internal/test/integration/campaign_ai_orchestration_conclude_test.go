//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const concludeSessionIntegrationSummary = `## Key Events

The harbor war finally ended.

## NPCs Met

Captain Vale.

## Decisions Made

The party chose peace over pursuit.

## Unresolved Threads

Who backed the raiders?

## Next Session Hooks

None. The campaign is over.`

func TestCampaignAIOrchestrationConcludeSessionCompletesCampaign(t *testing.T) {
	fixture := newSuiteFixture(t)
	userID := fixture.newUserID(t, uniqueTestUsername(t, "conclude-session"))
	suite := fixture.newGameSuite(t, userID)
	sceneClient := gamev1.NewSceneServiceClient(suite.conn)

	internalConn := dialGRPCWithServiceID(t, fixture.grpcAddr, serviceaddr.ServiceAI)
	defer internalConn.Close()
	campaignAIOrchestrationClient := gamev1.NewCampaignAIOrchestrationServiceClient(internalConn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	ctxWithUser := suite.ctx(ctx)

	campaignResp, err := suite.campaign.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:   "Conclude Session Campaign",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: gamev1.GmMode_HUMAN,
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	campaignID := campaignResp.GetCampaign().GetId()
	ownerParticipantID := campaignResp.GetOwnerParticipant().GetId()

	participantResp, err := suite.participant.CreateParticipant(ctxWithUser, &gamev1.CreateParticipantRequest{
		CampaignId: campaignID,
		Name:       "Closer",
		Role:       gamev1.ParticipantRole_PLAYER,
		Controller: gamev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}
	participantID := participantResp.GetParticipant().GetId()

	characterResp, err := suite.character.CreateCharacter(ctxWithUser, &gamev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       "Harbor Hero",
		Kind:       gamev1.CharacterKind_PC,
	})
	if err != nil {
		t.Fatalf("create character: %v", err)
	}
	characterID := characterResp.GetCharacter().GetId()

	if _, err := suite.character.UpdateCharacter(ctxWithUser, &gamev1.UpdateCharacterRequest{
		CampaignId:         campaignID,
		CharacterId:        characterID,
		OwnerParticipantId: wrapperspb.String(participantID),
	}); err != nil {
		t.Fatalf("set character owner: %v", err)
	}

	ensureDaggerheartCreationReadiness(t, ctxWithUser, suite.character, campaignID, characterID)
	ensureSessionStartReadiness(t, ctxWithUser, suite.participant, suite.character, campaignID, ownerParticipantID, characterID)

	startSession := startSessionWithDefaultControllers(t, ctxWithUser, suite.session, suite.character, campaignID, "Last Watch")
	sessionID := startSession.GetSession().GetId()

	createSceneResp, err := sceneClient.CreateScene(ctxWithUser, &gamev1.CreateSceneRequest{
		CampaignId:   campaignID,
		SessionId:    sessionID,
		Name:         "Harbor Gate",
		Description:  "The final threat ebbs with the tide.",
		CharacterIds: []string{characterID},
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	sceneID := createSceneResp.GetSceneId()
	if sceneID == "" {
		t.Fatal("expected scene id")
	}

	resp, err := campaignAIOrchestrationClient.ConcludeSession(ctxWithUser, &gamev1.ConcludeSessionRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		Conclusion:  "Dawn breaks over a harbor finally at peace.",
		Summary:     concludeSessionIntegrationSummary,
		EndCampaign: true,
		Epilogue:    "Years later, sailors still tell the story of the night the harbor chose peace.",
	})
	if err != nil {
		t.Fatalf("conclude session: %v", err)
	}
	if !resp.GetCampaignCompleted() {
		t.Fatal("campaign_completed = false, want true")
	}
	if len(resp.GetEndedSceneIds()) != 1 || resp.GetEndedSceneIds()[0] != sceneID {
		t.Fatalf("ended_scene_ids = %#v, want [%s]", resp.GetEndedSceneIds(), sceneID)
	}

	campaignState, err := suite.campaign.GetCampaign(ctxWithUser, &gamev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if campaignState.GetCampaign().GetStatus() != gamev1.CampaignStatus_COMPLETED {
		t.Fatalf("campaign status = %v, want %v", campaignState.GetCampaign().GetStatus(), gamev1.CampaignStatus_COMPLETED)
	}
	if campaignState.GetCampaign().GetCompletedAt() == nil {
		t.Fatal("campaign completed_at = nil, want timestamp")
	}

	sessionState, err := suite.session.GetSession(ctxWithUser, &gamev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sessionState.GetSession().GetStatus() != gamev1.SessionStatus_SESSION_ENDED {
		t.Fatalf("session status = %v, want %v", sessionState.GetSession().GetStatus(), gamev1.SessionStatus_SESSION_ENDED)
	}

	recapResp, err := suite.session.GetSessionRecap(ctxWithUser, &gamev1.GetSessionRecapRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		t.Fatalf("get session recap: %v", err)
	}
	recapMarkdown := recapResp.GetRecap().GetMarkdown()
	if !strings.Contains(recapMarkdown, "## Campaign Epilogue") {
		t.Fatalf("recap markdown = %q, want campaign epilogue heading", recapMarkdown)
	}
	if !strings.Contains(recapMarkdown, "Years later, sailors still tell the story") {
		t.Fatalf("recap markdown = %q, want epilogue text", recapMarkdown)
	}
}
