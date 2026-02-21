package server

import (
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
)

func TestCoreDeciderRoutesInviteCommands(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	decider := coreDecider{}
	state := aggregate.State{
		Invites: map[string]invite.State{
			"inv-1": {Created: true, Status: "pending"},
		},
	}

	decision := decider.Decide(state, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("invite.claim"),
		ActorType:   command.ActorTypeSystem,
		EntityType:  "invite",
		EntityID:    "inv-1",
		PayloadJSON: []byte(`{"invite_id":"inv-1","participant_id":"p-1","user_id":"user-1","jti":"jwt-1"}`),
	}, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	if decision.Events[0].Type != event.Type("invite.claimed") {
		t.Fatalf("event type = %s, want %s", decision.Events[0].Type, "invite.claimed")
	}
}

func TestCoreDeciderRoutesActionCommands(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	decider := coreDecider{}

	decision := decider.Decide(aggregate.State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("action.roll.resolve"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"request_id":"req-1"}`),
	}, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	if decision.Events[0].Type != event.Type("action.roll_resolved") {
		t.Fatalf("event type = %s, want %s", decision.Events[0].Type, "action.roll_resolved")
	}
}

func TestCoreDeciderRejectsUnsupportedCommandType(t *testing.T) {
	decider := coreDecider{}
	decision := decider.Decide(aggregate.State{}, command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("story.scene.start"),
		ActorType:  command.ActorTypeSystem,
	}, time.Now)

	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "COMMAND_TYPE_UNSUPPORTED" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "COMMAND_TYPE_UNSUPPORTED")
	}
}

func TestBuildCoreRouteTable_RejectsMissingCoreRoute(t *testing.T) {
	definitions := []command.Definition{
		{Type: command.Type("campaign.create"), Owner: command.OwnerCore},
		{Type: command.Type("story.scene.start"), Owner: command.OwnerCore},
	}

	_, err := buildCoreRouteTable(definitions)
	if err == nil {
		t.Fatal("expected error for missing core route")
	}
	if !strings.Contains(err.Error(), "story.scene.start") {
		t.Fatalf("expected missing command type in error, got %v", err)
	}
}

func TestBuildCoreRouteTable_IncludesRegisteredCoreRoutes(t *testing.T) {
	definitions := []command.Definition{
		{Type: command.Type("campaign.create"), Owner: command.OwnerCore},
		{Type: command.Type("action.roll.resolve"), Owner: command.OwnerCore},
		{Type: command.Type("participant.seat.reassign"), Owner: command.OwnerCore},
		{Type: command.Type("seat.reassign"), Owner: command.OwnerCore},
		{Type: command.Type("sys.alpha.action.attack.resolve"), Owner: command.OwnerSystem},
	}

	routes, err := buildCoreRouteTable(definitions)
	if err != nil {
		t.Fatalf("build core route table: %v", err)
	}
	if _, ok := routes[command.Type("campaign.create")]; !ok {
		t.Fatal("expected campaign.create route")
	}
	if _, ok := routes[command.Type("action.roll.resolve")]; !ok {
		t.Fatal("expected action.roll.resolve route")
	}
	if _, ok := routes[command.Type("participant.seat.reassign")]; !ok {
		t.Fatal("expected participant.seat.reassign route")
	}
	if _, ok := routes[command.Type("seat.reassign")]; !ok {
		t.Fatal("expected seat.reassign route")
	}
	if _, ok := routes[command.Type("sys.alpha.action.attack.resolve")]; ok {
		t.Fatal("did not expect system command route in core table")
	}
}
