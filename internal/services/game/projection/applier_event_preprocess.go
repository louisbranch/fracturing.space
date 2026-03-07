package projection

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"

// prepareEventForProjection resolves aliases and applies intent filtering.
// The returned bool reports whether the event should be projected.
func (a Applier) prepareEventForProjection(evt event.Event) (event.Event, bool) {
	if a.Events == nil {
		return evt, true
	}
	resolved := a.Events.Resolve(evt.Type)
	evt.Type = resolved
	// Skip events that should not be projected (audit-only and replay-only).
	// ShouldProject centralizes the intent contract.
	return evt, a.Events.ShouldProject(resolved)
}
