package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func (a Applier) applySystemEvent(ctx context.Context, evt event.Event) error {
	if a.Adapters == nil {
		return fmt.Errorf("system adapters are not configured")
	}
	if strings.TrimSpace(evt.SystemID) == "" {
		return fmt.Errorf("system_id is required for system events")
	}
	gameSystem, err := parseGameSystem(evt.SystemID)
	if err != nil {
		return err
	}
	adapter, err := a.Adapters.GetRequired(gameSystem, evt.SystemVersion)
	if err != nil {
		return err
	}
	return adapter.Apply(ctx, evt)
}
