package protocol

import (
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTranscriptMessageNormalizesTranscriptFields(t *testing.T) {
	t.Parallel()

	got := TranscriptMessage(transcript.Message{
		MessageID:  " msg-1 ",
		CampaignID: " c1 ",
		SessionID:  " s1 ",
		SequenceID: 7,
		SentAt:     " 2026-03-13T12:00:00Z ",
		Actor: transcript.MessageActor{
			ParticipantID: " p1 ",
			Name:          " Avery ",
		},
		Body:            " hello ",
		ClientMessageID: " cm-1 ",
	})

	if got.MessageID != "msg-1" {
		t.Fatalf("MessageID = %q, want %q", got.MessageID, "msg-1")
	}
	if got.CampaignID != "c1" || got.SessionID != "s1" {
		t.Fatalf("scope = (%q, %q), want (%q, %q)", got.CampaignID, got.SessionID, "c1", "s1")
	}
	if got.Actor.ParticipantID != "p1" || got.Actor.Name != "Avery" {
		t.Fatalf("actor = %#v", got.Actor)
	}
	if got.Body != "hello" || got.ClientMessageID != "cm-1" {
		t.Fatalf("message = %#v", got)
	}
}

func TestTranscriptMessagesPreservesOrdering(t *testing.T) {
	t.Parallel()

	got := TranscriptMessages([]transcript.Message{
		{MessageID: "m1", SequenceID: 1},
		{MessageID: "m2", SequenceID: 2},
	})

	if len(got) != 2 {
		t.Fatalf("len = %d, want %d", len(got), 2)
	}
	if got[0].MessageID != "m1" || got[1].MessageID != "m2" {
		t.Fatalf("messages = %#v", got)
	}
}

func TestInteractionStateFromGameStateBuildsPlayOwnedDTO(t *testing.T) {
	t.Parallel()

	got := InteractionStateFromGameState(&gamev1.InteractionState{
		CampaignId:   " c1 ",
		CampaignName: " Guildhouse ",
		Viewer: &gamev1.InteractionViewer{
			ParticipantId: " p1 ",
			Name:          " Avery ",
			Role:          gamev1.ParticipantRole_PLAYER,
		},
		ActiveSession: &gamev1.InteractionSession{
			SessionId: " s1 ",
			Name:      " Session One ",
		},
	})

	if got.CampaignID != "c1" || got.CampaignName != "Guildhouse" {
		t.Fatalf("state = %#v", got)
	}
	if got.Viewer == nil || got.Viewer.ParticipantID != "p1" || got.Viewer.Name != "Avery" || got.Viewer.Role != "player" {
		t.Fatalf("viewer = %#v", got.Viewer)
	}
	if got.ActiveSession == nil || got.ActiveSession.SessionID != "s1" || got.ActiveSession.Name != "Session One" {
		t.Fatalf("active session = %#v", got.ActiveSession)
	}
}

func TestInteractionStateFromGameStateMapsAllFields(t *testing.T) {
	t.Parallel()

	got := InteractionStateFromGameState(&gamev1.InteractionState{
		CampaignId:   "c1",
		CampaignName: "Guildhouse",
		Locale:       commonv1.Locale_LOCALE_EN_US,
		Viewer:       &gamev1.InteractionViewer{ParticipantId: "p1", Name: "Avery", Role: gamev1.ParticipantRole_PLAYER},
		ActiveScene: &gamev1.InteractionScene{
			SceneId:     "scene-1",
			Name:        "The Tavern",
			Description: "A dimly lit tavern.",
			Characters: []*gamev1.InteractionCharacter{
				{CharacterId: "ch-1", Name: "Lark", OwnerParticipantId: "p1"},
			},
			GmOutput: &gamev1.InteractionGMOutput{
				Text:          "You enter the tavern.",
				ParticipantId: "p2",
				UpdatedAt:     &timestamppb.Timestamp{Seconds: 1710331200},
			},
		},
		PlayerPhase: &gamev1.ScenePlayerPhase{
			PhaseId:              "phase-1",
			Status:               gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS,
			FrameText:            "What do you do?",
			ActingCharacterIds:   []string{"ch-1"},
			ActingParticipantIds: []string{"p1"},
			Slots: []*gamev1.ScenePlayerSlot{
				{
					ParticipantId: "p1",
					SummaryText:   "I look around",
					Yielded:       false,
					ReviewStatus:  gamev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_OPEN,
				},
			},
		},
		Ooc: &gamev1.OOCState{
			Open: true,
			Posts: []*gamev1.OOCPost{
				{PostId: "ooc-1", ParticipantId: "p1", Body: "brb"},
			},
			ReadyToResumeParticipantIds: []string{"p1"},
		},
		GmAuthorityParticipantId: "p2",
		AiTurn: &gamev1.AITurnState{
			Status:             gamev1.AITurnStatus_AI_TURN_STATUS_RUNNING,
			OwnerParticipantId: "p2",
		},
	})

	if got.Locale != "en-us" {
		t.Fatalf("locale = %q, want %q", got.Locale, "en-us")
	}
	if got.ActiveScene == nil || got.ActiveScene.SceneID != "scene-1" {
		t.Fatalf("active_scene = %#v", got.ActiveScene)
	}
	if len(got.ActiveScene.Characters) != 1 || got.ActiveScene.Characters[0].CharacterID != "ch-1" {
		t.Fatalf("scene characters = %#v", got.ActiveScene.Characters)
	}
	if got.ActiveScene.GMOutput == nil || got.ActiveScene.GMOutput.Text != "You enter the tavern." {
		t.Fatalf("gm_output = %#v", got.ActiveScene.GMOutput)
	}
	if got.PlayerPhase == nil || got.PlayerPhase.PhaseID != "phase-1" || got.PlayerPhase.Status != "players" {
		t.Fatalf("player_phase = %#v", got.PlayerPhase)
	}
	if len(got.PlayerPhase.Slots) != 1 || got.PlayerPhase.Slots[0].ReviewStatus != "open" {
		t.Fatalf("slots = %#v", got.PlayerPhase.Slots)
	}
	if got.OOC == nil || !got.OOC.Open || len(got.OOC.Posts) != 1 {
		t.Fatalf("ooc = %#v", got.OOC)
	}
	if got.GMAuthorityParticipantID != "p2" {
		t.Fatalf("gm_authority = %q", got.GMAuthorityParticipantID)
	}
	if got.AITurn == nil || got.AITurn.Status != "running" || got.AITurn.OwnerParticipantID != "p2" {
		t.Fatalf("ai_turn = %#v", got.AITurn)
	}
}

func TestSceneFromGameSceneNilReturnsNil(t *testing.T) {
	t.Parallel()
	if got := SceneFromGameScene(nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestPlayerPhaseFromGamePhaseNilReturnsNil(t *testing.T) {
	t.Parallel()
	if got := PlayerPhaseFromGamePhase(nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestOOCFromGameOOCNilReturnsNil(t *testing.T) {
	t.Parallel()
	if got := OOCFromGameOOC(nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestAITurnFromGameAITurnNilReturnsNil(t *testing.T) {
	t.Parallel()
	if got := AITurnFromGameAITurn(nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestParticipantFromGameParticipant(t *testing.T) {
	t.Parallel()
	got := ParticipantFromGameParticipant("https://cdn.example.com/assets", &gamev1.Participant{
		Id:            " p1 ",
		UserId:        " user-1 ",
		CampaignId:    " camp-1 ",
		Name:          " Avery ",
		Role:          gamev1.ParticipantRole_PLAYER,
		AvatarSetId:   " avatar_set_v1 ",
		AvatarAssetId: " ceremonial_choir_lead ",
	})
	if got.ID != "p1" || got.Name != "Avery" || got.Role != "player" {
		t.Fatalf("participant = %#v", got)
	}
	if !strings.Contains(got.AvatarURL, "ceremonial_choir_lead") {
		t.Fatalf("avatar_url = %q, want asset-backed URL", got.AvatarURL)
	}
}
