package engine

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestNewCoreDecider_ReturnsRouteTableError(t *testing.T) {
	_, err := NewCoreDecider(nil, []command.Definition{
		{Type: command.Type("core.missing.route"), Owner: command.OwnerCore},
	})
	if err == nil {
		t.Fatal("expected route-table build error")
	}
	if !strings.Contains(err.Error(), "core command route missing for registered type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildCoreRouteTable_IgnoresNonCoreDefinitions(t *testing.T) {
	definitions := append(
		allCoreCommandDefinitions(),
		command.Definition{Type: command.Type("sys.stub.cmd"), Owner: command.OwnerSystem},
	)
	routes, err := buildCoreRouteTable(definitions)
	if err != nil {
		t.Fatalf("buildCoreRouteTable() error = %v", err)
	}
	if len(routes) != len(allCoreCommandDefinitions()) {
		t.Fatalf("routes len = %d, want %d", len(routes), len(allCoreCommandDefinitions()))
	}
}

func TestCoreDeciderDecide_RoutesSystemCommandOnSuccess(t *testing.T) {
	systemRegistry := module.NewRegistry()
	if err := systemRegistry.Register(acceptingSystemModule{}); err != nil {
		t.Fatalf("register system module: %v", err)
	}

	decision := CoreDecider{systemCommands: newSystemCommandDispatcher(systemRegistry)}.Decide(
		aggregate.State{},
		command.Command{
			CampaignID:    "camp-1",
			Type:          command.Type("sys.stub.command"),
			SystemID:      "stub",
			SystemVersion: "v1",
		},
		time.Now,
	)
	if len(decision.Rejections) != 0 {
		t.Fatalf("rejections = %d, want 0 (%v)", len(decision.Rejections), decision.Rejections)
	}
	if len(decision.Events) != 1 || decision.Events[0].Type != event.Type("sys.stub.event") {
		t.Fatalf("events = %v, want one sys.stub.event", decision.Events)
	}
}

func TestSessionStartRoute_ReturnsSessionStartRejectionImmediately(t *testing.T) {
	decision := sessionStartRoute(
		coreCommandRouter{},
		readySessionStartAggregateState(campaign.StatusDraft),
		command.Command{
			CampaignID:  "camp-1",
			Type:        session.CommandTypeStart,
			ActorType:   command.ActorTypeParticipant,
			ActorID:     "gm-1",
			PayloadJSON: []byte(`{`),
		},
		func() time.Time { return time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC) },
	)
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if len(decision.Events) != 0 {
		t.Fatalf("events = %d, want 0 when session start rejects", len(decision.Events))
	}
}

func TestSessionStartRoute_BlankCampaignStatusFailsClosed(t *testing.T) {
	decision := sessionStartRoute(
		coreCommandRouter{},
		readySessionStartAggregateState(""),
		command.Command{
			CampaignID:  "camp-1",
			Type:        session.CommandTypeStart,
			ActorType:   command.ActorTypeParticipant,
			ActorID:     "gm-1",
			PayloadJSON: []byte(`{"session_id":"sess-1","session_name":"First Session"}`),
		},
		func() time.Time { return time.Date(2026, 3, 1, 13, 5, 0, 0, time.UTC) },
	)
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != readiness.RejectionCodeSessionReadinessCampaignStatusDisallowsStart {
		t.Fatalf(
			"rejection code = %s, want %s",
			decision.Rejections[0].Code,
			readiness.RejectionCodeSessionReadinessCampaignStatusDisallowsStart,
		)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("events = %d, want 0 when readiness rejects", len(decision.Events))
	}
}

func TestSessionStartRoute_ActiveSessionRejectedByReadiness(t *testing.T) {
	state := readySessionStartAggregateState(campaign.StatusActive)
	state.Session.Started = true

	decision := sessionStartRoute(
		coreCommandRouter{},
		state,
		command.Command{
			CampaignID:  "camp-1",
			Type:        session.CommandTypeStart,
			ActorType:   command.ActorTypeParticipant,
			ActorID:     "gm-1",
			PayloadJSON: []byte(`{"session_id":"sess-1","session_name":"First Session"}`),
		},
		func() time.Time { return time.Date(2026, 3, 1, 13, 7, 0, 0, time.UTC) },
	)

	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != readiness.RejectionCodeSessionReadinessActiveSessionExists {
		t.Fatalf(
			"rejection code = %s, want %s",
			decision.Rejections[0].Code,
			readiness.RejectionCodeSessionReadinessActiveSessionExists,
		)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("events = %d, want 0 when readiness rejects", len(decision.Events))
	}
}

func TestSessionStartRoute_NonDraftCampaignReturnsSessionStartOnly(t *testing.T) {
	decision := sessionStartRoute(
		coreCommandRouter{},
		readySessionStartAggregateState(campaign.StatusActive),
		command.Command{
			CampaignID:  "camp-1",
			Type:        session.CommandTypeStart,
			ActorType:   command.ActorTypeParticipant,
			ActorID:     "gm-1",
			PayloadJSON: []byte(`{"session_id":"sess-1","session_name":"First Session"}`),
		},
		func() time.Time { return time.Date(2026, 3, 1, 13, 10, 0, 0, time.UTC) },
	)
	if len(decision.Rejections) != 0 {
		t.Fatalf("rejections = %d, want 0", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("events = %d, want 1 for non-draft campaign", len(decision.Events))
	}
	if decision.Events[0].Type != session.EventTypeStarted {
		t.Fatalf("event[0].Type = %s, want %s", decision.Events[0].Type, session.EventTypeStarted)
	}
}

func TestStateResolvers_ReturnZeroWhenAggregateMapsMissing(t *testing.T) {
	state := aggregate.State{}
	if got := participantStateFor(command.Command{EntityID: "p-1"}, state); !reflect.DeepEqual(got, participant.State{}) {
		t.Fatalf("participantStateFor() = %+v, want zero state", got)
	}
	if got := characterStateFor(command.Command{EntityID: "c-1"}, state); !reflect.DeepEqual(got, character.State{}) {
		t.Fatalf("characterStateFor() = %+v, want zero state", got)
	}
	if got := inviteStateFor(command.Command{EntityID: "i-1"}, state); !reflect.DeepEqual(got, invite.State{}) {
		t.Fatalf("inviteStateFor() = %+v, want zero state", got)
	}
}

func TestStateResolvers_ReturnZeroWhenEntityIDIsEmpty(t *testing.T) {
	state := aggregate.State{
		Participants: map[ids.ParticipantID]participant.State{
			"p-1": {ParticipantID: "p-1", Joined: true},
		},
		Characters: map[ids.CharacterID]character.State{
			"c-1": {CharacterID: "c-1", Created: true},
		},
		Invites: map[ids.InviteID]invite.State{
			"i-1": {InviteID: "i-1", Created: true},
		},
	}

	if got := participantStateFor(command.Command{EntityID: "   "}, state); !reflect.DeepEqual(got, participant.State{}) {
		t.Fatalf("participantStateFor() = %+v, want zero state", got)
	}
	if got := characterStateFor(command.Command{EntityID: "   "}, state); !reflect.DeepEqual(got, character.State{}) {
		t.Fatalf("characterStateFor() = %+v, want zero state", got)
	}
	if got := inviteStateFor(command.Command{EntityID: "   "}, state); !reflect.DeepEqual(got, invite.State{}) {
		t.Fatalf("inviteStateFor() = %+v, want zero state", got)
	}
}

func readySessionStartAggregateState(status campaign.Status) aggregate.State {
	return aggregate.State{
		Campaign: campaign.State{
			Created: true,
			Status:  status,
		},
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
		},
		Characters: map[ids.CharacterID]character.State{
			"char-1": {
				CharacterID:   "char-1",
				Created:       true,
				ParticipantID: "player-1",
			},
		},
	}
}

type acceptingSystemModule struct{}

func (acceptingSystemModule) ID() string      { return "stub" }
func (acceptingSystemModule) Version() string { return "v1" }
func (acceptingSystemModule) RegisterCommands(*command.Registry) error {
	return nil
}
func (acceptingSystemModule) RegisterEvents(*event.Registry) error {
	return nil
}
func (acceptingSystemModule) EmittableEventTypes() []event.Type { return nil }
func (acceptingSystemModule) Decider() module.Decider           { return acceptingSystemDecider{} }
func (acceptingSystemModule) Folder() module.Folder             { return nil }
func (acceptingSystemModule) StateFactory() module.StateFactory { return nil }
func (acceptingSystemModule) CharacterReady(any, character.State) (bool, string) {
	return true, ""
}

type acceptingSystemDecider struct{}

func (acceptingSystemDecider) Decide(_ any, cmd command.Command, now func() time.Time) command.Decision {
	eventTime := time.Unix(0, 0).UTC()
	if now != nil {
		eventTime = now().UTC()
	}
	return command.Accept(command.NewEvent(
		cmd,
		event.Type("sys.stub.event"),
		"system_entity",
		"stub-1",
		[]byte(`{"ok":true}`),
		eventTime,
	))
}
