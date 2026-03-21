package interactiontransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestDefaultGMAuthorityParticipantPrefersOwnerHumanForHybridCampaigns(t *testing.T) {
	t.Parallel()

	record, err := defaultGMAuthorityParticipant(
		storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeHybrid},
		[]storage.ParticipantRecord{
			{ID: "gm-z", Role: participant.RoleGM, Controller: participant.ControllerHuman},
			{ID: "gm-a", Role: participant.RoleGM, Controller: participant.ControllerHuman, CampaignAccess: participant.CampaignAccessOwner},
			{ID: "gm-ai", Role: participant.RoleGM, Controller: participant.ControllerAI},
		},
	)
	if err != nil {
		t.Fatalf("defaultGMAuthorityParticipant error = %v", err)
	}
	if record.ID != "gm-a" {
		t.Fatalf("participant id = %q, want owner human gm", record.ID)
	}
}

func TestDefaultGMAuthorityParticipantSelectsAIGMForAICampaigns(t *testing.T) {
	t.Parallel()

	record, err := defaultGMAuthorityParticipant(
		storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI},
		[]storage.ParticipantRecord{
			{ID: "gm-b", Role: participant.RoleGM, Controller: participant.ControllerAI},
			{ID: "gm-a", Role: participant.RoleGM, Controller: participant.ControllerAI},
			{ID: "gm-human", Role: participant.RoleGM, Controller: participant.ControllerHuman},
		},
	)
	if err != nil {
		t.Fatalf("defaultGMAuthorityParticipant error = %v", err)
	}
	if record.ID != "gm-a" {
		t.Fatalf("participant id = %q, want lexicographically first ai gm", record.ID)
	}
}

func TestDefaultGMAuthorityParticipantReturnsErrorWhenNoMatchingGMExists(t *testing.T) {
	t.Parallel()

	_, err := defaultGMAuthorityParticipant(
		storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI},
		[]storage.ParticipantRecord{{ID: "gm-human", Role: participant.RoleGM, Controller: participant.ControllerHuman}},
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindCampaignParticipantAndAITurnToken(t *testing.T) {
	t.Parallel()

	record, ok := findCampaignParticipant([]storage.ParticipantRecord{{ID: " p-1 "}}, "p-1")
	if !ok || record.ID != " p-1 " {
		t.Fatalf("findCampaignParticipant = %#v, %v", record, ok)
	}
	if _, ok := findCampaignParticipant(nil, "missing"); ok {
		t.Fatal("expected missing participant lookup to fail")
	}

	token := aiTurnToken(" sess-1 ", " gm-ai ", " scene.player_phase_review_started ", " scene-1 ", " phase-1 ")
	if token != "sess-1|gm-ai|scene.player_phase_review_started|scene-1|phase-1" {
		t.Fatalf("token = %q", token)
	}
}

func TestAITurnEligibilityRequiresAIBindingAndAIGMAuthority(t *testing.T) {
	t.Parallel()

	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["camp-1"] = map[string]storage.ParticipantRecord{
		"gm-ai":    {ID: "gm-ai", CampaignID: "camp-1", Role: participant.RoleGM, Controller: participant.ControllerAI},
		"gm-human": {ID: "gm-human", CampaignID: "camp-1", Role: participant.RoleGM, Controller: participant.ControllerHuman},
	}

	tests := []struct {
		name        string
		campaign    storage.CampaignRecord
		interaction storage.SessionInteraction
		sceneState  map[string]storage.SceneInteraction
		source      string
		wantOK      bool
		wantReason  string
	}{
		{
			name:        "missing binding",
			campaign:    storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI},
			interaction: storage.SessionInteraction{CampaignID: "camp-1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-ai"},
			sceneState: map[string]storage.SceneInteraction{
				"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-1", PhaseOpen: true, PhaseID: "phase-1"},
			},
			wantReason: "campaign ai binding is required",
		},
		{
			name:        "human gm authority",
			campaign:    storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeHybrid, AIAgentID: "agent-1"},
			interaction: storage.SessionInteraction{CampaignID: "camp-1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-human"},
			sceneState: map[string]storage.SceneInteraction{
				"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-1", PhaseOpen: true, PhaseID: "phase-1"},
			},
			wantReason: "gm authority participant is not ai-controlled",
		},
		{
			name:        "players still acting",
			campaign:    storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI, AIAgentID: "agent-1"},
			interaction: storage.SessionInteraction{CampaignID: "camp-1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-ai"},
			sceneState: map[string]storage.SceneInteraction{
				"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-1", PhaseOpen: true, PhaseID: "phase-1"},
			},
			wantReason: "scene player phase is open",
		},
		{
			name:        "eligible",
			campaign:    storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI, AIAgentID: "agent-1"},
			interaction: storage.SessionInteraction{CampaignID: "camp-1", SessionID: "sess-1", ActiveSceneID: "scene-2", GMAuthorityParticipantID: "gm-ai"},
			wantOK:      true,
		},
		{
			name:     "bootstrap without active scene is eligible",
			campaign: storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI, AIAgentID: "agent-1"},
			interaction: storage.SessionInteraction{
				CampaignID:               "camp-1",
				SessionID:                "sess-1",
				GMAuthorityParticipantID: "gm-ai",
			},
			source: "session.started",
			wantOK: true,
		},
		{
			name:     "gm review is eligible",
			campaign: storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI, AIAgentID: "agent-1"},
			interaction: storage.SessionInteraction{
				CampaignID:               "camp-1",
				SessionID:                "sess-1",
				ActiveSceneID:            "scene-1",
				GMAuthorityParticipantID: "gm-ai",
			},
			sceneState: map[string]storage.SceneInteraction{
				"camp-1:scene-1": {
					CampaignID:  "camp-1",
					SceneID:     "scene-1",
					SessionID:   "sess-1",
					PhaseOpen:   true,
					PhaseID:     "phase-1",
					PhaseStatus: scene.PlayerPhaseStatusGMReview,
				},
			},
			wantOK: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := interactionApplication{
				stores: interactionApplicationStores{
					Participant: participantStore,
					SceneInteraction: interactionSceneInteractionStoreStub{
						interactions: tc.sceneState,
					},
				},
			}
			got, err := app.aiTurnEligibility(context.Background(), tc.campaign, storage.SessionRecord{ID: "sess-1"}, tc.interaction, tc.source)
			if err != nil {
				t.Fatalf("aiTurnEligibility error = %v", err)
			}
			if got.ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", got.ok, tc.wantOK)
			}
			if got.reason != tc.wantReason {
				t.Fatalf("reason = %q, want %q", got.reason, tc.wantReason)
			}
			if tc.wantOK && got.ownerParticipant.ID != "gm-ai" {
				t.Fatalf("owner participant = %#v", got.ownerParticipant)
			}
		})
	}
}

func TestAIOrchestrationQueueAIGMTurnReturnsIdleWhenSessionIsNotCurrentOrEligible(t *testing.T) {
	t.Parallel()

	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := gametest.NewFakeParticipantStore()
	sessionInteractionStore := gametest.NewFakeSessionInteractionStore()
	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	token := aiTurnToken("sess-1", "gm-ai", "session.started", "", "")
	payload, err := json.Marshal(session.AITurnQueuedPayload{
		SessionID:          "sess-1",
		TurnToken:          token,
		OwnerParticipantID: "gm-ai",
		SourceEventType:    "session.started",
	})
	if err != nil {
		t.Fatalf("marshal ai turn payload: %v", err)
	}
	domain := &fakeDomainEngine{resultsByType: map[command.Type]engine.Result{
		commandTypeSessionAITurnQueue: {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        session.EventTypeAITurnQueued,
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				EntityType:  "session",
				EntityID:    "sess-1",
				PayloadJSON: payload,
			}),
		},
	}}

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

	runtime := gametest.SetupRuntime()
	runtime.SetInlineApplyEnabled(false)
	app := NewAIOrchestrationApplication(Deps{
		Campaign:           campaignStore,
		Session:            sessionStore,
		Participant:        participantStore,
		SessionInteraction: sessionInteractionStore,
		Write:              domainwrite.WritePath{Executor: domain, Runtime: runtime},
	}, gametest.FixedIDGenerator("unused"))

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
