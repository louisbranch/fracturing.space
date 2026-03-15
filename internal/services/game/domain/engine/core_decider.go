package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaignbootstrap"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// CoreDecider is the top-level decider for core (non-system) commands.
//
// It keeps command routing explicit: each command type maps to exactly one
// aggregate route, while system commands are dispatched by system id/version.
type CoreDecider struct {
	Systems              *module.Registry
	SessionStartWorkflow readiness.SessionStartWorkflow
	definitions          map[command.Type]command.Definition
	routes               map[command.Type]coreCommandRoute
}

// coreCommandRoute maps a normalized aggregate state + command into one decision path.
type coreCommandRoute func(d CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision

// NewCoreDecider builds a CoreDecider with validated routes derived from
// the registered command definitions.
func NewCoreDecider(systems *module.Registry, definitions []command.Definition) (CoreDecider, error) {
	routes, err := buildCoreRouteTable(definitions)
	if err != nil {
		return CoreDecider{}, err
	}
	return CoreDecider{
		Systems:              systems,
		SessionStartWorkflow: readiness.NewSessionStartWorkflow(systems),
		definitions:          indexCommandDefinitions(definitions),
		routes:               routes,
	}, nil
}

func (d CoreDecider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
	current := aggregateState(state)
	if strings.TrimSpace(cmd.SystemID) != "" || strings.TrimSpace(cmd.SystemVersion) != "" {
		key := module.Key{ID: cmd.SystemID, Version: cmd.SystemVersion}
		systemState := current.Systems[key]
		decision, err := module.RouteCommand(d.Systems, systemState, cmd, now)
		if err != nil {
			return command.Reject(command.Rejection{Code: "SYSTEM_COMMAND_REJECTED", Message: err.Error()})
		}
		return decision
	}
	if decision, blocked := RejectActiveSessionBlockedCommand(current.Session, cmd, d.definitionFor(cmd.Type)); blocked {
		return decision
	}
	routes := d.routes
	if routes == nil {
		routes = staticCoreCommandRoutes()
	}
	route, ok := routes[cmd.Type]
	if !ok {
		return command.Reject(command.Rejection{
			Code:    "COMMAND_TYPE_UNSUPPORTED",
			Message: "command type is not supported by core decider",
		})
	}
	return route(d, current, cmd, now)
}

func (d CoreDecider) definitionFor(cmdType command.Type) command.Definition {
	if definition, ok := d.definitions[cmdType]; ok {
		return definition
	}
	return command.Definition{}
}

func indexCommandDefinitions(definitions []command.Definition) map[command.Type]command.Definition {
	if len(definitions) == 0 {
		return nil
	}
	indexed := make(map[command.Type]command.Definition, len(definitions))
	for _, definition := range definitions {
		indexed[definition.Type] = definition
	}
	return indexed
}

// aggregateState converts whatever aggregate representation reached this decider
// into a concrete value.
//
// It supports both typed values and pointers for convenience in tests and caller
// boundaries while keeping downstream code safe and side-effect free.
func aggregateState(state any) aggregate.State {
	switch typed := state.(type) {
	case aggregate.State:
		return typed
	case *aggregate.State:
		if typed != nil {
			return *typed
		}
	}
	return aggregate.State{}
}

// campaignRoute routes campaign-level commands to campaign deciders.
func campaignRoute(_ CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return campaign.Decide(current.Campaign, cmd, now)
}

// campaignBootstrapRoute handles the one intentional campaign bootstrap
// workflow that emits campaign and participant events atomically.
func campaignBootstrapRoute(_ CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return campaignbootstrap.Decide(current.Campaign, cmd, now)
}

// actionRoute routes gameplay action commands to the action decider.
func actionRoute(_ CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return action.Decide(current.Action, cmd, now)
}

// sessionRoute routes session commands to the session decider.
func sessionRoute(_ CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return session.Decide(current.Session, cmd, now)
}

// sceneRoute routes scene commands to the scene decider with the full scenes map.
func sceneRoute(_ CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return scene.Decide(current.Scenes, cmd, now)
}

// sessionStartRoute handles session.start with campaign-level readiness checks.
//
// This is an intentional cross-aggregate exception: for draft campaigns, it
// emits both campaign.updated(status=active) and session.started in a single
// decision so the transition is atomic. Without this, a crash between the two
// writes could leave a campaign active with no session or a session started
// on a draft campaign.
//
// All other routes stay within a single aggregate boundary. This exception is
// acceptable because campaign activation is a one-time lifecycle transition
// that is tightly coupled to the first session start.
func sessionStartRoute(d CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	workflow := d.SessionStartWorkflow
	if workflow == nil {
		workflow = readiness.NewSessionStartWorkflow(d.Systems)
	}
	return workflow.Start(current, cmd, now)
}

// participantRoute resolves the target participant snapshot and routes accordingly.
func participantRoute(_ CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return participant.Decide(participantStateFor(cmd, current), cmd, now)
}

// inviteRoute resolves the target invite snapshot and routes accordingly.
func inviteRoute(_ CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return invite.Decide(inviteStateFor(cmd, current), cmd, now)
}

// characterRoute resolves the target character snapshot and routes accordingly.
func characterRoute(_ CoreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return character.Decide(characterStateFor(cmd, current), cmd, now)
}

// staticCoreCommandRoutes enumerates domain routing for all known core command types.
//
// This table is the onboarding-friendly contract for what the core decider owns.
func staticCoreCommandRoutes() map[command.Type]coreCommandRoute {
	return map[command.Type]coreCommandRoute{
		campaign.CommandTypeCreate:                 campaignRoute,
		campaign.CommandTypeCreateWithParticipants: campaignBootstrapRoute,
		campaign.CommandTypeUpdate:                 campaignRoute,
		campaign.CommandTypeAIBind:                 campaignRoute,
		campaign.CommandTypeAIUnbind:               campaignRoute,
		campaign.CommandTypeAIAuthRotate:           campaignRoute,
		campaign.CommandTypeFork:                   campaignRoute,
		campaign.CommandTypeEnd:                    campaignRoute,
		campaign.CommandTypeArchive:                campaignRoute,
		campaign.CommandTypeRestore:                campaignRoute,
		action.CommandTypeRollResolve:              actionRoute,
		action.CommandTypeOutcomeApply:             actionRoute,
		action.CommandTypeOutcomeReject:            actionRoute,
		action.CommandTypeNoteAdd:                  actionRoute,
		session.CommandTypeStart:                   sessionStartRoute,
		session.CommandTypeEnd:                     sessionRoute,
		session.CommandTypeGateOpen:                sessionRoute,
		session.CommandTypeGateRespond:             sessionRoute,
		session.CommandTypeGateResolve:             sessionRoute,
		session.CommandTypeGateAbandon:             sessionRoute,
		session.CommandTypeSpotlightSet:            sessionRoute,
		session.CommandTypeSpotlightClear:          sessionRoute,
		participant.CommandTypeJoin:                participantRoute,
		participant.CommandTypeUpdate:              participantRoute,
		participant.CommandTypeLeave:               participantRoute,
		participant.CommandTypeBind:                participantRoute,
		participant.CommandTypeUnbind:              participantRoute,
		participant.CommandTypeSeatReassign:        participantRoute,
		invite.CommandTypeCreate:                   inviteRoute,
		invite.CommandTypeClaim:                    inviteRoute,
		invite.CommandTypeDecline:                  inviteRoute,
		invite.CommandTypeRevoke:                   inviteRoute,
		invite.CommandTypeUpdate:                   inviteRoute,
		character.CommandTypeCreate:                characterRoute,
		character.CommandTypeUpdate:                characterRoute,
		character.CommandTypeDelete:                characterRoute,
		scene.CommandTypeCreate:                    sceneRoute,
		scene.CommandTypeUpdate:                    sceneRoute,
		scene.CommandTypeEnd:                       sceneRoute,
		scene.CommandTypeCharacterAdd:              sceneRoute,
		scene.CommandTypeCharacterRemove:           sceneRoute,
		scene.CommandTypeCharacterTransfer:         sceneRoute,
		scene.CommandTypeTransition:                sceneRoute,
		scene.CommandTypeGateOpen:                  sceneRoute,
		scene.CommandTypeGateResolve:               sceneRoute,
		scene.CommandTypeGateAbandon:               sceneRoute,
		scene.CommandTypeSpotlightSet:              sceneRoute,
		scene.CommandTypeSpotlightClear:            sceneRoute,
	}
}

// buildCoreRouteTable makes sure every registered core command type has a routing
// target and every static route has a matching registration.
//
// The forward check catches new command types without a route. The reverse check
// catches stale routes left behind after a command type is removed.
func buildCoreRouteTable(definitions []command.Definition) (map[command.Type]coreCommandRoute, error) {
	available := staticCoreCommandRoutes()
	routes := make(map[command.Type]coreCommandRoute)
	registered := make(map[command.Type]struct{})
	for _, definition := range definitions {
		if definition.Owner != command.OwnerCore {
			continue
		}
		registered[definition.Type] = struct{}{}
		route, ok := available[definition.Type]
		if !ok {
			return nil, fmt.Errorf("core command route missing for registered type %s", definition.Type)
		}
		routes[definition.Type] = route
	}
	// Reverse check: detect stale static routes not backed by a registration.
	var stale []string
	for cmdType := range available {
		if _, ok := registered[cmdType]; !ok {
			stale = append(stale, string(cmdType))
		}
	}
	if len(stale) > 0 {
		return nil, fmt.Errorf("stale static core command routes without registration: %s",
			strings.Join(stale, ", "))
	}
	return routes, nil
}

// participantStateFor loads the target participant from command metadata.
func participantStateFor(cmd command.Command, current aggregate.State) participant.State {
	if current.Participants == nil {
		return participant.State{}
	}
	id := strings.TrimSpace(cmd.EntityID)
	if id == "" {
		return participant.State{}
	}
	return current.Participants[ids.ParticipantID(id)]
}

// characterStateFor loads the target character from command metadata.
func characterStateFor(cmd command.Command, current aggregate.State) character.State {
	if current.Characters == nil {
		return character.State{}
	}
	id := strings.TrimSpace(cmd.EntityID)
	if id == "" {
		return character.State{}
	}
	return current.Characters[ids.CharacterID(id)]
}

// inviteStateFor loads the target invite from command metadata.
func inviteStateFor(cmd command.Command, current aggregate.State) invite.State {
	if current.Invites == nil {
		return invite.State{}
	}
	id := strings.TrimSpace(cmd.EntityID)
	if id == "" {
		return invite.State{}
	}
	return current.Invites[ids.InviteID(id)]
}
