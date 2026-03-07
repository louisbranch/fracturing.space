package engine

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestActiveSessionPolicyForCommandType(t *testing.T) {
	tests := []struct {
		name    string
		cmdType command.Type
		want    ActiveSessionCommandPolicy
		ok      bool
	}{
		{name: "campaign blocked", cmdType: command.Type("campaign.update"), want: ActiveSessionCommandPolicyBlocked, ok: true},
		{name: "participant blocked", cmdType: command.Type("participant.join"), want: ActiveSessionCommandPolicyBlocked, ok: true},
		{name: "seat blocked", cmdType: command.Type("seat.reassign"), want: ActiveSessionCommandPolicyBlocked, ok: true},
		{name: "invite blocked", cmdType: command.Type("invite.claim"), want: ActiveSessionCommandPolicyBlocked, ok: true},
		{name: "character blocked", cmdType: command.Type("character.update"), want: ActiveSessionCommandPolicyBlocked, ok: true},
		{name: "session allowed", cmdType: command.Type("session.end"), want: ActiveSessionCommandPolicyAllowed, ok: true},
		{name: "action allowed", cmdType: command.Type("action.outcome.apply"), want: ActiveSessionCommandPolicyAllowed, ok: true},
		{name: "story allowed", cmdType: command.Type("story.note.add"), want: ActiveSessionCommandPolicyAllowed, ok: true},
		{name: "system allowed", cmdType: command.Type("sys.daggerheart.damage.apply"), want: ActiveSessionCommandPolicyAllowed, ok: true},
		{name: "unknown family", cmdType: command.Type("custom.command"), ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ActiveSessionPolicyForCommandType(tc.cmdType)
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
	if decision, blocked := RejectActiveSessionBlockedCommand(session.State{Started: false}, command.Command{Type: command.Type("campaign.update")}); blocked {
		t.Fatalf("unexpected blocked decision: %+v", decision)
	}
	if decision, blocked := RejectActiveSessionBlockedCommand(session.State{Started: true}, command.Command{Type: command.Type("action.roll.resolve")}); blocked {
		t.Fatalf("unexpected blocked decision: %+v", decision)
	}
}

func TestActiveSessionPolicyForCommand_AllowsOnlyInGameCharacterMutations(t *testing.T) {
	tests := []struct {
		name string
		cmd  command.Command
		want ActiveSessionCommandPolicy
		ok   bool
	}{
		{
			name: "character update blocked by default",
			cmd:  command.Command{Type: command.Type("character.update"), ActorType: command.ActorTypeParticipant},
			want: ActiveSessionCommandPolicyBlocked,
			ok:   true,
		},
		{
			name: "character delete allowed for system actor with session",
			cmd: command.Command{
				Type:      command.Type("character.delete"),
				ActorType: command.ActorTypeSystem,
				SessionID: "sess-1",
			},
			want: ActiveSessionCommandPolicyAllowed,
			ok:   true,
		},
		{
			name: "character delete blocked for system actor without session",
			cmd: command.Command{
				Type:      command.Type("character.delete"),
				ActorType: command.ActorTypeSystem,
			},
			want: ActiveSessionCommandPolicyBlocked,
			ok:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ActiveSessionPolicyForCommand(tc.cmd)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("policy = %q, want %q", got, tc.want)
			}
		})
	}
}
