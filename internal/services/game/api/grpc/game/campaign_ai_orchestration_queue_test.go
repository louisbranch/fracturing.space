package game

import (
	"context"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCampaignAIOrchestrationQueueAIGMTurnReturnsIdleWhenSessionIsNotCurrentOrEligible(t *testing.T) {
	t.Parallel()

	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := gametest.NewFakeParticipantStore()
	sessionInteractionStore := &gametest.FakeSessionInteractionStore{}

	campaignStore.Campaigns["camp-1"] = storage.CampaignRecord{
		ID:        "camp-1",
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeAI,
		AIAgentID: "agent-1",
	}
	sessionStore.Sessions["camp-1"] = map[string]storage.SessionRecord{
		"sess-1": {ID: "sess-1", CampaignID: "camp-1", Status: session.StatusActive},
	}
	sessionStore.ActiveSession["camp-1"] = "sess-1"
	participantStore.Participants["camp-1"] = map[string]storage.ParticipantRecord{
		"gm-ai": {ID: "gm-ai", CampaignID: "camp-1", Role: participant.RoleGM, Controller: participant.ControllerAI},
	}

	app := newCampaignAIOrchestrationApplicationWithDependencies(
		CampaignAIOrchestrationDeps{
			Campaign:           campaignStore,
			Session:            sessionStore,
			Participant:        participantStore,
			SessionInteraction: sessionInteractionStore,
			Applier:            projection.Applier{},
		},
		gametest.FixedIDGenerator("unused"),
	)

	state, err := app.QueueAIGMTurn(context.Background(), "camp-1", "sess-missing", "session.active_scene_set", "", "")
	if err != nil {
		t.Fatalf("QueueAIGMTurn mismatch error = %v", err)
	}
	if state.GetStatus() != gamev1.AITurnStatus_AI_TURN_STATUS_IDLE {
		t.Fatalf("mismatch status = %v, want idle", state.GetStatus())
	}

	sessionInteractionStore.Values = map[string]storage.SessionInteraction{
		"camp-1:sess-1": {
			CampaignID:               "camp-1",
			SessionID:                "sess-1",
			GMAuthorityParticipantID: "gm-ai",
			AITurn:                   storage.SessionAITurn{Status: session.AITurnStatusIdle},
		},
	}
	state, err = app.QueueAIGMTurn(context.Background(), "camp-1", "sess-1", "session.active_scene_set", "", "")
	if err != nil {
		t.Fatalf("QueueAIGMTurn ineligible error = %v", err)
	}
	if state.GetStatus() != gamev1.AITurnStatus_AI_TURN_STATUS_IDLE {
		t.Fatalf("ineligible status = %v, want idle", state.GetStatus())
	}
}
