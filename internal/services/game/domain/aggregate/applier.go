package aggregate

import (
	"errors"
	"sync"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// Folder folds events into aggregate state.
//
// The folder is where the domain boundary stays deterministic:
// each event type updates exactly one aggregate slice and is replayed
// identically whether during request execution or historical reconstruction.
// Named "Folder" (not "Applier") to distinguish pure state folds from
// projection.Applier, which performs side-effecting I/O writes to stores.
type Folder struct {
	// Events provides event definitions so the folder can skip audit-only
	// events that do not affect aggregate state.
	Events *event.Registry
	// SystemRegistry routes system events to their module-specific folder.
	SystemRegistry *module.Registry

	// foldSets are lazily built on first Apply to avoid dispatch into fold
	// functions that cannot possibly handle the event type.
	foldOnce         sync.Once
	campaignTypes    map[event.Type]struct{}
	sessionTypes     map[event.Type]struct{}
	actionTypes      map[event.Type]struct{}
	participantTypes map[event.Type]struct{}
	characterTypes   map[event.Type]struct{}
	inviteTypes      map[event.Type]struct{}
}

// initFoldSets builds per-fold type lookup sets from FoldHandledTypes.
func (a *Folder) initFoldSets() {
	a.foldOnce.Do(func() {
		toSet := func(types []event.Type) map[event.Type]struct{} {
			s := make(map[event.Type]struct{}, len(types))
			for _, t := range types {
				s[t] = struct{}{}
			}
			return s
		}
		a.campaignTypes = toSet(campaign.FoldHandledTypes())
		a.sessionTypes = toSet(session.FoldHandledTypes())
		a.actionTypes = toSet(action.FoldHandledTypes())
		a.participantTypes = toSet(participant.FoldHandledTypes())
		a.characterTypes = toSet(character.FoldHandledTypes())
		a.inviteTypes = toSet(invite.FoldHandledTypes())
	})
}

// FoldDispatchedTypes returns the union of all event types wired into the
// applier's fold dispatch sets. ValidateAggregateFoldDispatch uses this to
// verify that every type declared in CoreDomains().FoldHandledTypes actually
// reaches a fold function at runtime.
func (a *Folder) FoldDispatchedTypes() []event.Type {
	a.initFoldSets()
	var types []event.Type
	for _, s := range []map[event.Type]struct{}{
		a.campaignTypes,
		a.sessionTypes,
		a.actionTypes,
		a.participantTypes,
		a.characterTypes,
		a.inviteTypes,
	} {
		for t := range s {
			types = append(types, t)
		}
	}
	return types
}

// Apply applies a single event to aggregate state.
//
// The function only mutates aggregate state through fold functions so state
// transitions remain visible in one place per subdomain and replay behavior matches
// request-time behavior.
func (a *Folder) Apply(state any, evt event.Event) (any, error) {
	// Skip audit-only events: they do not affect aggregate state and should
	// not be passed to fold functions.
	if a.Events != nil {
		if def, ok := a.Events.Definition(evt.Type); ok && def.Intent == event.IntentAuditOnly {
			current, err := AssertState[State](state)
			if err != nil {
				return State{}, err
			}
			return current, nil
		}
	}

	a.initFoldSets()

	current, err := AssertState[State](state)
	if err != nil {
		return State{}, err
	}

	if _, ok := a.campaignTypes[evt.Type]; ok {
		campaignState, err := campaign.Fold(current.Campaign, evt)
		if err != nil {
			return current, err
		}
		current.Campaign = campaignState
	}

	if _, ok := a.sessionTypes[evt.Type]; ok {
		sessionState, err := session.Fold(current.Session, evt)
		if err != nil {
			return current, err
		}
		current.Session = sessionState
	}

	if _, ok := a.actionTypes[evt.Type]; ok {
		actionState, err := action.Fold(current.Action, evt)
		if err != nil {
			return current, err
		}
		current.Action = actionState
	}

	if evt.SystemID != "" || evt.SystemVersion != "" {
		if current.Systems == nil {
			current.Systems = make(map[module.Key]any)
		}
		if evt.SystemID == "" || evt.SystemVersion == "" {
			return current, errors.New("system id and version are required")
		}
		registry := a.SystemRegistry
		if registry == nil {
			return current, errors.New("system registry is required")
		}
		key := module.Key{ID: evt.SystemID, Version: evt.SystemVersion}
		systemState := current.Systems[key]
		mod := registry.Get(evt.SystemID, evt.SystemVersion)
		if mod != nil && systemState == nil {
			if factory := mod.StateFactory(); factory != nil {
				seed, err := factory.NewSnapshotState(evt.CampaignID)
				if err != nil {
					return current, err
				}
				systemState = seed
			}
		}
		updated, err := module.RouteEvent(registry, systemState, evt)
		if err != nil {
			return current, err
		}
		current.Systems[key] = updated
	}

	if evt.EntityID != "" {
		if _, ok := a.participantTypes[evt.Type]; ok {
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
		if _, ok := a.characterTypes[evt.Type]; ok {
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
		if _, ok := a.inviteTypes[evt.Type]; ok {
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
