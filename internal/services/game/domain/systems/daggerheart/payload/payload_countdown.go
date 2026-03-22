package payload

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"

// --- Countdowns ---

// CountdownCreatePayload captures the payload for sys.daggerheart.countdown.create commands.
type CountdownCreatePayload struct {
	CountdownID       dhids.CountdownID `json:"countdown_id"`
	Name              string            `json:"name"`
	Kind              string            `json:"kind"`
	Current           int               `json:"current"`
	Max               int               `json:"max"`
	Direction         string            `json:"direction"`
	Looping           bool              `json:"looping"`
	Variant           string            `json:"variant,omitempty"`
	TriggerEventType  string            `json:"trigger_event_type,omitempty"`
	LinkedCountdownID dhids.CountdownID `json:"linked_countdown_id,omitempty"`
}

// CountdownCreatedPayload captures the payload for sys.daggerheart.countdown_created events.
type CountdownCreatedPayload = CountdownCreatePayload

// CountdownUpdatePayload captures the payload for sys.daggerheart.countdown.update commands.
type CountdownUpdatePayload struct {
	CountdownID dhids.CountdownID `json:"countdown_id"`
	Before      int               `json:"before"`
	After       int               `json:"after"`
	Delta       int               `json:"delta"`
	Looped      bool              `json:"looped"`
	Reason      string            `json:"reason,omitempty"`
}

// CountdownUpdatedPayload captures the payload for sys.daggerheart.countdown_updated events.
type CountdownUpdatedPayload struct {
	CountdownID dhids.CountdownID `json:"countdown_id"`
	Value       int               `json:"after"`
	Delta       int               `json:"delta"`
	Looped      bool              `json:"looped"`
	Reason      string            `json:"reason,omitempty"`
}

// CountdownDeletePayload captures the payload for sys.daggerheart.countdown.delete commands.
type CountdownDeletePayload struct {
	CountdownID dhids.CountdownID `json:"countdown_id"`
	Reason      string            `json:"reason,omitempty"`
}

// CountdownDeletedPayload captures the payload for sys.daggerheart.countdown_deleted events.
type CountdownDeletedPayload = CountdownDeletePayload
