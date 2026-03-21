package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
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
	// Participant writes participant read models.
	Participant storage.ParticipantStore
	// Session writes session metadata read models.
	Session storage.SessionStore
	// SessionGate writes open/resolved gate state.
	SessionGate storage.SessionGateStore
	// SessionSpotlight writes session spotlight state.
	SessionSpotlight storage.SessionSpotlightStore
	// SessionInteraction writes active-scene and OOC interaction state.
	SessionInteraction storage.SessionInteractionStore
	// Scene writes scene metadata read models.
	Scene storage.SceneStore
	// SceneCharacter writes scene character membership.
	SceneCharacter storage.SceneCharacterStore
	// SceneGate writes scene gate state.
	SceneGate storage.SceneGateStore
	// SceneSpotlight writes scene spotlight state.
	SceneSpotlight storage.SceneSpotlightStore
	// SceneInteraction writes scene player-phase state.
	SceneInteraction storage.SceneInteractionStore
	// SceneGMInteraction writes immutable GM-authored interaction history.
	SceneGMInteraction storage.SceneGMInteractionStore
	// Adapters holds extension-specific projection hooks including
	// system event application and character profile updates.
	Adapters *bridge.AdapterRegistry
	// Watermarks tracks per-campaign projection progress so startup can
	// detect and repair gaps. When nil, watermark tracking is disabled.
	Watermarks storage.ProjectionWatermarkStore
	// Now returns the current time used for watermark timestamps. Tests can
	// override it for deterministic assertions; nil defaults to time.Now.
	Now func() time.Time
	// Auditor emits operational audit events for projection anomalies (e.g.
	// sequence gaps). Nil-safe: no events are emitted when unset.
	Auditor *audit.Emitter
}

// Apply routes domain events into denormalized read-model stores.
//
// The projection layer is the reason projections remain current for APIs and
// query use-cases: every event that changes campaign/world state in the domain
// gets mirrored here according to projection semantics.
func (a Applier) Apply(ctx context.Context, evt event.Event) error {
	resolvedEvent, shouldProject := a.prepareEventForProjection(evt)
	if !shouldProject {
		return nil
	}
	if err := a.routeEvent(ctx, resolvedEvent); err != nil {
		return err
	}
	if err := a.saveProjectionWatermark(ctx, resolvedEvent); err != nil {
		return fmt.Errorf("save projection watermark: %w", err)
	}
	return nil
}

// routeEvent dispatches a single event to the appropriate projection handler.
// Core event types are routed through the CoreRouter (which checks store/ID
// preconditions and auto-unmarshals payloads). Events with a non-empty SystemID
// fall through to the system adapter path; anything else is rejected.
func (a Applier) routeEvent(ctx context.Context, evt event.Event) error {
	if _, ok := coreRouter.handlers[evt.Type]; ok {
		return coreRouter.Route(a, ctx, evt)
	}
	hasSystemID := evt.SystemID != ""
	hasSystemVersion := evt.SystemVersion != ""
	if hasSystemID != hasSystemVersion {
		return fmt.Errorf("system id and version are both required but only got id=%q version=%q for event type %s",
			evt.SystemID, evt.SystemVersion, evt.Type)
	}
	if hasSystemID && hasSystemVersion {
		return a.applySystemEvent(ctx, evt)
	}
	return fmt.Errorf("unhandled projection event type: %s", evt.Type)
}

// ensureTimestamp normalizes timestamps to UTC and rejects zero values to
// preserve replay determinism — projections must never fabricate wall-clock time.
func ensureTimestamp(ts time.Time) (time.Time, error) {
	if ts.IsZero() {
		return time.Time{}, fmt.Errorf("event timestamp is required for projection")
	}
	return ts.UTC(), nil
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
