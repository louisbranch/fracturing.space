package campaign

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// FoldHandledTypes returns the event types handled by the campaign fold function.
// This is used by fold coverage validation to ensure every projection-required event
// has a corresponding fold handler.
func FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeCreated,
		EventTypeUpdated,
		EventTypeForked,
	}
}

// Fold applies an event to campaign state. It returns an error if a recognized
// event carries a payload that cannot be unmarshalled.
func Fold(state State, evt event.Event) (State, error) {
	switch evt.Type {
	case EventTypeCreated:
		state.Created = true
		var payload CreatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("campaign fold %s: %w", evt.Type, err)
		}
		state.Name = payload.Name
		state.GameSystem = payload.GameSystem
		state.GmMode = payload.GmMode
		state.Status = StatusDraft
		state.CoverAssetID = strings.TrimSpace(payload.CoverAssetID)
		state.CoverSetID = strings.TrimSpace(payload.CoverSetID)
	case EventTypeUpdated:
		var payload UpdatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("campaign fold %s: %w", evt.Type, err)
		}
		for key, value := range payload.Fields {
			switch key {
			case "name":
				state.Name = strings.TrimSpace(value)
			case "status":
				state.Status = Status(strings.TrimSpace(value))
			case "theme_prompt":
				state.ThemePrompt = strings.TrimSpace(value)
			case "cover_asset_id":
				state.CoverAssetID = strings.TrimSpace(value)
			case "cover_set_id":
				state.CoverSetID = strings.TrimSpace(value)
			}
		}
	case EventTypeForked:
		// Projection-only: fork lineage metadata does not affect campaign
		// aggregate state but is acknowledged here so fold coverage
		// validation knows the event was deliberately considered.
	}
	return state, nil
}
