package projection

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Applier applies event journal entries to projection stores.
type Applier struct {
	// Events resolves type aliases before routing.
	Events *event.Registry
	// Campaign writes campaign metadata read models.
	Campaign storage.CampaignStore
	// Character writes character read models.
	Character storage.CharacterStore
	// CampaignFork stores campaign fork metadata.
	CampaignFork storage.CampaignForkStore
	// ClaimIndex writes/reads participant-user claim mappings.
	ClaimIndex storage.ClaimIndexStore
	// Invite writes invite read models.
	Invite storage.InviteStore
	// Participant writes participant read models.
	Participant storage.ParticipantStore
	// Session writes session metadata read models.
	Session storage.SessionStore
	// SessionGate writes open/resolved gate state.
	SessionGate storage.SessionGateStore
	// SessionSpotlight writes session spotlight state.
	SessionSpotlight storage.SessionSpotlightStore
	// Adapters holds extension-specific projection hooks including
	// system event application and character profile updates.
	Adapters *systems.AdapterRegistry
}

// ensureTimestamp normalizes timestamps so projections always persist UTC, defaulting
// to now for defensive event payloads that do not set time.
func ensureTimestamp(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts.UTC()
}

// marshalResolutionPayload returns a compact JSON payload for gate resolution.
//
// This payload is read by API/UI layers so they can show both explicit decisions
// and resolved map values without requiring schema-specific types.
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

// marshalOptionalMap encodes an optional map and returns nil for empty payloads.
func marshalOptionalMap(values map[string]any) ([]byte, error) {
	if len(values) == 0 {
		return nil, nil
	}
	return json.Marshal(values)
}
