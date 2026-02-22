package aggregate

import (
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
)

// Applier folds events into aggregate state.
//
// The applier is where the domain boundary stays deterministic:
// each event type updates exactly one aggregate slice and is replayed
// identically whether during request execution or historical reconstruction.
type Applier struct {
	// Events provides event definitions so the applier can skip audit-only
	// events that do not affect aggregate state.
	Events *event.Registry
	// SystemRegistry routes system events to their module-specific projector.
	SystemRegistry *system.Registry
}

// Apply applies a single event to aggregate state.
//
// The function only mutates aggregate state through fold functions so state
// transitions remain visible in one place per subdomain and replay behavior matches
// request-time behavior.
func (a Applier) Apply(state any, evt event.Event) (any, error) {
	// Skip audit-only events: they do not affect aggregate state and should
	// not be passed to fold functions.
	if a.Events != nil {
		if def, ok := a.Events.Definition(evt.Type); ok && def.Intent == event.IntentAuditOnly {
			if existing, ok := state.(State); ok {
				return existing, nil
			}
			if existingPtr, ok := state.(*State); ok && existingPtr != nil {
				return *existingPtr, nil
			}
			return State{}, nil
		}
	}

	current := State{}
	if existing, ok := state.(State); ok {
		current = existing
	} else if existingPtr, ok := state.(*State); ok && existingPtr != nil {
		current = *existingPtr
	}

	campaignState, err := campaign.Fold(current.Campaign, evt)
	if err != nil {
		return current, err
	}
	current.Campaign = campaignState

	sessionState, err := session.Fold(current.Session, evt)
	if err != nil {
		return current, err
	}
	current.Session = sessionState

	actionState, err := action.Fold(current.Action, evt)
	if err != nil {
		return current, err
	}
	current.Action = actionState

	if evt.SystemID != "" || evt.SystemVersion != "" {
		if current.Systems == nil {
			current.Systems = make(map[system.Key]any)
		}
		if evt.SystemID == "" || evt.SystemVersion == "" {
			return current, errors.New("system id and version are required")
		}
		registry := a.SystemRegistry
		if registry == nil {
			return current, errors.New("system registry is required")
		}
		key := system.Key{ID: evt.SystemID, Version: evt.SystemVersion}
		systemState := current.Systems[key]
		module := registry.Get(evt.SystemID, evt.SystemVersion)
		if module != nil && systemState == nil {
			if factory := module.StateFactory(); factory != nil {
				seed, err := factory.NewSnapshotState(evt.CampaignID)
				if err != nil {
					return current, err
				}
				systemState = seed
			}
		}
		updated, err := system.RouteEvent(registry, systemState, evt)
		if err != nil {
			return current, err
		}
		current.Systems[key] = updated
	}

	switch evt.EntityType {
	case "participant":
		if evt.EntityID != "" {
			if current.Participants == nil {
				current.Participants = make(map[string]participant.State)
			}
			pState := current.Participants[evt.EntityID]
			pState, err := participant.Fold(pState, evt)
			if err != nil {
				return current, err
			}
			current.Participants[evt.EntityID] = pState
		}
	case "character":
		if evt.EntityID != "" {
			if current.Characters == nil {
				current.Characters = make(map[string]character.State)
			}
			cState := current.Characters[evt.EntityID]
			cState, err := character.Fold(cState, evt)
			if err != nil {
				return current, err
			}
			current.Characters[evt.EntityID] = cState
		}
	case "invite":
		if evt.EntityID != "" {
			if current.Invites == nil {
				current.Invites = make(map[string]invite.State)
			}
			iState := current.Invites[evt.EntityID]
			iState, err := invite.Fold(iState, evt)
			if err != nil {
				return current, err
			}
			current.Invites[evt.EntityID] = iState
		}
	}

	return current, nil
}
