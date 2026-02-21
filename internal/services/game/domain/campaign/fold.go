package campaign

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Fold applies an event to campaign state. It returns an error if a recognized
// event carries a payload that cannot be unmarshalled.
func Fold(state State, evt event.Event) (State, error) {
	if evt.Type == EventTypeCreated {
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
	}
	if evt.Type == EventTypeUpdated {
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
	}
	return state, nil
}
