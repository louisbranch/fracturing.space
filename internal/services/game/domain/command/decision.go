package command

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"

// Decision represents the pure outcome of handling a command.
//
// It separates accepted future state changes from immediate domain reasons so
// callers can decide whether to stop at validation, append events, or return a
// user-facing rejection.
type Decision struct {
	Events     []event.Event
	Rejections []Rejection
}

// Rejection captures a domain-level reason a command was declined.
//
// The rejection code is intentionally stable for integrations, while the message
// remains human-readable for diagnostics.
type Rejection struct {
	Code    string
	Message string
}

// Accept returns a decision that emits the provided events.
//
// Returning a decision instead of mutating state directly keeps command handlers
// deterministic and replay-friendly.
func Accept(events ...event.Event) Decision {
	return Decision{Events: append([]event.Event(nil), events...)}
}

// Reject returns a decision that carries the provided rejections.
//
// Reject intentionally carries no events, so no replayable state change is
// created for failed validation paths.
func Reject(rejections ...Rejection) Decision {
	return Decision{Rejections: append([]Rejection(nil), rejections...)}
}
