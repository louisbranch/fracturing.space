package readiness

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

func TestActionActionable(t *testing.T) {
	t.Parallel()

	if (Action{}).Actionable() {
		t.Fatal("empty action should not be actionable")
	}
	if (Action{ResolutionKind: ResolutionKindCreateCharacter}).Actionable() {
		t.Fatal("action without responsible principals should not be actionable")
	}
	if !(Action{
		ResponsibleUserIDs: []string{"user-1"},
		ResolutionKind:     ResolutionKindCreateCharacter,
	}).Actionable() {
		t.Fatal("action with resolution and responsible user should be actionable")
	}
}

func TestCloneActionNormalizesIDsAndTargets(t *testing.T) {
	t.Parallel()

	cloned := cloneAction(Action{
		ResponsibleUserIDs:        []string{" user-1 ", "user-1", "", "user-2"},
		ResponsibleParticipantIDs: []string{" part-1 ", "part-1", "", "part-2"},
		ResolutionKind:            ResolutionKindManageParticipants,
		TargetParticipantID:       " part-1 ",
		TargetCharacterID:         " char-1 ",
	})
	assertStringSliceEqual(t, cloned.ResponsibleUserIDs, []string{"user-1", "user-2"})
	assertStringSliceEqual(t, cloned.ResponsibleParticipantIDs, []string{"part-1", "part-2"})
	if cloned.TargetParticipantID != "part-1" {
		t.Fatalf("target participant id = %q, want %q", cloned.TargetParticipantID, "part-1")
	}
	if cloned.TargetCharacterID != "char-1" {
		t.Fatalf("target character id = %q, want %q", cloned.TargetCharacterID, "char-1")
	}
}

func TestNormalizeActionIDsReturnsNilWhenEmpty(t *testing.T) {
	t.Parallel()

	if got := normalizeActionIDs([]string{" ", ""}); got != nil {
		t.Fatalf("normalizeActionIDs() = %v, want nil", got)
	}
}

func TestActionHelpersBuildSelfServiceTargets(t *testing.T) {
	t.Parallel()

	index := participantIndex{
		byID: map[string]participant.State{
			"gm-owner": {
				ParticipantID:  "gm-owner",
				UserID:         ids.UserID("user-gm-owner"),
				Role:           participant.RoleGM,
				Controller:     participant.ControllerHuman,
				CampaignAccess: participant.CampaignAccessOwner,
			},
			"gm-ai": {
				ParticipantID:  "gm-ai",
				Role:           participant.RoleGM,
				Controller:     participant.ControllerAI,
				CampaignAccess: participant.CampaignAccessOwner,
			},
			"gm-member": {
				ParticipantID:  "gm-member",
				UserID:         ids.UserID("user-gm-member"),
				Role:           participant.RoleGM,
				Controller:     participant.ControllerHuman,
				CampaignAccess: participant.CampaignAccessMember,
			},
			"player-1": {
				ParticipantID:  "player-1",
				UserID:         ids.UserID("user-player-1"),
				Role:           participant.RolePlayer,
				Controller:     participant.ControllerHuman,
				CampaignAccess: participant.CampaignAccessMember,
			},
			"player-no-user": {
				ParticipantID:  "player-no-user",
				Role:           participant.RolePlayer,
				Controller:     participant.ControllerHuman,
				CampaignAccess: participant.CampaignAccessMember,
			},
		},
		gmIDs:     []string{"gm-owner", "gm-ai", "gm-member"},
		aiGMIDs:   []string{"gm-ai"},
		playerIDs: []string{"player-1", "player-no-user"},
	}

	aiAgent := aiAgentRequiredAction(index)
	if aiAgent.ResolutionKind != ResolutionKindConfigureAIAgent {
		t.Fatalf("aiAgentRequiredAction resolution = %q, want %q", aiAgent.ResolutionKind, ResolutionKindConfigureAIAgent)
	}
	assertStringSliceEqual(t, aiAgent.ResponsibleUserIDs, []string{"user-gm-owner"})
	assertStringSliceEqual(t, aiAgent.ResponsibleParticipantIDs, []string{"gm-owner"})
	if aiAgent.TargetParticipantID != "gm-ai" {
		t.Fatalf("aiAgentRequiredAction target participant = %q, want %q", aiAgent.TargetParticipantID, "gm-ai")
	}

	invite := invitePlayerAction(index)
	if invite.ResolutionKind != ResolutionKindInvitePlayer {
		t.Fatalf("invitePlayerAction resolution = %q, want %q", invite.ResolutionKind, ResolutionKindInvitePlayer)
	}
	assertStringSliceEqual(t, invite.ResponsibleUserIDs, []string{"user-gm-owner", "user-gm-member"})
	assertStringSliceEqual(t, invite.ResponsibleParticipantIDs, []string{"gm-owner", "gm-member"})

	create := createCharacterAction(index, "player-1")
	if create.ResolutionKind != ResolutionKindCreateCharacter {
		t.Fatalf("createCharacterAction resolution = %q, want %q", create.ResolutionKind, ResolutionKindCreateCharacter)
	}
	assertStringSliceEqual(t, create.ResponsibleUserIDs, []string{"user-player-1"})
	assertStringSliceEqual(t, create.ResponsibleParticipantIDs, []string{"player-1"})
	if create.TargetParticipantID != "player-1" {
		t.Fatalf("createCharacterAction target participant = %q, want %q", create.TargetParticipantID, "player-1")
	}

	complete := completeCharacterAction(index, "player-1", "char-1")
	if complete.ResolutionKind != ResolutionKindCompleteCharacter {
		t.Fatalf("completeCharacterAction resolution = %q, want %q", complete.ResolutionKind, ResolutionKindCompleteCharacter)
	}
	if complete.TargetParticipantID != "player-1" {
		t.Fatalf("completeCharacterAction target participant = %q, want %q", complete.TargetParticipantID, "player-1")
	}
	if complete.TargetCharacterID != "char-1" {
		t.Fatalf("completeCharacterAction target character = %q, want %q", complete.TargetCharacterID, "char-1")
	}
}

func TestActionHelpersReturnEmptyWhenParticipantCannotSelfServe(t *testing.T) {
	t.Parallel()

	index := participantIndex{
		byID: map[string]participant.State{
			"player-no-user": {
				ParticipantID: "player-no-user",
				Role:          participant.RolePlayer,
				Controller:    participant.ControllerHuman,
			},
		},
	}

	assertZeroAction(t, createCharacterAction(index, "missing"), "createCharacterAction(missing)")
	assertZeroAction(t, createCharacterAction(index, "player-no-user"), "createCharacterAction(no-user)")
	assertZeroAction(t, completeCharacterAction(index, "missing", "char-1"), "completeCharacterAction(missing)")
	assertZeroAction(t, completeCharacterAction(index, "player-no-user", "char-1"), "completeCharacterAction(no-user)")
}

func TestCampaignStatusAllowsSessionStart(t *testing.T) {
	t.Parallel()

	if !campaignStatusAllowsSessionStart("draft") {
		t.Fatal("draft should allow session start")
	}
	if !campaignStatusAllowsSessionStart("active") {
		t.Fatal("active should allow session start")
	}
	if campaignStatusAllowsSessionStart("completed") {
		t.Fatal("completed should not allow session start")
	}
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice len = %d, want %d; got=%v want=%v", len(got), len(want), got, want)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("slice[%d] = %q, want %q; got=%v want=%v", idx, got[idx], want[idx], got, want)
		}
	}
}

func assertZeroAction(t *testing.T, got Action, label string) {
	t.Helper()
	if got.ResolutionKind != ResolutionKindUnspecified {
		t.Fatalf("%s resolution = %q, want empty", label, got.ResolutionKind)
	}
	if len(got.ResponsibleUserIDs) != 0 || len(got.ResponsibleParticipantIDs) != 0 {
		t.Fatalf("%s responsible ids = users:%v participants:%v, want none", label, got.ResponsibleUserIDs, got.ResponsibleParticipantIDs)
	}
	if got.TargetParticipantID != "" || got.TargetCharacterID != "" {
		t.Fatalf("%s targets = participant:%q character:%q, want empty", label, got.TargetParticipantID, got.TargetCharacterID)
	}
}
