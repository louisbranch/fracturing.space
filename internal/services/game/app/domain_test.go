package server

import (
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
)

func TestCoreDeciderRoutesInviteCommands(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	decider := engine.CoreDecider{}
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
	decider := engine.CoreDecider{}

	decision := decider.Decide(aggregate.State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("action.roll.resolve"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"request_id":"req-1","roll_seq":1}`),
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
	decider := engine.CoreDecider{}
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

func TestNewCoreDecider_RejectsMissingCoreRoute(t *testing.T) {
	definitions := []command.Definition{
		{Type: command.Type("campaign.create"), Owner: command.OwnerCore},
		{Type: command.Type("story.scene.start"), Owner: command.OwnerCore},
	}

	_, err := engine.NewCoreDecider(nil, definitions)
	if err == nil {
		t.Fatal("expected error for missing core route")
	}
	if !strings.Contains(err.Error(), "story.scene.start") {
		t.Fatalf("expected missing command type in error, got %v", err)
	}
}

func TestNewCoreDecider_IncludesRegisteredCoreRoutes(t *testing.T) {
	registries, err := engine.BuildRegistries(registeredSystemModules()...)
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	definitions := registries.Commands.ListDefinitions()

	decider, err := engine.NewCoreDecider(registries.Systems, definitions)
	if err != nil {
		t.Fatalf("build core decider: %v", err)
	}

	// Verify the decider routes a known core command (it should not reject with
	// COMMAND_TYPE_UNSUPPORTED — domain-level rejections are acceptable since
	// we're only testing routing, not business logic).
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	decision := decider.Decide(aggregate.State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	}, func() time.Time { return now })
	for _, r := range decision.Rejections {
		if r.Code == "COMMAND_TYPE_UNSUPPORTED" {
			t.Fatalf("campaign.create should be routed, got COMMAND_TYPE_UNSUPPORTED")
		}
	}
}

func TestNewCoreDecider_RejectsStaleStaticRoute(t *testing.T) {
	// Pass only a subset of core routes — the missing ones should be detected.
	definitions := []command.Definition{
		{Type: command.Type("campaign.create"), Owner: command.OwnerCore},
	}

	_, err := engine.NewCoreDecider(nil, definitions)
	if err == nil {
		t.Fatal("expected error for stale static route")
	}
	if !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected 'stale' in error message, got: %v", err)
	}
}

func TestNewCoreDecider_StaticRoutesMatchCoreRegistry(t *testing.T) {
	registries, err := engine.BuildRegistries(registeredSystemModules()...)
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}

	// NewCoreDecider validates forward and reverse alignment between routes
	// and registered definitions — so just verifying it succeeds is enough.
	_, err = engine.NewCoreDecider(registries.Systems, registries.Commands.ListDefinitions())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
