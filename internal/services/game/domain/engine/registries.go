package engine

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// Registries bundles the command/event/system registries.
type Registries struct {
	Commands *command.Registry
	Events   *event.Registry
	Systems  *module.Registry
}
