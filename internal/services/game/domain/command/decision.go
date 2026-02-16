package command

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"

// Decision represents the pure outcome of handling a command.
type Decision struct {
	Events     []event.Event
	Rejections []Rejection
}

// Rejection captures a domain-level reason a command was declined.
type Rejection struct {
	Code    string
	Message string
}

// Accept returns a decision that emits the provided events.
func Accept(events ...event.Event) Decision {
	return Decision{Events: append([]event.Event(nil), events...)}
}

// Reject returns a decision that carries the provided rejections.
func Reject(rejections ...Rejection) Decision {
	return Decision{Rejections: append([]Rejection(nil), rejections...)}
}
