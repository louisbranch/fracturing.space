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
	systemID := strings.TrimSpace(evt.SystemID)
	if systemID == "" {
		return fmt.Errorf("system_id is required for system events")
	}
	adapter, err := a.Adapters.GetRequired(systemID, evt.SystemVersion)
	if err != nil {
		return err
	}
	return adapter.Apply(ctx, evt)
}
