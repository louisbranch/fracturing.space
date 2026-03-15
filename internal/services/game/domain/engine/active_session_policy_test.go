package engine

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestActiveSessionPolicyForDefinition(t *testing.T) {
	tests := []struct {
		name string
		def  command.Definition
		cmd  command.Command
		want command.ActiveSessionClassification
		ok   bool
	}{
		{
			name: "blocked",
			def:  command.Definition{ActiveSession: command.BlockedDuringActiveSession()},
			want: command.ActiveSessionClassificationBlocked,
			ok:   true,
		},
		{
			name: "allowed",
			def:  command.Definition{ActiveSession: command.AllowedDuringActiveSession()},
			want: command.ActiveSessionClassificationAllowed,
			ok:   true,
		},
		{
			name: "character override allows in-game system actor",
			def:  command.Definition{ActiveSession: command.BlockedDuringActiveSessionExceptInGameSystemActor()},
			cmd:  command.Command{ActorType: command.ActorTypeSystem, SessionID: "sess-1"},
			want: command.ActiveSessionClassificationAllowed,
			ok:   true,
		},
		{
			name: "unspecified",
			def:  command.Definition{},
			ok:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ActiveSessionPolicyForDefinition(tc.def, tc.cmd)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("policy = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRejectActiveSessionBlockedCommand(t *testing.T) {
	decision, blocked := RejectActiveSessionBlockedCommand(
		session.State{Started: true, SessionID: "sess-1"},
		command.Command{Type: command.Type("campaign.update")},
		command.Definition{ActiveSession: command.BlockedDuringActiveSession()},
	)
	if !blocked {
		t.Fatal("expected command to be blocked")
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	rejection := decision.Rejections[0]
	if rejection.Code != RejectionCodeCampaignActiveSessionLocked {
		t.Fatalf("rejection code = %s, want %s", rejection.Code, RejectionCodeCampaignActiveSessionLocked)
	}
	if rejection.Message != "campaign has an active session: active_session_id=sess-1" {
		t.Fatalf("rejection message = %q", rejection.Message)
	}
}

func TestRejectActiveSessionBlockedCommand_AllowsWhenInactiveOrAllowed(t *testing.T) {
	if decision, blocked := RejectActiveSessionBlockedCommand(session.State{Started: false}, command.Command{Type: command.Type("campaign.update")}, command.Definition{ActiveSession: command.BlockedDuringActiveSession()}); blocked {
		t.Fatalf("unexpected blocked decision: %+v", decision)
	}
	if decision, blocked := RejectActiveSessionBlockedCommand(session.State{Started: true}, command.Command{Type: command.Type("action.roll.resolve")}, command.Definition{ActiveSession: command.AllowedDuringActiveSession()}); blocked {
		t.Fatalf("unexpected blocked decision: %+v", decision)
	}
}

func TestRejectActiveSessionBlockedCommand_WithoutSessionIDUsesGenericMessage(t *testing.T) {
	decision, blocked := RejectActiveSessionBlockedCommand(
		session.State{Started: true},
		command.Command{Type: command.Type("campaign.update")},
		command.Definition{ActiveSession: command.BlockedDuringActiveSession()},
	)
	if !blocked {
		t.Fatal("expected command to be blocked")
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Message != "campaign has an active session" {
		t.Fatalf("rejection message = %q, want generic active-session message", decision.Rejections[0].Message)
	}
}

func TestActiveSessionPolicyForDefinition_AllowsOnlyInGameCharacterMutations(t *testing.T) {
	tests := []struct {
		name string
		cmd  command.Command
		want command.ActiveSessionClassification
		ok   bool
	}{
		{
			name: "character update blocked by default",
			cmd:  command.Command{Type: command.Type("character.update"), ActorType: command.ActorTypeParticipant},
			want: command.ActiveSessionClassificationBlocked,
			ok:   true,
		},
		{
			name: "character delete allowed for system actor with session",
			cmd: command.Command{
				Type:      command.Type("character.delete"),
				ActorType: command.ActorTypeSystem,
				SessionID: "sess-1",
			},
			want: command.ActiveSessionClassificationAllowed,
			ok:   true,
		},
		{
			name: "character delete blocked for system actor without session",
			cmd: command.Command{
				Type:      command.Type("character.delete"),
				ActorType: command.ActorTypeSystem,
			},
			want: command.ActiveSessionClassificationBlocked,
			ok:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ActiveSessionPolicyForDefinition(command.Definition{ActiveSession: command.BlockedDuringActiveSessionExceptInGameSystemActor()}, tc.cmd)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("policy = %q, want %q", got, tc.want)
			}
		})
	}
}
