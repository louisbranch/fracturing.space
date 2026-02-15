package engine

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
)

// Registries bundles the command/event/system registries.
type Registries struct {
	Commands *command.Registry
	Events   *event.Registry
	Systems  *system.Registry
}

// BuildRegistries registers core and system modules.
func BuildRegistries(modules ...system.Module) (Registries, error) {
	commandRegistry := command.NewRegistry()
	eventRegistry := event.NewRegistry()
	systemRegistry := system.NewRegistry()

	if err := campaign.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := action.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := session.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := participant.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := invite.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}
	if err := character.RegisterCommands(commandRegistry); err != nil {
		return Registries{}, err
	}

	if err := campaign.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := action.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := session.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := participant.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := invite.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}
	if err := character.RegisterEvents(eventRegistry); err != nil {
		return Registries{}, err
	}

	for _, module := range modules {
		if err := systemRegistry.Register(module); err != nil {
			return Registries{}, err
		}
		if err := module.RegisterCommands(commandRegistry); err != nil {
			return Registries{}, err
		}
		if err := module.RegisterEvents(eventRegistry); err != nil {
			return Registries{}, err
		}
	}

	return Registries{
		Commands: commandRegistry,
		Events:   eventRegistry,
		Systems:  systemRegistry,
	}, nil
}
