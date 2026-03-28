package aggregate

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/coredomain"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// CoreDomainRegistration bundles the built-in core-domain registration hooks
// plus the aggregate fold adapter for that domain.
//
// This keeps the authoritative core-domain inventory next to the aggregate fold
// wiring that every runtime path must agree on. Engine startup and validators
// consume this inventory instead of maintaining a parallel hardcoded list.
type CoreDomainRegistration struct {
	coredomain.Contracts
	Fold func(*State, event.Event) error
}

// Name returns a human-readable label for diagnostics and startup errors.
func (d CoreDomainRegistration) Name() string { return d.DomainName }

// registration binds the shared contract metadata to a concrete aggregate fold
// adapter for one built-in core domain.
func registration(contracts coredomain.Contracts, fold func(*State, event.Event) error) CoreDomainRegistration {
	return CoreDomainRegistration{
		Contracts: contracts,
		Fold:      fold,
	}
}

// foldDirectState adapts a core domain that owns one direct aggregate state
// field, such as campaign or session.
func foldDirectState[S any](
	state *State,
	selectState func(*State) *S,
	evt event.Event,
	fold func(S, event.Event) (S, error),
) error {
	current := selectState(state)
	updated, err := fold(*current, evt)
	if err != nil {
		return err
	}
	*current = updated
	return nil
}

// foldKeyedState adapts a core domain that owns a keyed aggregate state map.
// The key extractor can be the event envelope EntityID or a payload-derived key.
func foldKeyedState[K comparable, S any](
	state *State,
	selectStateMap func(*State) *map[K]S,
	keyForEvent func(event.Event) (K, error),
	evt event.Event,
	fold func(S, event.Event) (S, error),
) error {
	key, err := keyForEvent(evt)
	if err != nil {
		return err
	}
	stateMap := selectStateMap(state)
	if *stateMap == nil {
		*stateMap = make(map[K]S)
	}
	sub := (*stateMap)[key]
	updated, err := fold(sub, evt)
	if err != nil {
		return err
	}
	(*stateMap)[key] = updated
	return nil
}

// directCoreDomainRegistration builds a core-domain registration for aggregate
// state that lives in one direct field on State.
func directCoreDomainRegistration[S any](
	contracts coredomain.Contracts,
	selectState func(*State) *S,
	fold func(S, event.Event) (S, error),
) CoreDomainRegistration {
	return registration(contracts, func(state *State, evt event.Event) error {
		return foldDirectState(state, selectState, evt, fold)
	})
}

// entityKeyedCoreDomainRegistration builds a core-domain registration for
// aggregate state keyed by the event envelope EntityID.
func entityKeyedCoreDomainRegistration[K ~string, S any](
	contracts coredomain.Contracts,
	selectStateMap func(*State) *map[K]S,
	fold func(S, event.Event) (S, error),
) CoreDomainRegistration {
	return registration(contracts, func(state *State, evt event.Event) error {
		return foldEntityKeyed(selectStateMap(state), evt, contracts.DomainName, fold)
	})
}

// keyedCoreDomainRegistration builds a core-domain registration for aggregate
// state keyed by a domain-specific lookup such as a payload-derived scene ID.
func keyedCoreDomainRegistration[K comparable, S any](
	contracts coredomain.Contracts,
	selectStateMap func(*State) *map[K]S,
	keyForEvent func(event.Event) (K, error),
	fold func(S, event.Event) (S, error),
) CoreDomainRegistration {
	return registration(contracts, func(state *State, evt event.Event) error {
		return foldKeyedState(state, selectStateMap, keyForEvent, evt, fold)
	})
}

// CoreDomainRegistrations returns the built-in core-domain registration
// inventory used by aggregate replay and engine startup validation.
func CoreDomainRegistrations() []CoreDomainRegistration {
	return append([]CoreDomainRegistration(nil), builtInCoreDomainRegistrations...)
}

// builtInCoreDomainRegistrations is the canonical inventory of core domain
// registrations used by aggregate replay and engine startup validation.
//
// Three registration patterns are used:
//   - directCoreDomainRegistration: for domains that own one direct field on
//     aggregate State (e.g. campaign, session, action). Use when the domain has
//     exactly one state instance per campaign aggregate.
//   - entityKeyedCoreDomainRegistration: for domains keyed by the event envelope
//     EntityID (e.g. participant, character). Use when each entity instance is
//     identified by the event's EntityID field.
//   - keyedCoreDomainRegistration: for domains keyed by a payload-derived value
//     rather than EntityID (e.g. scene, keyed by SceneID extracted from the
//     event payload). Use when the state map key differs from EntityID.
var builtInCoreDomainRegistrations = []CoreDomainRegistration{
	directCoreDomainRegistration(campaign.CoreDomainContracts(), func(state *State) *campaign.State { return &state.Campaign }, campaign.Fold),
	directCoreDomainRegistration(session.CoreDomainContracts(), func(state *State) *session.State { return &state.Session }, session.Fold),
	directCoreDomainRegistration(action.CoreDomainContracts(), func(state *State) *action.State { return &state.Action }, action.Fold),
	entityKeyedCoreDomainRegistration(participant.CoreDomainContracts(), func(state *State) *map[ids.ParticipantID]participant.State { return &state.Participants }, participant.Fold),
	entityKeyedCoreDomainRegistration(character.CoreDomainContracts(), func(state *State) *map[ids.CharacterID]character.State { return &state.Characters }, character.Fold),
	keyedCoreDomainRegistration(scene.CoreDomainContracts(), func(state *State) *map[ids.SceneID]scene.State { return &state.Scenes }, extractSceneID, scene.Fold),
}
