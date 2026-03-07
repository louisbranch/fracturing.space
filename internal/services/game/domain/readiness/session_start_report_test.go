package readiness

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestEvaluateSessionStartReport_IncludesBoundaryAndCoreBlockersInOrder(t *testing.T) {
	report := EvaluateSessionStartReport(
		aggregate.State{
			Campaign: campaign.State{
				Status: campaign.StatusCompleted,
				GmMode: string(campaign.GmModeAI),
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
			Participants: map[string]participant.State{
				"gm-1": {
					ParticipantID: "gm-1",
					Joined:        true,
					Role:          string(participant.RoleGM),
				},
				"player-1": {
					ParticipantID: "player-1",
					Joined:        true,
					Role:          string(participant.RolePlayer),
				},
				"player-2": {
					ParticipantID: "player-2",
					Joined:        true,
					Role:          string(participant.RolePlayer),
				},
			},
			Characters: map[string]character.State{
				"char-b": {
					CharacterID:   "char-b",
					Created:       true,
					ParticipantID: "player-1",
				},
				"char-a": {
					CharacterID: "char-a",
					Created:     true,
				},
			},
		},
		ReportOptions{
			SystemReadiness: func(map[string]any) (bool, string) {
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
	if report.Blockers[3].Metadata["participant_id"] != "player-2" {
		t.Fatalf("player blocker participant_id = %q, want %q", report.Blockers[3].Metadata["participant_id"], "player-2")
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
