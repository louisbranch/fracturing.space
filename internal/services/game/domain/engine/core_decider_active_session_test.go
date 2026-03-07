package engine

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestCoreDeciderDecide_BlocksOutOfGameCommandWhenSessionActive(t *testing.T) {
	decider := CoreDecider{
		routes: map[command.Type]coreCommandRoute{
			command.Type("campaign.update"): campaignRoute,
		},
	}
	decision := decider.Decide(
		aggregate.State{
			Session: session.State{Started: true, SessionID: "sess-1"},
		},
		command.Command{Type: command.Type("campaign.update")},
		nil,
	)
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != RejectionCodeCampaignActiveSessionLocked {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, RejectionCodeCampaignActiveSessionLocked)
	}
}

func TestCoreDeciderDecide_AllowsSessionScopedFamiliesWhenSessionActive(t *testing.T) {
	decider := CoreDecider{
		routes: map[command.Type]coreCommandRoute{
			command.Type("session.end"): sessionRoute,
		},
	}
	decision := decider.Decide(
		aggregate.State{
			Session: sessionStateStarted("sess-1"),
		},
		command.Command{Type: command.Type("session.end"), PayloadJSON: []byte(`{}`)},
		nil,
	)
	if len(decision.Rejections) == 0 {
		t.Fatal("expected route-level rejection from session decider")
	}
	if decision.Rejections[0].Code == RejectionCodeCampaignActiveSessionLocked {
		t.Fatal("expected session route to execute instead of active-session lock")
	}
}

func sessionStateStarted(sessionID string) session.State {
	return session.State{
		Started:   true,
		SessionID: sessionID,
	}
}

func TestCoreDeciderDecide_BlockedMessageIncludesSessionID(t *testing.T) {
	decider := CoreDecider{
		routes: map[command.Type]coreCommandRoute{
			command.Type("campaign.update"): campaignRoute,
		},
	}
	decision := decider.Decide(
		aggregate.State{
			Campaign: campaign.State{Created: true},
			Session:  sessionStateStarted("sess-123"),
		},
		command.Command{
			Type:        command.Type("campaign.update"),
			PayloadJSON: []byte(`{"fields":{"name":"x"}}`),
		},
		nil,
	)
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Message != "campaign has an active session: active_session_id=sess-123" {
		t.Fatalf("rejection message = %q", decision.Rejections[0].Message)
	}
}

func TestCoreDeciderDecide_AllowsInGameSystemCharacterCommandWhenSessionActive(t *testing.T) {
	decider := CoreDecider{
		routes: map[command.Type]coreCommandRoute{
			command.Type("character.delete"): func(_ CoreDecider, _ aggregate.State, _ command.Command, _ func() time.Time) command.Decision {
				return command.Accept()
			},
		},
	}
	decision := decider.Decide(
		aggregate.State{Session: sessionStateStarted("sess-1")},
		command.Command{
			Type:      command.Type("character.delete"),
			ActorType: command.ActorTypeSystem,
			SessionID: "sess-1",
		},
		nil,
	)
	if len(decision.Rejections) > 0 {
		t.Fatalf("rejections = %v, want none", decision.Rejections)
	}
}
