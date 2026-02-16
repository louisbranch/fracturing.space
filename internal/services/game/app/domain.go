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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func configureDomain(srvEnv serverEnv, stores *gamegrpc.Stores) error {
	if !srvEnv.DomainEnabled {
		return nil
	}
	if stores == nil {
		return errors.New("stores are required")
	}
	buildDomainEngine, err := buildDomainEngine(stores.Event)
	if err != nil {
		return fmt.Errorf("build domain engine: %w", err)
	}
	stores.Domain = buildDomainEngine
	return nil
}

func buildDomainEngine(eventStore storage.EventStore) (gamegrpc.Domain, error) {
	if eventStore == nil {
		return nil, errors.New("event store is required")
	}
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		return nil, fmt.Errorf("build registries: %w", err)
	}

	checkpoints := checkpoint.NewMemory()
	applier := aggregate.Applier{SystemRegistry: registries.Systems}
	stateLoader := engine.ReplayStateLoader{
		Events:       gamegrpc.NewEventStoreAdapter(eventStore),
		Checkpoints:  checkpoints,
		Snapshots:    checkpoints,
		Applier:      applier,
		StateFactory: func() any { return aggregate.State{} },
	}
	return engine.Handler{
		Commands:        registries.Commands,
		Events:          registries.Events,
		Journal:         gamegrpc.NewJournalAdapter(eventStore),
		Checkpoints:     checkpoints,
		Snapshots:       checkpoints,
		Gate:            engine.DecisionGate{Registry: registries.Commands},
		GateStateLoader: engine.ReplayGateStateLoader{StateLoader: stateLoader},
		StateLoader:     stateLoader,
		Decider:         coreDecider{Systems: registries.Systems},
		Applier:         applier,
	}, nil
}

type coreDecider struct {
	Systems *system.Registry
}

func (d coreDecider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
	current := aggregateState(state)
	if strings.TrimSpace(cmd.SystemID) != "" || strings.TrimSpace(cmd.SystemVersion) != "" {
		key := system.Key{ID: cmd.SystemID, Version: cmd.SystemVersion}
		systemState := current.Systems[key]
		decision, err := system.RouteCommand(d.Systems, systemState, cmd, now)
		if err != nil {
			return command.Reject(command.Rejection{Code: "SYSTEM_COMMAND_REJECTED", Message: err.Error()})
		}
		return decision
	}
	cmdType := string(cmd.Type)
	switch {
	case strings.HasPrefix(cmdType, "campaign."):
		return campaign.Decide(current.Campaign, cmd, now)
	case strings.HasPrefix(cmdType, "action."):
		return action.Decide(action.State{}, cmd, now)
	case strings.HasPrefix(cmdType, "session."):
		return session.Decide(current.Session, cmd, now)
	case strings.HasPrefix(cmdType, "participant.") || strings.HasPrefix(cmdType, "seat."):
		return participant.Decide(participantStateFor(cmd, current), cmd, now)
	case strings.HasPrefix(cmdType, "invite."):
		return invite.Decide(inviteStateFor(cmd, current), cmd, now)
	case strings.HasPrefix(cmdType, "character."):
		return character.Decide(characterStateFor(cmd, current), cmd, now)
	default:
		return command.Reject(command.Rejection{
			Code:    "COMMAND_TYPE_UNSUPPORTED",
			Message: "command type is not supported by core decider",
		})
	}
}

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
