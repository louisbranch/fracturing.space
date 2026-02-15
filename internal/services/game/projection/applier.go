package projection

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Applier applies event journal entries to projection stores.
type Applier struct {
	Campaign         storage.CampaignStore
	Character        storage.CharacterStore
	CampaignFork     storage.CampaignForkStore
	Daggerheart      storage.DaggerheartStore
	ClaimIndex       storage.ClaimIndexStore
	Invite           storage.InviteStore
	Participant      storage.ParticipantStore
	Session          storage.SessionStore
	SessionGate      storage.SessionGateStore
	SessionSpotlight storage.SessionSpotlightStore
	Adapters         *systems.AdapterRegistry
}

// ensureTimestamp returns the given timestamp in UTC, or the current time if zero.
func ensureTimestamp(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts.UTC()
}

// marshalResolutionPayload encodes a gate resolution decision and resolution map
// into a single JSON blob. Returns nil when both inputs are empty.
func marshalResolutionPayload(decision string, resolution map[string]any) ([]byte, error) {
	if strings.TrimSpace(decision) == "" && len(resolution) == 0 {
		return nil, nil
	}
	combined := map[string]any{}
	if strings.TrimSpace(decision) != "" {
		combined["decision"] = strings.TrimSpace(decision)
	}
	for key, value := range resolution {
		combined[key] = value
	}
	return json.Marshal(combined)
}

// marshalOptionalMap encodes a map to JSON, returning nil when the map is empty.
func marshalOptionalMap(values map[string]any) ([]byte, error) {
	if len(values) == 0 {
		return nil, nil
	}
	return json.Marshal(values)
}
