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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// coreCommandRouter owns routing for core (non-system) command families.
//
// It is the only engine collaborator that knows about the readiness-owned
// `session.start` exception.
type coreCommandRouter struct {
	systems      *module.Registry
	sessionStart readiness.SessionStartWorkflow
	definitions  map[command.Type]command.Definition
	routes       map[command.Type]coreCommandRoute
}

// coreCommandRoute maps a normalized aggregate state + command into one
// core-domain decision path.
type coreCommandRoute func(router coreCommandRouter, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision

// newCoreCommandRouter validates core route coverage once and captures the
// readiness workflow seam used by `session.start`.
func newCoreCommandRouter(systems *module.Registry, definitions []command.Definition) (coreCommandRouter, error) {
	routes, err := buildCoreRouteTable(definitions)
	if err != nil {
		return coreCommandRouter{}, err
	}
	return coreCommandRouter{
		systems:      systems,
		sessionStart: readiness.NewSessionStartWorkflow(systems),
		definitions:  indexCommandDefinitions(definitions),
		routes:       routes,
	}, nil
}

// Decide applies active-session policy and then routes a core command to its
// owning aggregate or workflow seam.
func (r coreCommandRouter) Decide(current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	if decision, blocked := RejectActiveSessionBlockedCommand(current.Session, cmd, r.definitionFor(cmd.Type)); blocked {
		return decision
	}
	routes := r.routes
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
	return route(r, current, cmd, now)
}

// definitionFor returns the registered command definition when one exists so
// active-session policy checks can stay data-driven.
func (r coreCommandRouter) definitionFor(cmdType command.Type) command.Definition {
	if definition, ok := r.definitions[cmdType]; ok {
		return definition
	}
	return command.Definition{}
}

// sessionStartWorkflow returns the injected readiness workflow when present and
// otherwise reconstructs the default workflow for zero-value decider tests.
func (r coreCommandRouter) sessionStartWorkflow() readiness.SessionStartWorkflow {
	if r.sessionStart != nil {
		return r.sessionStart
	}
	return readiness.NewSessionStartWorkflow(r.systems)
}

// indexCommandDefinitions builds a command-type lookup table used by active
// session policy checks.
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

// campaignRoute routes campaign-level commands to campaign deciders.
func campaignRoute(_ coreCommandRouter, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return campaign.Decide(current.Campaign, cmd, now)
}

// campaignBootstrapRoute handles the one intentional campaign bootstrap
// workflow that emits campaign and participant events atomically.
func campaignBootstrapRoute(_ coreCommandRouter, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return campaignbootstrap.Decide(current.Campaign, cmd, now)
}

// actionRoute routes gameplay action commands to the action decider.
func actionRoute(_ coreCommandRouter, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return action.Decide(current.Action, cmd, now)
}

// sessionRoute routes session commands to the session decider.
func sessionRoute(_ coreCommandRouter, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return session.Decide(current.Session, cmd, now)
}

// sceneRoute routes scene commands to the scene decider with the full scenes map.
func sceneRoute(_ coreCommandRouter, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
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
func sessionStartRoute(router coreCommandRouter, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return router.sessionStartWorkflow().Start(current, cmd, now)
}

// participantRoute resolves the target participant snapshot and routes accordingly.
func participantRoute(_ coreCommandRouter, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return participant.Decide(participantStateFor(cmd, current), cmd, now)
}

// characterRoute resolves the target character snapshot and routes accordingly.
func characterRoute(_ coreCommandRouter, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return character.Decide(characterStateFor(cmd, current), cmd, now)
}

// staticCoreCommandRoutes enumerates domain routing for all known core command types.
//
// This table is the onboarding-friendly contract for what the core router owns.
func staticCoreCommandRoutes() map[command.Type]coreCommandRoute {
	return map[command.Type]coreCommandRoute{
		campaign.CommandTypeCreate:                   campaignRoute,
		campaign.CommandTypeCreateWithParticipants:   campaignBootstrapRoute,
		campaign.CommandTypeUpdate:                   campaignRoute,
		campaign.CommandTypeAIBind:                   campaignRoute,
		campaign.CommandTypeAIUnbind:                 campaignRoute,
		campaign.CommandTypeAIAuthRotate:             campaignRoute,
		campaign.CommandTypeFork:                     campaignRoute,
		campaign.CommandTypeEnd:                      campaignRoute,
		campaign.CommandTypeArchive:                  campaignRoute,
		campaign.CommandTypeRestore:                  campaignRoute,
		action.CommandTypeRollResolve:                actionRoute,
		action.CommandTypeOutcomeApply:               actionRoute,
		action.CommandTypeOutcomeReject:              actionRoute,
		action.CommandTypeNoteAdd:                    actionRoute,
		session.CommandTypeStart:                     sessionStartRoute,
		session.CommandTypeEnd:                       sessionRoute,
		session.CommandTypeGateOpen:                  sessionRoute,
		session.CommandTypeGateRespond:               sessionRoute,
		session.CommandTypeGateResolve:               sessionRoute,
		session.CommandTypeGateAbandon:               sessionRoute,
		session.CommandTypeSpotlightSet:              sessionRoute,
		session.CommandTypeSpotlightClear:            sessionRoute,
		session.CommandTypeSceneActivate:             sessionRoute,
		session.CommandTypeGMAuthoritySet:            sessionRoute,
		session.CommandTypeOOCOpen:                   sessionRoute,
		session.CommandTypeOOCPost:                   sessionRoute,
		session.CommandTypeOOCReadyMark:              sessionRoute,
		session.CommandTypeOOCReadyClear:             sessionRoute,
		session.CommandTypeOOCClose:                  sessionRoute,
		session.CommandTypeOOCResolve:                sessionRoute,
		session.CommandTypeAITurnQueue:               sessionRoute,
		session.CommandTypeAITurnStart:               sessionRoute,
		session.CommandTypeAITurnFail:                sessionRoute,
		session.CommandTypeAITurnClear:               sessionRoute,
		participant.CommandTypeJoin:                  participantRoute,
		participant.CommandTypeUpdate:                participantRoute,
		participant.CommandTypeLeave:                 participantRoute,
		participant.CommandTypeBind:                  participantRoute,
		participant.CommandTypeUnbind:                participantRoute,
		participant.CommandTypeSeatReassign:          participantRoute,
		character.CommandTypeCreate:                  characterRoute,
		character.CommandTypeUpdate:                  characterRoute,
		character.CommandTypeDelete:                  characterRoute,
		scene.CommandTypeCreate:                      sceneRoute,
		scene.CommandTypeUpdate:                      sceneRoute,
		scene.CommandTypeEnd:                         sceneRoute,
		scene.CommandTypeCharacterAdd:                sceneRoute,
		scene.CommandTypeCharacterRemove:             sceneRoute,
		scene.CommandTypeCharacterTransfer:           sceneRoute,
		scene.CommandTypeTransition:                  sceneRoute,
		scene.CommandTypeGateOpen:                    sceneRoute,
		scene.CommandTypeGateResolve:                 sceneRoute,
		scene.CommandTypeGateAbandon:                 sceneRoute,
		scene.CommandTypeSpotlightSet:                sceneRoute,
		scene.CommandTypeSpotlightClear:              sceneRoute,
		scene.CommandTypePlayerPhaseStart:            sceneRoute,
		scene.CommandTypePlayerPhasePost:             sceneRoute,
		scene.CommandTypePlayerPhaseYield:            sceneRoute,
		scene.CommandTypePlayerPhaseUnyield:          sceneRoute,
		scene.CommandTypePlayerPhaseAccept:           sceneRoute,
		scene.CommandTypePlayerPhaseRequestRevisions: sceneRoute,
		scene.CommandTypePlayerPhaseEnd:              sceneRoute,
		scene.CommandTypeGMInteractionCommit:         sceneRoute,
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
