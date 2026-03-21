package snapstate

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
)

type TemporaryArmorBucket = mechanics.TemporaryArmorBucket
type CharacterState = mechanics.CharacterState

// SnapshotState captures campaign-level Daggerheart state.
type SnapshotState struct {
	CampaignID              ids.CampaignID
	GMFear                  int
	CharacterProfiles       map[ids.CharacterID]CharacterProfile
	CharacterStates         map[ids.CharacterID]CharacterState
	CharacterClassStates    map[ids.CharacterID]CharacterClassState
	CharacterSubclassStates map[ids.CharacterID]CharacterSubclassState
	CharacterCompanions     map[ids.CharacterID]CharacterCompanionState
	AdversaryStates         map[ids.AdversaryID]AdversaryState
	EnvironmentStates       map[ids.EnvironmentEntityID]EnvironmentEntityState
	CountdownStates         map[ids.CountdownID]CountdownState
}

// EnsureMaps initializes nil maps on SnapshotState. Call this for
// deserialized states where maps may be nil (e.g. legacy snapshots loaded from
// storage). NewSnapshotState already returns initialized maps, so this is only
// needed for states not created through the factory.
func (s *SnapshotState) EnsureMaps() {
	if s.CharacterProfiles == nil {
		s.CharacterProfiles = make(map[ids.CharacterID]CharacterProfile)
	}
	if s.CharacterStates == nil {
		s.CharacterStates = make(map[ids.CharacterID]CharacterState)
	}
	if s.CharacterClassStates == nil {
		s.CharacterClassStates = make(map[ids.CharacterID]CharacterClassState)
	}
	if s.CharacterSubclassStates == nil {
		s.CharacterSubclassStates = make(map[ids.CharacterID]CharacterSubclassState)
	}
	if s.CharacterCompanions == nil {
		s.CharacterCompanions = make(map[ids.CharacterID]CharacterCompanionState)
	}
	if s.AdversaryStates == nil {
		s.AdversaryStates = make(map[ids.AdversaryID]AdversaryState)
	}
	if s.EnvironmentStates == nil {
		s.EnvironmentStates = make(map[ids.EnvironmentEntityID]EnvironmentEntityState)
	}
	if s.CountdownStates == nil {
		s.CountdownStates = make(map[ids.CountdownID]CountdownState)
	}
}

// AdversaryState captures Daggerheart adversary state for aggregate projections.
type AdversaryState struct {
	CampaignID        ids.CampaignID
	AdversaryID       ids.AdversaryID
	AdversaryEntryID  string
	Name              string
	Kind              string
	SessionID         ids.SessionID
	SceneID           ids.SceneID
	Notes             string
	HP                int
	HPMax             int
	Stress            int
	StressMax         int
	Evasion           int
	Major             int
	Severe            int
	Armor             int
	Conditions        []string
	FeatureStates     []rules.AdversaryFeatureState
	PendingExperience *rules.AdversaryPendingExperience
	SpotlightGateID   ids.GateID
	SpotlightCount    int
}

// EnvironmentEntityState captures instantiated environment state for
// aggregate projections and GM move validation.
type EnvironmentEntityState struct {
	CampaignID          ids.CampaignID
	EnvironmentEntityID ids.EnvironmentEntityID
	EnvironmentID       string
	Name                string
	Type                string
	Tier                int
	Difficulty          int
	SessionID           ids.SessionID
	SceneID             ids.SceneID
	Notes               string
}

// CountdownState captures Daggerheart countdown state for aggregate projections.
type CountdownState struct {
	CampaignID        ids.CampaignID
	CountdownID       ids.CountdownID
	Name              string
	Kind              string
	Current           int
	Max               int
	Direction         string
	Looping           bool
	Variant           string
	TriggerEventType  string
	LinkedCountdownID ids.CountdownID
}

// SnapshotOrDefault extracts a SnapshotState from the state value for the
// decider path. Returns (state, true) for known types, or a factory-aligned
// default with initialized maps on nil/unknown. Using the same defaults as
// NewSnapshotState ensures decider defaults never silently diverge from the
// factory.
func SnapshotOrDefault(state any) (SnapshotState, bool) {
	switch typed := state.(type) {
	case SnapshotState:
		typed.EnsureMaps()
		return typed, true
	case *SnapshotState:
		if typed != nil {
			typed.EnsureMaps()
			return *typed, true
		}
	}
	s := SnapshotState{GMFear: GMFearDefault}
	s.EnsureMaps()
	return s, false
}

// AssertSnapshotState converts untyped state to *SnapshotState for the fold
// router. It handles nil (first event), value types, and pointer types.
// EnsureMaps is called on the result so deserialized states with nil maps
// are safe to use immediately.
func AssertSnapshotState(state any) (*SnapshotState, error) {
	var s *SnapshotState
	switch typed := state.(type) {
	case nil:
		v := SnapshotState{GMFear: GMFearDefault}
		s = &v
	case SnapshotState:
		s = &typed
	case *SnapshotState:
		if typed != nil {
			s = typed
		} else {
			v := SnapshotState{GMFear: GMFearDefault}
			s = &v
		}
	default:
		return nil, fmt.Errorf("unsupported state type %T", state)
	}
	s.EnsureMaps()
	return s, nil
}

// AppendUnique appends a string value to a slice only if it is not already present.
func AppendUnique(slice []string, value string) []string {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

// NewSnapshotState creates initial snapshot state for the given campaign.
// GMFear starts neutral here; first-session bootstrap owns the actual initial
// Fear seed once campaign readiness resolves the created-PC roster.
func NewSnapshotState(campaignID ids.CampaignID) SnapshotState {
	return SnapshotState{
		CampaignID:              ids.CampaignID(strings.TrimSpace(string(campaignID))),
		GMFear:                  GMFearDefault,
		CharacterProfiles:       make(map[ids.CharacterID]CharacterProfile),
		CharacterStates:         make(map[ids.CharacterID]CharacterState),
		CharacterClassStates:    make(map[ids.CharacterID]CharacterClassState),
		CharacterSubclassStates: make(map[ids.CharacterID]CharacterSubclassState),
		CharacterCompanions:     make(map[ids.CharacterID]CharacterCompanionState),
		AdversaryStates:         make(map[ids.AdversaryID]AdversaryState),
		EnvironmentStates:       make(map[ids.EnvironmentEntityID]EnvironmentEntityState),
		CountdownStates:         make(map[ids.CountdownID]CountdownState),
	}
}
