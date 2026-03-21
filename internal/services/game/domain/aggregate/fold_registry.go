package aggregate

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
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
// domains. Adding a new core domain requires only adding an entry here.
//
// Entity-keyed entries (participant, character) perform an EntityID
// presence check at fold time. This is intentional defense-in-depth:
// ValidateEntityKeyedAddressing catches missing EntityIDs at startup, but the
// runtime check guards against regression if a new event type is registered
// without the startup validator being updated. Both checks are cheap and
// should be preserved.
func coreFoldEntries() []foldEntry {
	return []foldEntry{
		{
			types: campaign.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				updated, err := campaign.Fold(state.Campaign, evt)
				if err != nil {
					return err
				}
				state.Campaign = updated
				return nil
			},
		},
		{
			types: session.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				updated, err := session.Fold(state.Session, evt)
				if err != nil {
					return err
				}
				state.Session = updated
				return nil
			},
		},
		{
			types: action.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				updated, err := action.Fold(state.Action, evt)
				if err != nil {
					return err
				}
				state.Action = updated
				return nil
			},
		},
		{
			types: participant.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				return foldEntityKeyed(&state.Participants, evt, "participant", participant.Fold)
			},
		},
		{
			types: character.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				return foldEntityKeyed(&state.Characters, evt, "character", character.Fold)
			},
		},
		{
			types: scene.FoldHandledTypes,
			fold:  foldScene,
		},
	}
}

// foldScene routes scene events to the correct scene state entry.
//
// Scene events use different EntityID conventions depending on event type:
// most use the SceneID as EntityID, but gate events use the GateID. To
// handle both patterns uniformly, foldScene extracts the scene_id from
// the event payload.
func foldScene(state *State, evt event.Event) error {
	sceneID, err := extractSceneID(evt)
	if err != nil {
		return fmt.Errorf("scene fold: %w", err)
	}
	if state.Scenes == nil {
		state.Scenes = make(map[ids.SceneID]scene.State)
	}
	sub := state.Scenes[sceneID]
	updated, err := scene.Fold(sub, evt)
	if err != nil {
		return err
	}
	state.Scenes[sceneID] = updated
	return nil
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
