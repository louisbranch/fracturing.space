package app

import (
	"context"
	"testing"
)

type gameReadGatewayStub struct {
	surface CampaignGameSurface
	err     error
}

func (g gameReadGatewayStub) CampaignGameSurface(context.Context, string) (CampaignGameSurface, error) {
	if g.err != nil {
		return CampaignGameSurface{}, g.err
	}
	return g.surface, nil
}

func TestGameServiceCampaignGameSurfaceNormalizesInteractionState(t *testing.T) {
	t.Parallel()

	service := gameService{read: gameReadGatewayStub{surface: CampaignGameSurface{
		Participant: CampaignGameParticipant{
			ID:   " p1 ",
			Name: " ",
		},
		SessionID:   " sess-1 ",
		SessionName: " ",
		ActiveScene: &CampaignGameScene{
			ID:          " scene-1 ",
			SessionID:   " sess-1 ",
			Name:        " ",
			Description: " bridge ",
			Characters: []CampaignGameCharacter{
				{ID: " char-2 ", Name: " ", OwnerParticipantID: " p2 "},
			},
		},
		PlayerPhase: &CampaignGamePlayerPhase{
			PhaseID:              " phase-1 ",
			Status:               " ",
			FrameText:            " frame ",
			ActingCharacterIDs:   []string{" char-2 ", " ", "char-1"},
			ActingParticipantIDs: []string{" p2 ", "", "p1"},
			Slots: []CampaignGamePlayerSlot{
				{
					ParticipantID:      " p1 ",
					SummaryText:        " move ",
					CharacterIDs:       []string{" char-1 ", ""},
					ReviewStatus:       " changes_requested ",
					ReviewReason:       " wrong spell ",
					ReviewCharacterIDs: []string{" char-1 ", ""},
				},
			},
		},
		OOC: CampaignGameOOCState{
			ReadyToResumeParticipantIDs: []string{" p2 ", "", "p1"},
			Posts: []CampaignGameOOCPost{
				{PostID: " ooc-1 ", ParticipantID: " p2 ", Body: " clarify "},
			},
		},
		GMAuthorityParticipantID: " gm-ai ",
		AITurn: CampaignGameAITurn{
			Status:             " failed ",
			TurnToken:          " turn-1 ",
			OwnerParticipantID: " gm-ai ",
			SourceEventType:    " scene.player_phase_review_started ",
			SourceSceneID:      " scene-1 ",
			SourcePhaseID:      " phase-1 ",
			LastError:          " timeout ",
		},
	}}}

	got, err := service.campaignGameSurface(context.Background(), " camp-1 ")
	if err != nil {
		t.Fatalf("campaignGameSurface error = %v", err)
	}

	if got.Participant.Name != "p1" || got.Participant.Role != "Unspecified" {
		t.Fatalf("participant = %#v", got.Participant)
	}
	if got.SessionName != "sess-1" {
		t.Fatalf("session = %#v", got)
	}
	if got.ActiveScene == nil || got.ActiveScene.Name != "scene-1" || got.ActiveScene.Characters[0].Name != "char-2" {
		t.Fatalf("active scene = %#v", got.ActiveScene)
	}
	if got.PlayerPhase == nil || got.PlayerPhase.Status != "gm" {
		t.Fatalf("player phase = %#v", got.PlayerPhase)
	}
	if len(got.PlayerPhase.ActingCharacterIDs) != 2 || got.PlayerPhase.ActingCharacterIDs[0] != "char-2" {
		t.Fatalf("acting characters = %#v", got.PlayerPhase.ActingCharacterIDs)
	}
	if len(got.PlayerPhase.Slots) != 1 || got.PlayerPhase.Slots[0].ReviewStatus != "changes_requested" || got.PlayerPhase.Slots[0].ReviewReason != "wrong spell" {
		t.Fatalf("player slots = %#v", got.PlayerPhase.Slots)
	}
	if len(got.OOC.ReadyToResumeParticipantIDs) != 2 || got.OOC.ReadyToResumeParticipantIDs[0] != "p2" {
		t.Fatalf("ooc ready ids = %#v", got.OOC.ReadyToResumeParticipantIDs)
	}
	if got.OOC.Posts[0].Body != "clarify" {
		t.Fatalf("ooc posts = %#v", got.OOC.Posts)
	}
	if got.GMAuthorityParticipantID != "gm-ai" || got.AITurn.Status != "failed" || got.AITurn.LastError != "timeout" {
		t.Fatalf("gm/ai turn = %#v / %#v", got.GMAuthorityParticipantID, got.AITurn)
	}
}

func TestGameServiceCampaignGameSurfaceRejectsBlankCampaignID(t *testing.T) {
	t.Parallel()

	service := gameService{read: gameReadGatewayStub{}}
	if _, err := service.campaignGameSurface(context.Background(), " "); err == nil {
		t.Fatal("expected blank campaign id to fail")
	}
}

func TestTrimStringSlice(t *testing.T) {
	t.Parallel()

	got := trimStringSlice([]string{" a ", "", " ", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("trimStringSlice = %#v", got)
	}
	if got := trimStringSlice(nil); len(got) != 0 {
		t.Fatalf("trimStringSlice(nil) = %#v, want empty", got)
	}
}
