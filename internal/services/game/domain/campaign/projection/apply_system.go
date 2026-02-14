package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
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
	adapter := a.Adapters.Get(gameSystem, evt.SystemVersion)
	if adapter == nil {
		return fmt.Errorf("system adapter not found for %s (%s)", evt.SystemID, evt.SystemVersion)
	}
	return adapter.ApplyEvent(ctx, evt)
}
