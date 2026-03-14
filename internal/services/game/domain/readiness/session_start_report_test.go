package readiness

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestEvaluateSessionStartReport_IncludesBoundaryAndCoreBlockersInOrder(t *testing.T) {
	report := EvaluateSessionStartReport(
		aggregate.State{
			Campaign: campaign.State{
				Status: campaign.StatusCompleted,
				GmMode: campaign.GmModeAI,
			},
		},
		ReportOptions{
			IncludeSessionBoundary: true,
			HasActiveSession:       true,
		},
	)

	if report.Ready() {
		t.Fatal("expected non-ready report")
	}

	gotCodes := make([]string, 0, len(report.Blockers))
	for _, blocker := range report.Blockers {
		gotCodes = append(gotCodes, blocker.Code)
	}
	wantCodes := []string{
		RejectionCodeSessionReadinessCampaignStatusDisallowsStart,
		RejectionCodeSessionReadinessActiveSessionExists,
		RejectionCodeSessionReadinessAIAgentRequired,
		RejectionCodeSessionReadinessAIGMParticipantRequired,
		RejectionCodeSessionReadinessGMRequired,
		RejectionCodeSessionReadinessPlayerRequired,
	}
	if strings.Join(gotCodes, ",") != strings.Join(wantCodes, ",") {
		t.Fatalf("blocker codes = %v, want %v", gotCodes, wantCodes)
	}
}

func TestEvaluateSessionStartReport_CollectsCharacterAndPlayerBlockers(t *testing.T) {
	report := EvaluateSessionStartReport(
		aggregate.State{
			Participants: map[ids.ParticipantID]participant.State{
				"gm-1": {
					ParticipantID: "gm-1",
					Joined:        true,
					Role:          participant.RoleGM,
				},
				"player-1": {
					ParticipantID: "player-1",
					Joined:        true,
					Role:          participant.RolePlayer,
				},
				"player-2": {
					ParticipantID: "player-2",
					Joined:        true,
					Name:          "Player Two",
					Role:          participant.RolePlayer,
				},
			},
			Characters: map[ids.CharacterID]character.State{
				"char-b": {
					CharacterID:   "char-b",
					Created:       true,
					ParticipantID: "player-1",
				},
				"char-a": {
					CharacterID: "char-a",
					Created:     true,
					Name:        "Aria",
				},
			},
		},
		ReportOptions{
			SystemReadiness: func(string) (bool, string) {
				return false, "system profile is incomplete"
			},
		},
	)

	if report.Ready() {
		t.Fatal("expected non-ready report")
	}
	gotCodes := make([]string, 0, len(report.Blockers))
	for _, blocker := range report.Blockers {
		gotCodes = append(gotCodes, blocker.Code)
	}
	wantCodes := []string{
		RejectionCodeSessionReadinessCharacterControllerRequired,
		RejectionCodeSessionReadinessCharacterSystemRequired,
		RejectionCodeSessionReadinessCharacterSystemRequired,
		RejectionCodeSessionReadinessPlayerCharacterRequired,
	}
	if strings.Join(gotCodes, ",") != strings.Join(wantCodes, ",") {
		t.Fatalf("blocker codes = %v, want %v", gotCodes, wantCodes)
	}

	if report.Blockers[0].Metadata["character_id"] != "char-a" {
		t.Fatalf("first blocker character_id = %q, want %q", report.Blockers[0].Metadata["character_id"], "char-a")
	}
	if report.Blockers[0].Metadata["character_name"] != "Aria" {
		t.Fatalf("first blocker character_name = %q, want %q", report.Blockers[0].Metadata["character_name"], "Aria")
	}
	if !strings.Contains(report.Blockers[0].Message, "Aria") {
		t.Fatalf("first blocker message = %q, want character name", report.Blockers[0].Message)
	}
	if strings.Contains(report.Blockers[0].Message, "char-a") {
		t.Fatalf("first blocker message = %q, did not expect character id when name is present", report.Blockers[0].Message)
	}
	if report.Blockers[1].Metadata["character_name"] != "Aria" {
		t.Fatalf("system blocker character_name = %q, want %q", report.Blockers[1].Metadata["character_name"], "Aria")
	}
	if !strings.Contains(report.Blockers[1].Message, "Aria") {
		t.Fatalf("system blocker message = %q, want character name", report.Blockers[1].Message)
	}
	if strings.Contains(report.Blockers[1].Message, "char-a") {
		t.Fatalf("system blocker message = %q, did not expect character id when name is present", report.Blockers[1].Message)
	}
	if report.Blockers[3].Metadata["participant_id"] != "player-2" {
		t.Fatalf("player blocker participant_id = %q, want %q", report.Blockers[3].Metadata["participant_id"], "player-2")
	}
	if report.Blockers[3].Metadata["participant_name"] != "Player Two" {
		t.Fatalf("player blocker participant_name = %q, want %q", report.Blockers[3].Metadata["participant_name"], "Player Two")
	}
	if !strings.Contains(report.Blockers[3].Message, "Player Two") {
		t.Fatalf("player blocker message = %q, want participant name", report.Blockers[3].Message)
	}
	if strings.Contains(report.Blockers[3].Message, "player-2") {
		t.Fatalf("player blocker message = %q, did not expect participant id when name is present", report.Blockers[3].Message)
	}
}

func TestEvaluateSessionStartReport_UnknownCampaignStatusFailsClosed(t *testing.T) {
	report := EvaluateSessionStartReport(
		aggregate.State{
			Campaign: campaign.State{
				Status: campaign.Status("rolling"),
			},
		},
		ReportOptions{
			IncludeSessionBoundary: true,
		},
	)

	if report.Ready() {
		t.Fatal("expected non-ready report")
	}
	if len(report.Blockers) == 0 {
		t.Fatal("expected blockers")
	}
	first := report.Blockers[0]
	if first.Code != RejectionCodeSessionReadinessCampaignStatusDisallowsStart {
		t.Fatalf("first blocker code = %q, want %q", first.Code, RejectionCodeSessionReadinessCampaignStatusDisallowsStart)
	}
	if got := first.Metadata["status"]; got != "rolling" {
		t.Fatalf("status metadata = %q, want %q", got, "rolling")
	}
	if !strings.Contains(first.Message, "rolling") {
		t.Fatalf("status blocker message = %q, want to include unknown status", first.Message)
	}
}
