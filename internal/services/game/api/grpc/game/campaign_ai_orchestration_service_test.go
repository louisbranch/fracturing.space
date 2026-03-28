package game

import (
	"context"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestNewCampaignAIOrchestrationServiceRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	svc := NewCampaignAIOrchestrationService(CampaignAIOrchestrationDeps{})
	ctx := context.Background()

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "queue nil", run: func() error { _, err := svc.QueueAIGMTurn(ctx, nil); return err }},
		{name: "queue missing campaign", run: func() error {
			_, err := svc.QueueAIGMTurn(ctx, &gamev1.QueueAIGMTurnRequest{SessionId: "sess-1"})
			return err
		}},
		{name: "start missing token", run: func() error {
			_, err := svc.StartAIGMTurn(ctx, &gamev1.StartAIGMTurnRequest{CampaignId: "c1", SessionId: "sess-1"})
			return err
		}},
		{name: "fail missing token", run: func() error {
			_, err := svc.FailAIGMTurn(ctx, &gamev1.FailAIGMTurnRequest{CampaignId: "c1", SessionId: "sess-1"})
			return err
		}},
		{name: "complete missing token", run: func() error {
			_, err := svc.CompleteAIGMTurn(ctx, &gamev1.CompleteAIGMTurnRequest{CampaignId: "c1", SessionId: "sess-1"})
			return err
		}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertStatusCode(t, tc.run(), codes.InvalidArgument)
		})
	}
}

func TestCampaignAIOrchestrationServiceQueueReturnsIdleWhenSessionIsNotEligible(t *testing.T) {
	t.Parallel()

	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := gametest.NewFakeParticipantStore()
	sessionInteractionStore := &gametest.FakeSessionInteractionStore{}

	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    bridge.SystemIDDaggerheart,
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeAI,
		AIAgentID: "agent-1",
	}
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"sess-1": {ID: "sess-1", CampaignID: "c1", Status: session.StatusActive},
	}
	sessionStore.ActiveSession["c1"] = "sess-1"
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"gm-ai": {
			ID:         "gm-ai",
			CampaignID: "c1",
			Role:       participant.RoleGM,
			Controller: participant.ControllerAI,
		},
	}

	svc := NewCampaignAIOrchestrationService(CampaignAIOrchestrationDeps{
		Campaign:           campaignStore,
		Session:            sessionStore,
		Participant:        participantStore,
		SessionInteraction: sessionInteractionStore,
		Applier:            projection.Applier{},
	})

	resp, err := svc.QueueAIGMTurn(context.Background(), &gamev1.QueueAIGMTurnRequest{
		CampaignId:      "c1",
		SessionId:       "sess-1",
		SourceEventType: "scene.player_phase_ended",
	})
	if err != nil {
		t.Fatalf("QueueAIGMTurn error = %v", err)
	}
	if resp.GetAiTurn().GetStatus() != gamev1.AITurnStatus_AI_TURN_STATUS_IDLE {
		t.Fatalf("ai turn status = %v, want idle", resp.GetAiTurn().GetStatus())
	}
}

func TestCampaignAIOrchestrationServiceLifecycleRPCsReachWritePathBoundary(t *testing.T) {
	t.Parallel()

	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	sessionInteractionStore := &gametest.FakeSessionInteractionStore{
		Values: map[string]storage.SessionInteraction{
			"c1:sess-1": {
				CampaignID: "c1",
				SessionID:  "sess-1",
				AITurn: storage.SessionAITurn{
					Status:    session.AITurnStatusQueued,
					TurnToken: "turn-1",
				},
			},
		},
	}

	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		System: bridge.SystemIDDaggerheart,
		Status: campaign.StatusActive,
	}
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"sess-1": {ID: "sess-1", CampaignID: "c1", Status: session.StatusActive},
	}
	sessionStore.ActiveSession["c1"] = "sess-1"

	svc := NewCampaignAIOrchestrationService(CampaignAIOrchestrationDeps{
		Campaign:           campaignStore,
		Session:            sessionStore,
		SessionInteraction: sessionInteractionStore,
		Applier:            projection.Applier{},
	})

	_, err := svc.StartAIGMTurn(context.Background(), &gamev1.StartAIGMTurnRequest{
		CampaignId: "c1",
		SessionId:  "sess-1",
		TurnToken:  "turn-1",
	})
	assertStatusCode(t, err, codes.Internal)

	_, err = svc.FailAIGMTurn(context.Background(), &gamev1.FailAIGMTurnRequest{
		CampaignId: "c1",
		SessionId:  "sess-1",
		TurnToken:  "turn-1",
		LastError:  "boom",
	})
	assertStatusCode(t, err, codes.Internal)

	_, err = svc.CompleteAIGMTurn(context.Background(), &gamev1.CompleteAIGMTurnRequest{
		CampaignId: "c1",
		SessionId:  "sess-1",
		TurnToken:  "turn-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCampaignAIOrchestrationApplicationCampaignSupportsAI(t *testing.T) {
	t.Parallel()

	app := newCampaignAIOrchestrationApplicationWithDependencies(CampaignAIOrchestrationDeps{}, runtimekit.FixedIDGenerator("unused"))
	if !app.CampaignSupportsAI(storage.CampaignRecord{GmMode: campaign.GmModeAI}) {
		t.Fatal("gm mode ai should be supported")
	}
	if !app.CampaignSupportsAI(storage.CampaignRecord{GmMode: campaign.GmModeHybrid}) {
		t.Fatal("gm mode hybrid should be supported")
	}
	if app.CampaignSupportsAI(storage.CampaignRecord{GmMode: campaign.GmModeHuman}) {
		t.Fatal("gm mode human should not be supported")
	}
}
