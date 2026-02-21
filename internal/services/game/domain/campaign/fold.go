package campaign

import (
	"encoding/json"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Fold applies an event to campaign state.
func Fold(state State, evt event.Event) State {
	if evt.Type == eventTypeCreated {
		state.Created = true
		var payload CreatePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		state.Name = payload.Name
		state.GameSystem = payload.GameSystem
		state.GmMode = payload.GmMode
		state.Status = StatusDraft
		state.CoverAssetID = strings.TrimSpace(payload.CoverAssetID)
	}
	if evt.Type == eventTypeUpdated {
		var payload UpdatePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
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
			}
		}
	}
	return state
}
