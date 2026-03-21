package interactiontransport

import (
	"context"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

type interactionServiceHarness struct {
	campaign           *gametest.FakeCampaignStore
	participants       *gametest.FakeParticipantStore
	characters         *gametest.FakeCharacterStore
	sessions           *gametest.FakeSessionStore
	sessionInteraction *gametest.FakeSessionInteractionStore
	sceneStore         interactionSceneStoreStub
	sceneCharacters    interactionSceneCharacterStoreStub
	sceneInteraction   interactionSceneInteractionStoreStub
}

func newInteractionServiceHarness() *interactionServiceHarness {
	h := &interactionServiceHarness{
		campaign:           gametest.NewFakeCampaignStore(),
		participants:       gametest.NewFakeParticipantStore(),
		characters:         gametest.NewFakeCharacterStore(),
		sessions:           gametest.NewFakeSessionStore(),
		sessionInteraction: &gametest.FakeSessionInteractionStore{},
		sceneStore:         interactionSceneStoreStub{scenes: map[string]storage.SceneRecord{}},
		sceneCharacters:    interactionSceneCharacterStoreStub{records: map[string][]storage.SceneCharacterRecord{}},
		sceneInteraction:   interactionSceneInteractionStoreStub{interactions: map[string]storage.SceneInteraction{}},
	}

	h.campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    bridge.SystemIDDaggerheart,
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeHybrid,
		AIAgentID: "",
	}
	h.participants.Participants["c1"] = map[string]storage.ParticipantRecord{
		"gm-1": {
			ID:             "gm-1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessOwner,
		},
		"gm-ai": {
			ID:             "gm-ai",
			CampaignID:     "c1",
			Name:           "AI GM",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessManager,
		},
		"player-1": {
			ID:             "player-1",
			CampaignID:     "c1",
			Name:           "Aria",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessMember,
		},
		"player-2": {
			ID:             "player-2",
			CampaignID:     "c1",
			Name:           "Borin",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	h.characters.Characters["c1"] = map[string]storage.CharacterRecord{
		"char-1": {
			ID:                 "char-1",
			CampaignID:         "c1",
			Name:               "Aria",
			OwnerParticipantID: "player-1",
		},
		"char-2": {
			ID:                 "char-2",
			CampaignID:         "c1",
			Name:               "Borin",
			OwnerParticipantID: "player-2",
		},
	}
	h.sessions.Sessions["c1"] = map[string]storage.SessionRecord{
		"sess-1": {
			ID:         "sess-1",
			CampaignID: "c1",
			Name:       "Session One",
			Status:     session.StatusActive,
		},
		"sess-2": {
			ID:         "sess-2",
			CampaignID: "c1",
			Name:       "Other Session",
			Status:     session.StatusActive,
		},
	}
	h.sessions.ActiveSession["c1"] = "sess-1"
	h.sceneStore.scenes["c1:scene-1"] = storage.SceneRecord{
		CampaignID:  "c1",
		SceneID:     "scene-1",
		SessionID:   "sess-1",
		Name:        "Bridge",
		Description: "A rope bridge sways over a chasm.",
	}
	h.sceneStore.scenes["c1:scene-2"] = storage.SceneRecord{
		CampaignID:  "c1",
		SceneID:     "scene-2",
		SessionID:   "sess-2",
		Name:        "Inn",
		Description: "A quiet inn.",
	}
	h.sceneStore.scenes["c1:scene-3"] = storage.SceneRecord{
		CampaignID:  "c1",
		SceneID:     "scene-3",
		SessionID:   "sess-1",
		Name:        "Forest",
		Description: "A moonlit trail.",
	}
	h.sceneCharacters.records["c1:scene-1"] = []storage.SceneCharacterRecord{
		{CampaignID: "c1", SceneID: "scene-1", CharacterID: "char-1"},
		{CampaignID: "c1", SceneID: "scene-1", CharacterID: "char-2"},
	}
	return h
}

func (h *interactionServiceHarness) deps() Deps {
	return Deps{
		Auth: authz.PolicyDeps{
			Participant: h.participants,
			Character:   h.characters,
		},
		Campaign:           h.campaign,
		Participant:        h.participants,
		Character:          h.characters,
		Session:            h.sessions,
		SessionInteraction: h.sessionInteraction,
		Scene:              h.sceneStore,
		SceneCharacter:     h.sceneCharacters,
		SceneInteraction:   h.sceneInteraction,
	}
}

func (h *interactionServiceHarness) service() *InteractionService {
	return NewInteractionService(h.deps())
}

func (h *interactionServiceHarness) serviceWithSuccessfulWrite(t *testing.T) *InteractionService {
	t.Helper()

	runtime := testWriteRuntime(t)
	runtime.SetInlineApplyEnabled(false)
	deps := h.deps()
	deps.Write = domainwriteexec.WritePath{
		Executor: fakeDomainExecutor{
			result: testAcceptedDomainResult(),
		},
		Runtime: runtime,
	}
	return NewInteractionService(deps)
}

func testAcceptedDomainResult() engine.Result {
	return engine.Result{
		Decision: command.Decision{Events: []event.Event{testDecisionEvent()}},
	}
}

func TestNewInteractionServiceGetInteractionStateReturnsProjectedSnapshot(t *testing.T) {
	t.Parallel()

	h := newInteractionServiceHarness()
	now := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	h.sessionInteraction.Values = map[string]storage.SessionInteraction{
		"c1:sess-1": {
			CampaignID:                  "c1",
			SessionID:                   "sess-1",
			ActiveSceneID:               "scene-1",
			OOCPaused:                   true,
			GMAuthorityParticipantID:    "gm-1",
			ReadyToResumeParticipantIDs: []string{"player-2", "player-1"},
			OOCPosts: []storage.SessionOOCPost{
				{PostID: "ooc-1", ParticipantID: "player-1", Body: "Do we roll with hope?", CreatedAt: now},
			},
			AITurn: storage.SessionAITurn{
				Status:             session.AITurnStatusFailed,
				TurnToken:          "turn-1",
				OwnerParticipantID: "gm-ai",
				SourceEventType:    "scene.player_phase_ended",
				SourceSceneID:      "scene-1",
				SourcePhaseID:      "phase-1",
				LastError:          "provider timeout",
			},
		},
	}
	h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
		CampaignID:           "c1",
		SceneID:              "scene-1",
		SessionID:            "sess-1",
		PhaseOpen:            true,
		PhaseID:              "phase-1",
		PhaseStatus:          scene.PlayerPhaseStatusPlayers,
		FrameText:            "What do you do next?",
		ActingCharacterIDs:   []string{"char-1", "char-2"},
		ActingParticipantIDs: []string{"player-1", "player-2"},
		Slots: []storage.ScenePlayerSlot{
			{ParticipantID: "player-1", SummaryText: "Aria draws steel.", CharacterIDs: []string{"char-1"}, UpdatedAt: now},
			{ParticipantID: "player-2", Yielded: true},
		},
		GMOutputText:          "The bridge groans beneath your boots.",
		GMOutputParticipantID: "gm-ai",
		GMOutputUpdatedAt:     &now,
	}

	resp, err := h.service().GetInteractionState(
		gametest.ContextWithParticipantID("player-1"),
		&gamev1.GetInteractionStateRequest{CampaignId: "c1"},
	)
	if err != nil {
		t.Fatalf("GetInteractionState error = %v", err)
	}
	if resp.GetState().GetViewer().GetParticipantId() != "player-1" {
		t.Fatalf("viewer participant = %q", resp.GetState().GetViewer().GetParticipantId())
	}
	if resp.GetState().GetActiveSession().GetSessionId() != "sess-1" {
		t.Fatalf("active session = %#v", resp.GetState().GetActiveSession())
	}
	if resp.GetState().GetActiveScene().GetSceneId() != "scene-1" {
		t.Fatalf("active scene = %#v", resp.GetState().GetActiveScene())
	}
	if resp.GetState().GetActiveScene().GetGmOutput().GetText() != "The bridge groans beneath your boots." {
		t.Fatalf("gm output = %#v", resp.GetState().GetActiveScene().GetGmOutput())
	}
	if resp.GetState().GetPlayerPhase().GetPhaseId() != "phase-1" {
		t.Fatalf("player phase = %#v", resp.GetState().GetPlayerPhase())
	}
	if !resp.GetState().GetOoc().GetOpen() || len(resp.GetState().GetOoc().GetReadyToResumeParticipantIds()) != 2 {
		t.Fatalf("ooc state = %#v", resp.GetState().GetOoc())
	}
	if resp.GetState().GetAiTurn().GetStatus() != gamev1.AITurnStatus_AI_TURN_STATUS_FAILED {
		t.Fatalf("ai turn = %#v", resp.GetState().GetAiTurn())
	}
}

func TestInteractionServiceSceneRPCPreconditions(t *testing.T) {
	t.Parallel()

	t.Run("set active scene rejects scene outside active session", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1"},
		}

		_, err := h.service().SetActiveScene(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.SetActiveSceneRequest{CampaignId: "c1", SceneId: "scene-2"},
		)
		assertStatusCode(t, err, codes.FailedPrecondition)
	})

	t.Run("set active scene reaches write path for active-session scene", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1"},
		}

		_, err := h.service().SetActiveScene(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.SetActiveSceneRequest{CampaignId: "c1", SceneId: "scene-1"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("start scene player phase rejects while ooc paused", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", OOCPaused: true},
		}

		_, err := h.service().StartScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.StartScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1", CharacterIds: []string{"char-1"}},
		)
		assertStatusCode(t, err, codes.FailedPrecondition)
	})

	t.Run("submit scene player post rejects while ooc paused", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", OOCPaused: true},
		}

		_, err := h.service().SubmitScenePlayerPost(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.SubmitScenePlayerPostRequest{CampaignId: "c1", SceneId: "scene-1", SummaryText: "I rush ahead."},
		)
		assertStatusCode(t, err, codes.FailedPrecondition)
	})

	t.Run("yield scene player phase rejects when no phase is open", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID: "c1",
			SceneID:    "scene-1",
			SessionID:  "sess-1",
		}

		_, err := h.service().YieldScenePlayerPhase(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.YieldScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		)
		assertStatusCode(t, err, codes.FailedPrecondition)
	})

	t.Run("unyield scene player phase rejects non-acting participant", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			ActingParticipantIDs: []string{"player-2"},
		}

		_, err := h.service().UnyieldScenePlayerPhase(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.UnyieldScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		)
		assertStatusCode(t, err, codes.PermissionDenied)
	})

	t.Run("end scene player phase rejects when no phase is open", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID: "c1",
			SceneID:    "scene-1",
			SessionID:  "sess-1",
		}

		_, err := h.service().EndScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.EndScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		)
		assertStatusCode(t, err, codes.FailedPrecondition)
	})
}

func TestInteractionServiceOOCAndAuthorityRPCsReachWritePathBoundary(t *testing.T) {
	t.Parallel()

	t.Run("pause session for ooc reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1"},
		}

		_, err := h.service().PauseSessionForOOC(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.PauseSessionForOOCRequest{CampaignId: "c1", Reason: "rules question"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("pause session for ooc closes an open phase first", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID: "c1",
			SceneID:    "scene-1",
			SessionID:  "sess-1",
			PhaseOpen:  true,
			PhaseID:    "phase-1",
		}

		_, err := h.service().PauseSessionForOOC(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.PauseSessionForOOCRequest{CampaignId: "c1", Reason: "rules question"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("post session ooc reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", OOCPaused: true},
		}

		_, err := h.service().PostSessionOOC(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.PostSessionOOCRequest{CampaignId: "c1", Body: "Can I help here?"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("mark ooc ready reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", OOCPaused: true},
		}

		_, err := h.service().MarkOOCReadyToResume(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.MarkOOCReadyToResumeRequest{CampaignId: "c1"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("clear ooc ready reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", OOCPaused: true},
		}

		_, err := h.service().ClearOOCReadyToResume(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.ClearOOCReadyToResumeRequest{CampaignId: "c1"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("resume from ooc reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", OOCPaused: true},
		}

		_, err := h.service().ResumeFromOOC(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.ResumeFromOOCRequest{CampaignId: "c1"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("set session gm authority reaches write path for gm target", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1"},
		}

		_, err := h.service().SetSessionGMAuthority(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.SetSessionGMAuthorityRequest{CampaignId: "c1", ParticipantId: "gm-ai"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("retry ai gm turn reaches write path for failed eligible turn", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.campaign.Campaigns["c1"] = storage.CampaignRecord{
			ID:        "c1",
			Name:      "Test Campaign",
			System:    bridge.SystemIDDaggerheart,
			Status:    campaign.StatusActive,
			GmMode:    campaign.GmModeAI,
			AIAgentID: "agent-1",
		}
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {
				CampaignID:               "c1",
				SessionID:                "sess-1",
				ActiveSceneID:            "scene-1",
				GMAuthorityParticipantID: "gm-ai",
				AITurn: storage.SessionAITurn{
					Status:             session.AITurnStatusFailed,
					TurnToken:          "turn-1",
					OwnerParticipantID: "gm-ai",
					SourceEventType:    "scene.player_phase_ended",
					SourceSceneID:      "scene-1",
					SourcePhaseID:      "phase-1",
					LastError:          "timeout",
				},
			},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID: "c1",
			SceneID:    "scene-1",
			SessionID:  "sess-1",
			PhaseOpen:  false,
		}

		_, err := h.service().RetryAIGMTurn(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.RetryAIGMTurnRequest{CampaignId: "c1"},
		)
		assertStatusCode(t, err, codes.Internal)
	})
}

func TestInteractionServiceSceneRPCsReachWritePathBoundaryWhenPreconditionsPass(t *testing.T) {
	t.Parallel()

	t.Run("set active scene interrupts open phase before switching", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID: "c1",
			SceneID:    "scene-1",
			SessionID:  "sess-1",
			PhaseOpen:  true,
			PhaseID:    "phase-1",
		}

		_, err := h.service().SetActiveScene(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.SetActiveSceneRequest{CampaignId: "c1", SceneId: "scene-3"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("start scene player phase reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}

		_, err := h.service().StartScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.StartScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1", CharacterIds: []string{"char-1"}},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("submit scene player post reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			ActingCharacterIDs:   []string{"char-1"},
			ActingParticipantIDs: []string{"player-1"},
		}

		_, err := h.service().SubmitScenePlayerPost(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.SubmitScenePlayerPostRequest{
				CampaignId:   "c1",
				SceneId:      "scene-1",
				CharacterIds: []string{"char-1"},
				SummaryText:  "Aria advances.",
			},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("yield scene player phase reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			ActingParticipantIDs: []string{"player-1"},
		}

		_, err := h.service().YieldScenePlayerPhase(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.YieldScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("unyield scene player phase reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			ActingParticipantIDs: []string{"player-1"},
		}

		_, err := h.service().UnyieldScenePlayerPhase(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.UnyieldScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("end scene player phase reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID: "c1",
			SceneID:    "scene-1",
			SessionID:  "sess-1",
			PhaseOpen:  true,
			PhaseID:    "phase-1",
		}

		_, err := h.service().EndScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.EndScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1", Reason: "gm_interrupted"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("accept scene player phase reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:  "c1",
			SceneID:     "scene-1",
			SessionID:   "sess-1",
			PhaseOpen:   true,
			PhaseID:     "phase-1",
			PhaseStatus: scene.PlayerPhaseStatusGMReview,
		}

		_, err := h.service().AcceptScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.AcceptScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		)
		assertStatusCode(t, err, codes.Internal)
	})

	t.Run("request scene player revisions reaches write path", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			PhaseStatus:          scene.PlayerPhaseStatusGMReview,
			ActingCharacterIDs:   []string{"char-1", "char-2"},
			ActingParticipantIDs: []string{"player-1", "player-2"},
		}

		_, err := h.service().RequestScenePlayerRevisions(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.RequestScenePlayerRevisionsRequest{
				CampaignId: "c1",
				SceneId:    "scene-1",
				Revisions: []*gamev1.ScenePlayerRevisionRequest{
					{ParticipantId: "player-1", CharacterIds: []string{"char-1"}, Reason: "Revise the spell."},
				},
			},
		)
		assertStatusCode(t, err, codes.Internal)
	})
}

func TestInteractionServiceMutationRPCsSucceedWhenWriteRuntimeAcceptsEvents(t *testing.T) {
	t.Parallel()

	t.Run("set active scene", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1"},
		}

		resp, err := h.serviceWithSuccessfulWrite(t).SetActiveScene(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.SetActiveSceneRequest{CampaignId: "c1", SceneId: "scene-1"},
		)
		if err != nil || resp.GetState() == nil {
			t.Fatalf("SetActiveScene() = %#v, %v", resp, err)
		}
	})

	t.Run("start phase and submit player actions", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			ActingCharacterIDs:   []string{"char-1"},
			ActingParticipantIDs: []string{"player-1"},
		}
		svc := h.serviceWithSuccessfulWrite(t)

		if resp, err := svc.StartScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.StartScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1", CharacterIds: []string{"char-1"}, FrameText: "Act"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("StartScenePlayerPhase() = %#v, %v", resp, err)
		}
		if resp, err := svc.SubmitScenePlayerPost(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.SubmitScenePlayerPostRequest{CampaignId: "c1", SceneId: "scene-1", CharacterIds: []string{"char-1"}, SummaryText: "Advance", YieldAfterPost: true},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("SubmitScenePlayerPost() = %#v, %v", resp, err)
		}
		if resp, err := svc.YieldScenePlayerPhase(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.YieldScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("YieldScenePlayerPhase() = %#v, %v", resp, err)
		}
		if resp, err := svc.UnyieldScenePlayerPhase(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.UnyieldScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("UnyieldScenePlayerPhase() = %#v, %v", resp, err)
		}
		if resp, err := svc.EndScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.EndScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1", Reason: "gm_interrupted"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("EndScenePlayerPhase() = %#v, %v", resp, err)
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			PhaseStatus:          scene.PlayerPhaseStatusGMReview,
			ActingCharacterIDs:   []string{"char-1"},
			ActingParticipantIDs: []string{"player-1"},
		}
		if resp, err := svc.AcceptScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.AcceptScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("AcceptScenePlayerPhase() = %#v, %v", resp, err)
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			PhaseStatus:          scene.PlayerPhaseStatusGMReview,
			ActingCharacterIDs:   []string{"char-1"},
			ActingParticipantIDs: []string{"player-1"},
		}
		if resp, err := svc.RequestScenePlayerRevisions(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.RequestScenePlayerRevisionsRequest{
				CampaignId: "c1",
				SceneId:    "scene-1",
				Revisions: []*gamev1.ScenePlayerRevisionRequest{
					{ParticipantId: "player-1", CharacterIds: []string{"char-1"}, Reason: "Clarify action."},
				},
			},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("RequestScenePlayerRevisions() = %#v, %v", resp, err)
		}
	})

	t.Run("review phase requests revisions and accepts", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			PhaseStatus:          scene.PlayerPhaseStatusGMReview,
			ActingCharacterIDs:   []string{"char-1"},
			ActingParticipantIDs: []string{"player-1"},
			Slots: []storage.ScenePlayerSlot{{
				ParticipantID: "player-1",
				SummaryText:   "Advance",
				CharacterIDs:  []string{"char-1"},
				Yielded:       true,
				ReviewStatus:  scene.PlayerPhaseSlotReviewStatusUnderReview,
			}},
		}
		svc := h.serviceWithSuccessfulWrite(t)

		if resp, err := svc.RequestScenePlayerRevisions(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.RequestScenePlayerRevisionsRequest{
				CampaignId: "c1",
				SceneId:    "scene-1",
				Revisions: []*gamev1.ScenePlayerRevisionRequest{{
					ParticipantId: "player-1",
					Reason:        "Corin does not know Fireball.",
					CharacterIds:  []string{"char-1"},
				}},
			},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("RequestScenePlayerRevisions() = %#v, %v", resp, err)
		}

		if resp, err := svc.AcceptScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.AcceptScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("AcceptScenePlayerPhase() = %#v, %v", resp, err)
		}
	})

	t.Run("review actions require authoritative gm owner and revision reason", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-ai"},
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID:           "c1",
			SceneID:              "scene-1",
			SessionID:            "sess-1",
			PhaseOpen:            true,
			PhaseID:              "phase-1",
			PhaseStatus:          scene.PlayerPhaseStatusGMReview,
			ActingCharacterIDs:   []string{"char-1"},
			ActingParticipantIDs: []string{"player-1"},
		}
		svc := h.serviceWithSuccessfulWrite(t)

		_, err := svc.AcceptScenePlayerPhase(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.AcceptScenePlayerPhaseRequest{CampaignId: "c1", SceneId: "scene-1"},
		)
		assertStatusCode(t, err, codes.PermissionDenied)

		h.sessionInteraction.Values["c1:sess-1"] = storage.SessionInteraction{
			CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-1",
		}
		_, err = svc.RequestScenePlayerRevisions(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.RequestScenePlayerRevisionsRequest{
				CampaignId: "c1",
				SceneId:    "scene-1",
				Revisions: []*gamev1.ScenePlayerRevisionRequest{{
					ParticipantId: "player-1",
					Reason:        " ",
					CharacterIds:  []string{"char-1"},
				}},
			},
		)
		assertStatusCode(t, err, codes.InvalidArgument)
	})

	t.Run("ooc lifecycle and authority changes", func(t *testing.T) {
		t.Parallel()

		h := newInteractionServiceHarness()
		svc := h.serviceWithSuccessfulWrite(t)

		h.sessionInteraction.Values = map[string]storage.SessionInteraction{
			"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1"},
		}
		if resp, err := svc.PauseSessionForOOC(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.PauseSessionForOOCRequest{CampaignId: "c1", Reason: "rules"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("PauseSessionForOOC() = %#v, %v", resp, err)
		}

		h.sessionInteraction.Values["c1:sess-1"] = storage.SessionInteraction{
			CampaignID: "c1", SessionID: "sess-1", OOCPaused: true,
		}
		if resp, err := svc.PostSessionOOC(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.PostSessionOOCRequest{CampaignId: "c1", Body: "Question"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("PostSessionOOC() = %#v, %v", resp, err)
		}
		if resp, err := svc.MarkOOCReadyToResume(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.MarkOOCReadyToResumeRequest{CampaignId: "c1"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("MarkOOCReadyToResume() = %#v, %v", resp, err)
		}
		if resp, err := svc.ClearOOCReadyToResume(
			gametest.ContextWithParticipantID("player-1"),
			&gamev1.ClearOOCReadyToResumeRequest{CampaignId: "c1"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("ClearOOCReadyToResume() = %#v, %v", resp, err)
		}
		if resp, err := svc.ResumeFromOOC(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.ResumeFromOOCRequest{CampaignId: "c1"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("ResumeFromOOC() = %#v, %v", resp, err)
		}

		h.sessionInteraction.Values["c1:sess-1"] = storage.SessionInteraction{
			CampaignID: "c1", SessionID: "sess-1", ActiveSceneID: "scene-1", GMAuthorityParticipantID: "gm-ai",
			AITurn: storage.SessionAITurn{
				Status:             session.AITurnStatusFailed,
				TurnToken:          "turn-1",
				OwnerParticipantID: "gm-ai",
				SourceEventType:    "scene.player_phase_ended",
				SourceSceneID:      "scene-1",
				SourcePhaseID:      "phase-1",
			},
		}
		h.campaign.Campaigns["c1"] = storage.CampaignRecord{
			ID:        "c1",
			Name:      "Test Campaign",
			System:    bridge.SystemIDDaggerheart,
			Status:    campaign.StatusActive,
			GmMode:    campaign.GmModeAI,
			AIAgentID: "agent-1",
		}
		h.sceneInteraction.interactions["c1:scene-1"] = storage.SceneInteraction{
			CampaignID: "c1", SceneID: "scene-1", SessionID: "sess-1", PhaseOpen: false,
		}
		if resp, err := svc.SetSessionGMAuthority(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.SetSessionGMAuthorityRequest{CampaignId: "c1", ParticipantId: "gm-ai"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("SetSessionGMAuthority() = %#v, %v", resp, err)
		}
		if resp, err := svc.RetryAIGMTurn(
			gametest.ContextWithParticipantID("gm-1"),
			&gamev1.RetryAIGMTurnRequest{CampaignId: "c1"},
		); err != nil || resp.GetState() == nil {
			t.Fatalf("RetryAIGMTurn() = %#v, %v", resp, err)
		}
	})
}

func TestInteractionApplicationSetActiveScenePreservesOwningAIGMTurn(t *testing.T) {
	t.Parallel()

	h := newInteractionServiceHarness()
	h.campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    bridge.SystemIDDaggerheart,
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeAI,
		AIAgentID: "agent-1",
	}
	h.sessionInteraction.Values = map[string]storage.SessionInteraction{
		"c1:sess-1": {
			CampaignID:               "c1",
			SessionID:                "sess-1",
			GMAuthorityParticipantID: "gm-ai",
			AITurn: storage.SessionAITurn{
				Status:             session.AITurnStatusRunning,
				TurnToken:          "turn-1",
				OwnerParticipantID: "gm-ai",
				SourceEventType:    "session.started",
			},
		},
	}

	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{
		resultsByType: map[command.Type]engine.Result{
			commandTypeSessionActiveSceneSet: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        session.EventTypeActiveSceneSet,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					ActorID:     "gm-ai",
					SessionID:   "sess-1",
					EntityType:  "session",
					EntityID:    "sess-1",
					PayloadJSON: []byte(`{"session_id":"sess-1","active_scene_id":"scene-1"}`),
				}),
			},
			commandTypeSessionAITurnClear: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        session.EventTypeAITurnCleared,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					ActorID:     "gm-ai",
					SessionID:   "sess-1",
					EntityType:  "session",
					EntityID:    "sess-1",
					PayloadJSON: []byte(`{"session_id":"sess-1","turn_token":"turn-1","reason":"active_scene_switched"}`),
				}),
			},
		},
	}
	runtime := domainwrite.NewRuntime()
	runtime.SetInlineApplyEnabled(false)
	app := newInteractionApplicationWithDependencies(Deps{
		Auth:               authz.PolicyDeps{Participant: h.participants, Character: h.characters},
		Campaign:           h.campaign,
		Participant:        h.participants,
		Character:          h.characters,
		Session:            h.sessions,
		SessionInteraction: h.sessionInteraction,
		Scene:              h.sceneStore,
		SceneCharacter:     h.sceneCharacters,
		SceneInteraction:   h.sceneInteraction,
		Write:              domainwriteexec.WritePath{Executor: domain, Runtime: runtime},
	}, gametest.FixedIDGenerator("unused"))

	if _, err := app.SetActiveScene(
		gametest.ContextWithParticipantID("gm-ai"),
		"c1",
		&gamev1.SetActiveSceneRequest{SceneId: "scene-1"},
	); err != nil {
		t.Fatalf("SetActiveScene() error = %v", err)
	}

	if len(domain.commands) != 1 || domain.commands[0].Type != commandTypeSessionActiveSceneSet {
		t.Fatalf("commands = %#v, want only %q", domain.commands, commandTypeSessionActiveSceneSet)
	}
}

func TestInteractionApplicationSetActiveSceneClearsAIGMTurnForDifferentActor(t *testing.T) {
	t.Parallel()

	h := newInteractionServiceHarness()
	h.campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    bridge.SystemIDDaggerheart,
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeHybrid,
		AIAgentID: "agent-1",
	}
	h.sessionInteraction.Values = map[string]storage.SessionInteraction{
		"c1:sess-1": {
			CampaignID:               "c1",
			SessionID:                "sess-1",
			GMAuthorityParticipantID: "gm-1",
			AITurn: storage.SessionAITurn{
				Status:             session.AITurnStatusRunning,
				TurnToken:          "turn-1",
				OwnerParticipantID: "gm-ai",
				SourceEventType:    "session.started",
			},
		},
	}

	now := time.Date(2026, 3, 13, 12, 5, 0, 0, time.UTC)
	domain := &fakeDomainEngine{
		resultsByType: map[command.Type]engine.Result{
			commandTypeSessionAITurnClear: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        session.EventTypeAITurnCleared,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					ActorID:     "gm-1",
					SessionID:   "sess-1",
					EntityType:  "session",
					EntityID:    "sess-1",
					PayloadJSON: []byte(`{"session_id":"sess-1","turn_token":"turn-1","reason":"active_scene_switched"}`),
				}),
			},
			commandTypeSessionActiveSceneSet: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        session.EventTypeActiveSceneSet,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					ActorID:     "gm-1",
					SessionID:   "sess-1",
					EntityType:  "session",
					EntityID:    "sess-1",
					PayloadJSON: []byte(`{"session_id":"sess-1","active_scene_id":"scene-1"}`),
				}),
			},
		},
	}
	runtime := domainwrite.NewRuntime()
	runtime.SetInlineApplyEnabled(false)
	app := newInteractionApplicationWithDependencies(Deps{
		Auth:               authz.PolicyDeps{Participant: h.participants, Character: h.characters},
		Campaign:           h.campaign,
		Participant:        h.participants,
		Character:          h.characters,
		Session:            h.sessions,
		SessionInteraction: h.sessionInteraction,
		Scene:              h.sceneStore,
		SceneCharacter:     h.sceneCharacters,
		SceneInteraction:   h.sceneInteraction,
		Write:              domainwriteexec.WritePath{Executor: domain, Runtime: runtime},
	}, gametest.FixedIDGenerator("unused"))

	if _, err := app.SetActiveScene(
		gametest.ContextWithParticipantID("gm-1"),
		"c1",
		&gamev1.SetActiveSceneRequest{SceneId: "scene-1"},
	); err != nil {
		t.Fatalf("SetActiveScene() error = %v", err)
	}

	if len(domain.commands) != 2 || domain.commands[0].Type != commandTypeSessionAITurnClear || domain.commands[1].Type != commandTypeSessionActiveSceneSet {
		t.Fatalf("commands = %#v, want clear then set active", domain.commands)
	}
}

func TestInteractionServiceOOCRequiresPausedSessionForParticipantActions(t *testing.T) {
	t.Parallel()

	h := newInteractionServiceHarness()
	h.sessionInteraction.Values = map[string]storage.SessionInteraction{
		"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1"},
	}
	svc := h.service()

	_, err := svc.PostSessionOOC(
		gametest.ContextWithParticipantID("player-1"),
		&gamev1.PostSessionOOCRequest{CampaignId: "c1", Body: "hello"},
	)
	assertStatusCode(t, err, codes.FailedPrecondition)

	_, err = svc.MarkOOCReadyToResume(
		gametest.ContextWithParticipantID("player-1"),
		&gamev1.MarkOOCReadyToResumeRequest{CampaignId: "c1"},
	)
	assertStatusCode(t, err, codes.FailedPrecondition)

	_, err = svc.ClearOOCReadyToResume(
		gametest.ContextWithParticipantID("player-1"),
		&gamev1.ClearOOCReadyToResumeRequest{CampaignId: "c1"},
	)
	assertStatusCode(t, err, codes.FailedPrecondition)

	_, err = svc.ResumeFromOOC(
		gametest.ContextWithParticipantID("gm-1"),
		&gamev1.ResumeFromOOCRequest{CampaignId: "c1"},
	)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInteractionServiceSetSessionGMAuthorityRejectsPlayerTarget(t *testing.T) {
	t.Parallel()

	h := newInteractionServiceHarness()
	h.sessionInteraction.Values = map[string]storage.SessionInteraction{
		"c1:sess-1": {CampaignID: "c1", SessionID: "sess-1"},
	}

	_, err := h.service().SetSessionGMAuthority(
		gametest.ContextWithParticipantID("gm-1"),
		&gamev1.SetSessionGMAuthorityRequest{CampaignId: "c1", ParticipantId: "player-1"},
	)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInteractionServiceRetryAIGMTurnRequiresFailedTurn(t *testing.T) {
	t.Parallel()

	h := newInteractionServiceHarness()
	h.campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Test Campaign",
		System:    bridge.SystemIDDaggerheart,
		Status:    campaign.StatusActive,
		GmMode:    campaign.GmModeAI,
		AIAgentID: "agent-1",
	}
	h.sessionInteraction.Values = map[string]storage.SessionInteraction{
		"c1:sess-1": {
			CampaignID:               "c1",
			SessionID:                "sess-1",
			ActiveSceneID:            "scene-1",
			GMAuthorityParticipantID: "gm-ai",
			AITurn:                   storage.SessionAITurn{Status: session.AITurnStatusIdle},
		},
	}

	_, err := h.service().RetryAIGMTurn(
		gametest.ContextWithParticipantID("gm-1"),
		&gamev1.RetryAIGMTurnRequest{CampaignId: "c1"},
	)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInteractionServiceGetInteractionStateRequiresVisibleCampaign(t *testing.T) {
	t.Parallel()

	participantStore := gametest.NewFakeParticipantStore()
	svc := NewInteractionService(Deps{
		Auth:        authz.PolicyDeps{Participant: participantStore},
		Campaign:    interactionActiveCampaignStore("c1"),
		Participant: participantStore,
	})

	_, err := svc.GetInteractionState(context.Background(), &gamev1.GetInteractionStateRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func interactionActiveCampaignStore(campaignID string) *gametest.FakeCampaignStore {
	store := gametest.NewFakeCampaignStore()
	store.Campaigns[campaignID] = gametest.ActiveCampaignRecord(campaignID)
	return store
}
