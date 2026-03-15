package campaigns

import (
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

func TestCampaignGameHelperFallbacks(t *testing.T) {
	t.Parallel()

	if got := campaignGameSceneName(nil); got != "" {
		t.Fatalf("scene name = %q, want empty", got)
	}
	if got := campaignGameSceneDescription(nil); got != "" {
		t.Fatalf("scene description = %q, want empty", got)
	}
	if got := campaignGamePhaseLabel(nil); got != "GM turn" {
		t.Fatalf("phase label = %q, want GM turn", got)
	}
	if got := campaignGamePhaseFrame(nil); got != "" {
		t.Fatalf("phase frame = %q, want empty", got)
	}
	if got := campaignGameOOCSummary(campaignapp.CampaignGameOOCState{}); got != "In character" {
		t.Fatalf("ooc summary = %q, want in-character fallback", got)
	}
	if got := campaignGameGMAuthorityLabel(campaignapp.CampaignGameSurface{}); got != "Unassigned" {
		t.Fatalf("gm authority label = %q, want Unassigned", got)
	}
	if got := campaignGameAITurnStatus(campaignapp.CampaignGameSurface{}); got != "idle" {
		t.Fatalf("ai turn status = %q, want idle", got)
	}
	if got := campaignGameAITurnSummary(campaignapp.CampaignGameSurface{}); got != "No AI GM turn queued" {
		t.Fatalf("ai turn summary = %q, want idle summary", got)
	}
	if got := firstNonEmpty(" ", "", "value", "other"); got != "value" {
		t.Fatalf("firstNonEmpty = %q, want value", got)
	}
}

func TestCampaignGameCharacterAndPostViews(t *testing.T) {
	t.Parallel()

	surface := campaignapp.CampaignGameSurface{
		Participant: campaignapp.CampaignGameParticipant{ID: "p1"},
		ActiveScene: &campaignapp.CampaignGameScene{
			ID:          "scene-1",
			Name:        "Bridge",
			Description: "A narrow rope bridge.",
			Characters: []campaignapp.CampaignGameCharacter{
				{ID: "char-1", Name: "Aria", OwnerParticipantID: "p1"},
				{ID: "char-2", Name: "", OwnerParticipantID: "p2"},
			},
		},
		PlayerPhase: &campaignapp.CampaignGamePlayerPhase{
			Status:             "players",
			FrameText:          "What do you do?",
			ActingCharacterIDs: []string{"char-2"},
			Slots: []campaignapp.CampaignGamePlayerSlot{
				{ParticipantID: "p1", SummaryText: "Aria tests the ropes.", CharacterIDs: []string{"char-1"}},
				{ParticipantID: "p2", SummaryText: "Backup plan.", CharacterIDs: []string{"char-2"}, Yielded: true, ReviewStatus: "changes_requested", ReviewReason: "Corin does not know Fireball", ReviewCharacterIDs: []string{"char-2"}},
			},
		},
		OOC: campaignapp.CampaignGameOOCState{
			Open:                        true,
			ReadyToResumeParticipantIDs: []string{"p2"},
			Posts: []campaignapp.CampaignGameOOCPost{
				{ParticipantID: "p1", Body: "Quick ruling check."},
			},
		},
		GMAuthorityParticipantID: "gm-ai",
		AITurn: campaignapp.CampaignGameAITurn{
			Status:             "failed",
			OwnerParticipantID: "gm-ai",
			LastError:          "provider timeout",
		},
	}

	characters := campaignGameCharacterViews(surface.ActiveScene, surface.PlayerPhase)
	if len(characters) != 2 || characters[0].Active || !characters[1].Active {
		t.Fatalf("character views = %#v", characters)
	}

	sceneCharacters := campaignGameSceneCharacters(surface.ActiveScene)
	if sceneCharacters[1].Name != "char-2" {
		t.Fatalf("scene character fallback = %#v", sceneCharacters)
	}

	slots := campaignGameSlotViews(surface)
	if len(slots) != 2 {
		t.Fatalf("slot views = %#v", slots)
	}
	if !slots[0].Viewer || slots[0].CharacterLabel != "Aria" {
		t.Fatalf("viewer slot = %#v", slots[0])
	}
	if !slots[1].Yielded || slots[1].CharacterLabel != "char-2" {
		t.Fatalf("yielded slot = %#v", slots[1])
	}
	if slots[1].ReviewLabel != "Changes requested" || slots[1].ReviewCharacterLabel != "char-2" {
		t.Fatalf("review slot = %#v", slots[1])
	}

	oocPosts := campaignGameOOCPostViews(surface)
	if len(oocPosts) != 1 || !oocPosts[0].Viewer {
		t.Fatalf("ooc post views = %#v", oocPosts)
	}

	if got := campaignGamePhaseLabel(surface.PlayerPhase); got != "Players acting" {
		t.Fatalf("phase label = %q, want players label", got)
	}
	if got := campaignGamePhaseStatus(surface.PlayerPhase); got != "players" {
		t.Fatalf("phase status = %q, want players", got)
	}
	surface.PlayerPhase.Status = "gm_review"
	if got := campaignGamePhaseLabel(surface.PlayerPhase); got != "GM reviewing" {
		t.Fatalf("phase label = %q, want gm review label", got)
	}
	surface.PlayerPhase.Status = "gm"
	if got := campaignGamePhaseLabel(surface.PlayerPhase); got != "GM turn" {
		t.Fatalf("phase label = %q, want gm label", got)
	}
	surface.PlayerPhase.Status = "mystery"
	if got := campaignGamePhaseLabel(surface.PlayerPhase); got != "Scene phase" {
		t.Fatalf("phase label = %q, want fallback label", got)
	}
	if got := campaignGamePhaseFrame(surface.PlayerPhase); got != "What do you do?" {
		t.Fatalf("phase frame = %q, want frame", got)
	}
	if got := campaignGameOOCSummary(surface.OOC); got != "OOC paused · ready 1" {
		t.Fatalf("ooc summary = %q, want ready summary", got)
	}
	if got := campaignGameGMAuthorityLabel(surface); got != "gm-ai" {
		t.Fatalf("gm authority label = %q, want gm-ai", got)
	}
	if got := campaignGameAITurnStatus(surface); got != "failed" {
		t.Fatalf("ai turn status = %q, want failed", got)
	}
	if got := campaignGameAITurnSummary(surface); got != "AI GM turn failed" {
		t.Fatalf("ai turn summary = %q, want failed summary", got)
	}
	if got := campaignGameAITurnError(surface); got != "provider timeout" {
		t.Fatalf("ai turn error = %q, want provider timeout", got)
	}
	if got := campaignGameYieldedParticipants(surface); len(got) != 1 || got[0] != "p2" {
		t.Fatalf("yielded participants = %#v", got)
	}
	if got := campaignGameYieldedParticipants(campaignapp.CampaignGameSurface{}); len(got) != 0 {
		t.Fatalf("nil phase yielded participants = %#v, want empty", got)
	}
	if label, badgeClass := campaignGameSlotReviewLabel("under_review"); label != "Under review" || badgeClass != "badge-outline" {
		t.Fatalf("slot review label = %q/%q", label, badgeClass)
	}
	if got := normalizeCampaignGameSlotReviewStatus(""); got != "open" {
		t.Fatalf("normalizeCampaignGameSlotReviewStatus(\"\") = %q, want open", got)
	}
}
