package aggregate

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// foldEntry describes how a set of event types maps to a fold function that
// updates one slice of aggregate state. Each entry is either direct (single
// field on State) or entity-keyed (map on State keyed by EntityID).
type foldEntry struct {
	// types returns the event types handled by this fold entry.
	types func() []event.Type
	// fold applies a single event to a sub-state and writes the result back
	// into the aggregate state. Entity-keyed entries receive the EntityID from
	// the event envelope.
	fold func(state *State, evt event.Event) error
}

// foldEntityKeyed is a generic helper for entity-keyed fold entries. It
// validates the EntityID, lazily initializes the map, looks up the sub-state,
// calls the domain fold, and writes back the result.
func foldEntityKeyed[K ~string, S any](
	m *map[K]S,
	evt event.Event,
	domainName string,
	fold func(S, event.Event) (S, error),
) error {
	if evt.EntityID == "" {
		return fmt.Errorf("%s fold requires EntityID but got empty for %s", domainName, evt.Type)
	}
	if *m == nil {
		*m = make(map[K]S)
	}
	key := K(evt.EntityID)
	sub := (*m)[key]
	updated, err := fold(sub, evt)
	if err != nil {
		return err
	}
	(*m)[key] = updated
	return nil
}

// coreFoldEntries returns the declarative fold dispatch table for all core
// domains. The authoritative core-domain registration inventory lives in
// CoreDomainRegistrations so engine startup and aggregate replay stay aligned.
//
// Entity-keyed entries (participant, character) perform an EntityID
// presence check at fold time. This is intentional defense-in-depth:
// ValidateEntityKeyedAddressing catches missing EntityIDs at startup, but the
// runtime check guards against regression if a new event type is registered
// without the startup validator being updated. Both checks are cheap and
// should be preserved.
func coreFoldEntries() []foldEntry {
	registrations := CoreDomainRegistrations()
	entries := make([]foldEntry, 0, len(registrations))
	for _, registration := range registrations {
		entry := registration
		entries = append(entries, foldEntry{
			types: entry.FoldHandledTypes,
			fold:  entry.Fold,
		})
	}
	return entries
}

// extractSceneID reads the scene_id field from any scene event payload.
func extractSceneID(evt event.Event) (ids.SceneID, error) {
	var envelope struct {
		SceneID string `json:"scene_id"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &envelope); err != nil {
		return "", fmt.Errorf("extract scene_id from %s payload: %w", evt.Type, err)
	}
	if envelope.SceneID == "" {
		return "", fmt.Errorf("scene_id is empty in %s payload", evt.Type)
	}
	return ids.SceneID(envelope.SceneID), nil
}
