package integrity

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// EventHash computes the content hash for a single event payload.
//
// Delegates to the domain event package's canonical envelope builder so
// field ordering is defined in one place and cannot drift between layers.
func EventHash(evt event.Event) (string, error) {
	return event.EventHash(evt)
}

// ChainHash computes the SHA-256 hash that links an event to its predecessor.
//
// Delegates to the domain event package's canonical envelope builder so
// field ordering is defined in one place and cannot drift between layers.
func ChainHash(evt event.Event, prevHash string) (string, error) {
	return event.ChainHash(evt, prevHash)
}
