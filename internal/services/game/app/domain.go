package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/checkpoint"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// configureDomain wires the write-path domain engine into gRPC stores when enabled.
//
// This is intentionally guarded by config so deployments can run without domain
// execution in specific environments (for example, projection-only workflows).
func configureDomain(srvEnv serverEnv, stores *gamegrpc.Stores, registries engine.Registries) error {
	if !srvEnv.DomainEnabled {
		return nil
	}
	if stores == nil {
		return errors.New("stores are required")
	}
	domainEngine, err := buildDomainEngine(stores.Event, registries)
	if err != nil {
		return fmt.Errorf("build domain engine: %w", err)
	}
	stores.Domain = domainEngine
	return nil
}

// buildDomainEngine builds the replay-capable domain handler used by write paths.
//
// It composes registries, replay-based state loading, gate evaluation, and
// decider routing once, so command execution stays consistent for every request.
func buildDomainEngine(eventStore storage.EventStore, registries engine.Registries) (gamegrpc.Domain, error) {
	if eventStore == nil {
		return nil, errors.New("event store is required")
	}
	routes, err := buildCoreRouteTable(registries.Commands.ListDefinitions())
	if err != nil {
		return nil, fmt.Errorf("build core command routes: %w", err)
	}

	checkpoints := checkpoint.NewMemory()
	applier := &aggregate.Applier{
		Events:         registries.Events,
		SystemRegistry: registries.Systems,
	}
	stateLoader := engine.ReplayStateLoader{
		Events:       gamegrpc.NewEventStoreAdapter(eventStore),
		Checkpoints:  checkpoints,
		Snapshots:    checkpoints,
		Applier:      applier,
		StateFactory: func() any { return aggregate.State{} },
	}
	return engine.NewHandler(engine.HandlerConfig{
		Commands:        registries.Commands,
		Events:          registries.Events,
		Journal:         gamegrpc.NewJournalAdapter(eventStore),
		Checkpoints:     checkpoints,
		Snapshots:       checkpoints,
		Gate:            engine.DecisionGate{Registry: registries.Commands},
		GateStateLoader: engine.ReplayGateStateLoader{StateLoader: stateLoader},
		StateLoader:     stateLoader,
		Decider:         coreDecider{Systems: registries.Systems, routes: routes},
		Applier:         applier,
	})
}

// coreDecider is the top-level decider for core (non-system) commands.
//
// It keeps command routing explicit: each command type maps to exactly one
// aggregate route, while system commands are dispatched by system id/version.
type coreDecider struct {
	Systems *module.Registry
	routes  map[command.Type]coreCommandRoute
}

// coreCommandRoute maps a normalized aggregate state + command into one decision path.
type coreCommandRoute func(d coreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision

const (
	coreCommandTypeCampaignCreate          command.Type = "campaign.create"
	coreCommandTypeCampaignUpdate          command.Type = "campaign.update"
	coreCommandTypeCampaignFork            command.Type = "campaign.fork"
	coreCommandTypeCampaignEnd             command.Type = "campaign.end"
	coreCommandTypeCampaignArchive         command.Type = "campaign.archive"
	coreCommandTypeCampaignRestore         command.Type = "campaign.restore"
	coreCommandTypeActionRollResolve       command.Type = "action.roll.resolve"
	coreCommandTypeActionOutcomeApply      command.Type = "action.outcome.apply"
	coreCommandTypeActionOutcomeReject     command.Type = "action.outcome.reject"
	coreCommandTypeStoryNoteAdd            command.Type = "story.note.add"
	coreCommandTypeSessionStart            command.Type = "session.start"
	coreCommandTypeSessionEnd              command.Type = "session.end"
	coreCommandTypeSessionGateOpen         command.Type = "session.gate_open"
	coreCommandTypeSessionGateResolve      command.Type = "session.gate_resolve"
	coreCommandTypeSessionGateAbandon      command.Type = "session.gate_abandon"
	coreCommandTypeSessionSpotlightSet     command.Type = "session.spotlight_set"
	coreCommandTypeSessionSpotlightClear   command.Type = "session.spotlight_clear"
	coreCommandTypeParticipantJoin         command.Type = "participant.join"
	coreCommandTypeParticipantSeatReassign command.Type = "participant.seat.reassign"
	coreCommandTypeParticipantUpdate       command.Type = "participant.update"
	coreCommandTypeParticipantLeave        command.Type = "participant.leave"
	coreCommandTypeParticipantBind         command.Type = "participant.bind"
	coreCommandTypeParticipantUnbind       command.Type = "participant.unbind"
	coreCommandTypeSeatReassign            command.Type = "seat.reassign"
	coreCommandTypeInviteCreate            command.Type = "invite.create"
	coreCommandTypeInviteClaim             command.Type = "invite.claim"
	coreCommandTypeInviteRevoke            command.Type = "invite.revoke"
	coreCommandTypeInviteUpdate            command.Type = "invite.update"
	coreCommandTypeCharacterCreate         command.Type = "character.create"
	coreCommandTypeCharacterUpdate         command.Type = "character.update"
	coreCommandTypeCharacterDelete         command.Type = "character.delete"
	coreCommandTypeCharacterProfileUpdate  command.Type = "character.profile_update"
)

// campaignRoute routes campaign-level commands to campaign deciders.
func campaignRoute(_ coreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return campaign.Decide(current.Campaign, cmd, now)
}

// actionRoute routes gameplay action commands to the action decider.
func actionRoute(_ coreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return action.Decide(current.Action, cmd, now)
}

// sessionRoute routes session commands to the session decider.
func sessionRoute(_ coreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return session.Decide(current.Session, cmd, now)
}

// participantRoute resolves the target participant snapshot and routes accordingly.
func participantRoute(_ coreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return participant.Decide(participantStateFor(cmd, current), cmd, now)
}

// inviteRoute resolves the target invite snapshot and routes accordingly.
func inviteRoute(_ coreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return invite.Decide(inviteStateFor(cmd, current), cmd, now)
}

// characterRoute resolves the target character snapshot and routes accordingly.
func characterRoute(_ coreDecider, current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	return character.Decide(characterStateFor(cmd, current), cmd, now)
}

// staticCoreCommandRoutes enumerates domain routing for all known core command types.
//
// This table is the onboarding-friendly contract for what the core decider owns.
func staticCoreCommandRoutes() map[command.Type]coreCommandRoute {
	return map[command.Type]coreCommandRoute{
		coreCommandTypeCampaignCreate:          campaignRoute,
		coreCommandTypeCampaignUpdate:          campaignRoute,
		coreCommandTypeCampaignFork:            campaignRoute,
		coreCommandTypeCampaignEnd:             campaignRoute,
		coreCommandTypeCampaignArchive:         campaignRoute,
		coreCommandTypeCampaignRestore:         campaignRoute,
		coreCommandTypeActionRollResolve:       actionRoute,
		coreCommandTypeActionOutcomeApply:      actionRoute,
		coreCommandTypeActionOutcomeReject:     actionRoute,
		coreCommandTypeStoryNoteAdd:            actionRoute,
		coreCommandTypeSessionStart:            sessionRoute,
		coreCommandTypeSessionEnd:              sessionRoute,
		coreCommandTypeSessionGateOpen:         sessionRoute,
		coreCommandTypeSessionGateResolve:      sessionRoute,
		coreCommandTypeSessionGateAbandon:      sessionRoute,
		coreCommandTypeSessionSpotlightSet:     sessionRoute,
		coreCommandTypeSessionSpotlightClear:   sessionRoute,
		coreCommandTypeParticipantJoin:         participantRoute,
		coreCommandTypeParticipantSeatReassign: participantRoute,
		coreCommandTypeParticipantUpdate:       participantRoute,
		coreCommandTypeParticipantLeave:        participantRoute,
		coreCommandTypeParticipantBind:         participantRoute,
		coreCommandTypeParticipantUnbind:       participantRoute,
		coreCommandTypeSeatReassign:            participantRoute,
		coreCommandTypeInviteCreate:            inviteRoute,
		coreCommandTypeInviteClaim:             inviteRoute,
		coreCommandTypeInviteRevoke:            inviteRoute,
		coreCommandTypeInviteUpdate:            inviteRoute,
		coreCommandTypeCharacterCreate:         characterRoute,
		coreCommandTypeCharacterUpdate:         characterRoute,
		coreCommandTypeCharacterDelete:         characterRoute,
		coreCommandTypeCharacterProfileUpdate:  characterRoute,
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

func (d coreDecider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
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

// aggregateState converts whatever aggregate representation reached this decider into a concrete value.
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

type participantIDPayload struct {
	ParticipantID string `json:"participant_id"`
}

func participantStateFor(cmd command.Command, current aggregate.State) participant.State {
	if current.Participants == nil {
		return participant.State{}
	}
	participantID := strings.TrimSpace(cmd.EntityID)
	if participantID == "" {
		var payload participantIDPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		participantID = strings.TrimSpace(payload.ParticipantID)
	}
	if participantID == "" {
		return participant.State{}
	}
	return current.Participants[participantID]
}

type characterIDPayload struct {
	CharacterID string `json:"character_id"`
}

// characterStateFor loads the target character from command metadata.
//
// Commands can carry the character reference in either EntityID or payload body;
// this lets callers keep transport-level shape flexible while preserving deterministic
// routing.
func characterStateFor(cmd command.Command, current aggregate.State) character.State {
	if current.Characters == nil {
		return character.State{}
	}
	characterID := strings.TrimSpace(cmd.EntityID)
	if characterID == "" {
		var payload characterIDPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		characterID = strings.TrimSpace(payload.CharacterID)
	}
	if characterID == "" {
		return character.State{}
	}
	return current.Characters[characterID]
}

type inviteIDPayload struct {
	InviteID string `json:"invite_id"`
}

func inviteStateFor(cmd command.Command, current aggregate.State) invite.State {
	if current.Invites == nil {
		return invite.State{}
	}
	inviteID := strings.TrimSpace(cmd.EntityID)
	if inviteID == "" {
		var payload inviteIDPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		inviteID = strings.TrimSpace(payload.InviteID)
	}
	if inviteID == "" {
		return invite.State{}
	}
	return current.Invites[inviteID]
}
