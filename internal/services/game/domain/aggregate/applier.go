package aggregate

import (
	"errors"

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
	// SystemRegistry routes system events to their module-specific projector.
	SystemRegistry *system.Registry
}

// Apply applies a single event to aggregate state.
//
// The function only mutates aggregate state through fold functions so state
// transitions remain visible in one place per subdomain and replay behavior matches
// request-time behavior.
func (a Applier) Apply(state any, evt event.Event) (any, error) {
	current := State{}
	if existing, ok := state.(State); ok {
		current = existing
	} else if existingPtr, ok := state.(*State); ok && existingPtr != nil {
		current = *existingPtr
	}

	current.Campaign = campaign.Fold(current.Campaign, evt)
	current.Session = session.Fold(current.Session, evt)

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
			participantState := current.Participants[evt.EntityID]
			participantState = participant.Fold(participantState, evt)
			current.Participants[evt.EntityID] = participantState
		}
	case "character":
		if evt.EntityID != "" {
			if current.Characters == nil {
				current.Characters = make(map[string]character.State)
			}
			characterState := current.Characters[evt.EntityID]
			characterState = character.Fold(characterState, evt)
			current.Characters[evt.EntityID] = characterState
		}
	case "invite":
		if evt.EntityID != "" {
			if current.Invites == nil {
				current.Invites = make(map[string]invite.State)
			}
			inviteState := current.Invites[evt.EntityID]
			inviteState = invite.Fold(inviteState, evt)
			current.Invites[evt.EntityID] = inviteState
		}
	}

	return current, nil
}
