package module

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// DecideFunc handles the common unmarshal → validate → marshal → emit flow for
// simple decider cases. The validate callback receives a pointer to the
// unmarshaled payload and may mutate it (e.g. TrimSpace normalization) before
// it is re-marshaled into the emitted event. Return a non-nil Rejection to
// reject the command.
//
// The entityID callback derives the fallback entity ID from the payload when
// cmd.EntityID is empty — the common pattern in system deciders.
//
// Complex cases that emit multiple events, transform the event type, or need
// access to aggregate snapshot state should use the raw switch approach instead.
func DecideFunc[P any](
	cmd command.Command,
	eventType event.Type,
	entityType string,
	entityID func(*P) string,
	validate func(*P, func() time.Time) *command.Rejection,
	now func() time.Time,
) command.Decision {
	var payload P
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_DECODE_FAILED",
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	if now == nil {
		now = time.Now
	}
	if validate != nil {
		if rejection := validate(&payload, now); rejection != nil {
			return command.Reject(*rejection)
		}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    "PAYLOAD_ENCODE_FAILED",
			Message: fmt.Sprintf("encode %s payload: %v", cmd.Type, err),
		})
	}
	eid := strings.TrimSpace(cmd.EntityID)
	if eid == "" && entityID != nil {
		eid = entityID(&payload)
	}
	etype := strings.TrimSpace(cmd.EntityType)
	if etype == "" {
		etype = entityType
	}
	evt := command.NewEvent(cmd, eventType, etype, eid, payloadJSON, now().UTC())
	return command.Accept(evt)
}
