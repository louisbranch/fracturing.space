package engine

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestNewCoreDecider_BuildsRoutesForRegisteredCoreDefinitions(t *testing.T) {
	definitions := allCoreCommandDefinitions()

	decider, err := NewCoreDecider(nil, definitions)
	if err != nil {
		t.Fatalf("NewCoreDecider returned error: %v", err)
	}
	if len(decider.coreCommands.routes) != len(definitions) {
		t.Fatalf("routes = %d, want %d", len(decider.coreCommands.routes), len(definitions))
	}
	if decider.coreCommands.sessionStart == nil {
		t.Fatal("expected session-start workflow to be initialized")
	}
}

func TestIndexCommandDefinitions_HandlesEmptyAndIndexesByType(t *testing.T) {
	if got := indexCommandDefinitions(nil); got != nil {
		t.Fatalf("indexCommandDefinitions(nil) = %#v, want nil", got)
	}

	definitions := []command.Definition{
		{Type: campaign.CommandTypeCreate},
		{Type: session.CommandTypeStart},
	}
	got := indexCommandDefinitions(definitions)
	if len(got) != len(definitions) {
		t.Fatalf("indexed definitions = %d, want %d", len(got), len(definitions))
	}
	if got[campaign.CommandTypeCreate].Type != campaign.CommandTypeCreate {
		t.Fatalf("campaign definition missing from index: %#v", got[campaign.CommandTypeCreate])
	}
	if got[session.CommandTypeStart].Type != session.CommandTypeStart {
		t.Fatalf("session definition missing from index: %#v", got[session.CommandTypeStart])
	}
}

func TestBuildCoreRouteTable_RejectsMissingRouteForRegisteredCoreType(t *testing.T) {
	definitions := allCoreCommandDefinitions()
	definitions = append(definitions, command.Definition{
		Type:  command.Type("core.unknown.route"),
		Owner: command.OwnerCore,
	})

	_, err := buildCoreRouteTable(definitions)
	if err == nil {
		t.Fatal("expected route validation error")
	}
	if !strings.Contains(err.Error(), "core command route missing for registered type core.unknown.route") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildCoreRouteTable_RejectsStaleStaticRoutesWithoutRegistration(t *testing.T) {
	definitions := allCoreCommandDefinitions()
	removedType := definitions[0].Type
	definitions = definitions[1:]

	_, err := buildCoreRouteTable(definitions)
	if err == nil {
		t.Fatal("expected stale route error")
	}
	if !strings.Contains(err.Error(), "stale static core command routes without registration") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), string(removedType)) {
		t.Fatalf("error = %v, want removed type %s included", err, removedType)
	}
}

func TestCoreDeciderDecide_RejectsUnsupportedCoreCommandType(t *testing.T) {
	decision := CoreDecider{}.Decide(aggregate.State{}, command.Command{Type: command.Type("core.unknown")}, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "COMMAND_TYPE_UNSUPPORTED" {
		t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, "COMMAND_TYPE_UNSUPPORTED")
	}
}

func TestCoreDeciderDecide_RejectsSystemCommandWhenSystemRegistryMissing(t *testing.T) {
	decision := CoreDecider{}.Decide(aggregate.State{}, command.Command{
		Type:          command.Type("sys.stub.action"),
		SystemID:      "stub",
		SystemVersion: "v1",
	}, time.Now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "SYSTEM_COMMAND_REJECTED" {
		t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, "SYSTEM_COMMAND_REJECTED")
	}
}

func TestCoreDeciderDecide_UsesInjectedRouteTable(t *testing.T) {
	customType := command.Type("custom.route")
	called := false

	decision := CoreDecider{
		coreCommands: coreCommandRouter{
			routes: map[command.Type]coreCommandRoute{
				customType: func(_ coreCommandRouter, _ aggregate.State, _ command.Command, _ func() time.Time) command.Decision {
					called = true
					return command.Accept(event.Event{Type: event.Type("custom.routed")})
				},
			},
		},
	}.Decide(&aggregate.State{Campaign: campaign.State{Created: true}}, command.Command{Type: customType}, time.Now)

	if !called {
		t.Fatal("expected custom route to be invoked")
	}
	if len(decision.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(decision.Events))
	}
	if decision.Events[0].Type != event.Type("custom.routed") {
		t.Fatalf("event type = %q, want %q", decision.Events[0].Type, event.Type("custom.routed"))
	}
}

type stubSessionStartWorkflow struct {
	decision command.Decision
	called   bool
}

func (s *stubSessionStartWorkflow) Start(_ aggregate.State, _ command.Command, _ func() time.Time) command.Decision {
	s.called = true
	return s.decision
}

var _ readiness.SessionStartWorkflow = (*stubSessionStartWorkflow)(nil)

func TestSessionStartRoute_UsesInjectedSessionStartWorkflow(t *testing.T) {
	workflow := &stubSessionStartWorkflow{
		decision: command.Accept(event.Event{Type: event.Type("session.started")}),
	}

	decision := sessionStartRoute(
		coreCommandRouter{sessionStart: workflow},
		aggregate.State{},
		command.Command{Type: session.CommandTypeStart},
		time.Now,
	)

	if !workflow.called {
		t.Fatal("expected injected session-start workflow to be used")
	}
	if len(decision.Events) != 1 || decision.Events[0].Type != session.EventTypeStarted {
		t.Fatalf("events = %v, want one %s event", decision.Events, session.EventTypeStarted)
	}
}

func TestAggregateState_ConvertsSupportedInputs(t *testing.T) {
	value := aggregate.State{Campaign: campaign.State{Created: true}}
	if got := aggregateState(value); !got.Campaign.Created {
		t.Fatal("expected aggregateState to return value input")
	}

	if got := aggregateState(&value); !got.Campaign.Created {
		t.Fatal("expected aggregateState to dereference pointer input")
	}

	var nilPointer *aggregate.State
	if got := aggregateState(nilPointer); !reflect.DeepEqual(got, aggregate.State{}) {
		t.Fatalf("aggregateState(nil pointer) = %+v, want zero state", got)
	}

	if got := aggregateState(42); !reflect.DeepEqual(got, aggregate.State{}) {
		t.Fatalf("aggregateState(invalid input) = %+v, want zero state", got)
	}
}

func TestCoreRouteWrappers_DelegateToDomainDeciders(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) }
	state := aggregate.State{
		Participants: map[ids.ParticipantID]participant.State{
			"part-1": {ParticipantID: "part-1", Joined: true},
		},
		Characters: map[ids.CharacterID]character.State{
			"char-1": {CharacterID: "char-1", Created: true},
		},
	}

	tests := []struct {
		name  string
		route coreCommandRoute
		cmd   command.Command
	}{
		{
			name:  "campaign route",
			route: campaignRoute,
			cmd:   command.Command{Type: command.Type("campaign.unknown"), PayloadJSON: []byte(`{}`)},
		},
		{
			name:  "action route",
			route: actionRoute,
			cmd:   command.Command{Type: command.Type("action.unknown"), PayloadJSON: []byte(`{}`)},
		},
		{
			name:  "session route",
			route: sessionRoute,
			cmd:   command.Command{Type: command.Type("session.unknown"), PayloadJSON: []byte(`{}`)},
		},
		{
			name:  "scene route",
			route: sceneRoute,
			cmd:   command.Command{Type: command.Type("scene.unknown"), PayloadJSON: []byte(`{}`)},
		},
		{
			name:  "participant route",
			route: participantRoute,
			cmd: command.Command{
				Type:        command.Type("participant.unknown"),
				EntityID:    "part-1",
				PayloadJSON: []byte(`{}`),
			},
		},
		{
			name:  "character route",
			route: characterRoute,
			cmd: command.Command{
				Type:        command.Type("character.unknown"),
				EntityID:    "char-1",
				PayloadJSON: []byte(`{}`),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := tc.route(coreCommandRouter{}, state, tc.cmd, now)
			if len(decision.Rejections) != 1 {
				t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
			}
			if decision.Rejections[0].Code != "COMMAND_TYPE_UNSUPPORTED" {
				t.Fatalf("rejection code = %q, want COMMAND_TYPE_UNSUPPORTED", decision.Rejections[0].Code)
			}
		})
	}
}

func TestParticipantStateFor_UsesNormalizedEntityID(t *testing.T) {
	current := aggregate.State{
		Participants: map[ids.ParticipantID]participant.State{
			"part-1": {ParticipantID: "part-1", Joined: true},
		},
	}

	if got := participantStateFor(command.Command{EntityID: " part-1 "}, current); got.ParticipantID != "part-1" {
		t.Fatalf("participantStateFor(entity id) = %+v, want participant_id part-1", got)
	}
}

func TestCharacterStateFor_UsesNormalizedEntityID(t *testing.T) {
	current := aggregate.State{
		Characters: map[ids.CharacterID]character.State{
			"char-1": {CharacterID: "char-1", Created: true},
		},
	}

	if got := characterStateFor(command.Command{EntityID: " char-1 "}, current); got.CharacterID != "char-1" {
		t.Fatalf("characterStateFor(entity id) = %+v, want character_id char-1", got)
	}
}

func allCoreCommandDefinitions() []command.Definition {
	routes := staticCoreCommandRoutes()
	definitions := make([]command.Definition, 0, len(routes))
	for cmdType := range routes {
		definitions = append(definitions, command.Definition{
			Type:  cmdType,
			Owner: command.OwnerCore,
		})
	}
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Type < definitions[j].Type
	})
	return definitions
}
