package state

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

type TemporaryArmorBucket = mechanics.TemporaryArmorBucket
type CharacterState = mechanics.CharacterState
type CountdownState = CampaignCountdownState

// SnapshotState captures campaign-level Daggerheart state.
type SnapshotState struct {
	CampaignID              ids.CampaignID
	GMFear                  int
	CharacterProfiles       map[ids.CharacterID]CharacterProfile
	CharacterStates         map[ids.CharacterID]CharacterState
	CharacterClassStates    map[ids.CharacterID]CharacterClassState
	CharacterSubclassStates map[ids.CharacterID]CharacterSubclassState
	CharacterCompanions     map[ids.CharacterID]CharacterCompanionState
	CharacterStatModifiers  map[ids.CharacterID][]rules.StatModifierState
	AdversaryStates         map[dhids.AdversaryID]AdversaryState
	EnvironmentStates       map[dhids.EnvironmentEntityID]EnvironmentEntityState
	SceneCountdownStates    map[dhids.CountdownID]SceneCountdownState
	CampaignCountdownStates map[dhids.CountdownID]CampaignCountdownState
	CountdownStates         map[dhids.CountdownID]CampaignCountdownState
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
	if s.CharacterStatModifiers == nil {
		s.CharacterStatModifiers = make(map[ids.CharacterID][]rules.StatModifierState)
	}
	if s.AdversaryStates == nil {
		s.AdversaryStates = make(map[dhids.AdversaryID]AdversaryState)
	}
	if s.EnvironmentStates == nil {
		s.EnvironmentStates = make(map[dhids.EnvironmentEntityID]EnvironmentEntityState)
	}
	if s.SceneCountdownStates == nil {
		s.SceneCountdownStates = make(map[dhids.CountdownID]SceneCountdownState)
	}
	if s.CampaignCountdownStates == nil {
		s.CampaignCountdownStates = make(map[dhids.CountdownID]CampaignCountdownState)
	}
	if s.CountdownStates == nil {
		s.CountdownStates = s.CampaignCountdownStates
	}
	s.CountdownStates = s.CampaignCountdownStates
}

// AdversaryState captures Daggerheart adversary state for aggregate projections.
type AdversaryState struct {
	CampaignID        ids.CampaignID
	AdversaryID       dhids.AdversaryID
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
	EnvironmentEntityID dhids.EnvironmentEntityID
	EnvironmentID       string
	Name                string
	Type                string
	Tier                int
	Difficulty          int
	SessionID           ids.SessionID
	SceneID             ids.SceneID
	Notes               string
}

// SceneCountdownState captures scene-owned Daggerheart countdown state for
// spotlight/combat board projections.
type SceneCountdownState struct {
	CampaignID        ids.CampaignID
	SessionID         ids.SessionID
	SceneID           ids.SceneID
	CountdownID       dhids.CountdownID
	Name              string
	Tone              string
	AdvancementPolicy string
	StartingValue     int
	RemainingValue    int
	LoopBehavior      string
	Status            string
	LinkedCountdownID dhids.CountdownID
	StartingRoll      *rules.CountdownStartingRoll
}

// CampaignCountdownState captures campaign-owned Daggerheart countdown state
// for persistent clocks such as rest/project progress.
type CampaignCountdownState struct {
	CampaignID        ids.CampaignID
	CountdownID       dhids.CountdownID
	Name              string
	Tone              string
	AdvancementPolicy string
	StartingValue     int
	RemainingValue    int
	LoopBehavior      string
	Status            string
	LinkedCountdownID dhids.CountdownID
	StartingRoll      *rules.CountdownStartingRoll
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
	return defaultSnapshotState(), false
}

// SnapshotOrDefaultIfAbsent converts untyped state to SnapshotState while
// treating nil or nil pointers as the factory-aligned default.
//
// Unlike SnapshotOrDefault, unsupported types remain errors so readiness and
// bootstrap paths can distinguish absent state from invalid state wiring.
func SnapshotOrDefaultIfAbsent(state any) (SnapshotState, error) {
	switch typed := state.(type) {
	case nil:
		return defaultSnapshotState(), nil
	case SnapshotState:
		typed.EnsureMaps()
		return typed, nil
	case *SnapshotState:
		if typed != nil {
			typed.EnsureMaps()
			return *typed, nil
		}
		return defaultSnapshotState(), nil
	default:
		return SnapshotState{}, fmt.Errorf("unsupported state type %T", state)
	}
}

// RequireSnapshotState converts untyped state to *SnapshotState for the fold
// router. Nil inputs are rejected because write-path folding must receive
// state from the module StateFactory rather than silently fabricating it.
func RequireSnapshotState(state any) (*SnapshotState, error) {
	switch typed := state.(type) {
	case nil:
		return nil, fmt.Errorf("unsupported state type %T", state)
	case SnapshotState:
		typed.EnsureMaps()
		return &typed, nil
	case *SnapshotState:
		if typed == nil {
			return nil, fmt.Errorf("unsupported state type %T", state)
		}
		typed.EnsureMaps()
		return typed, nil
	default:
		return nil, fmt.Errorf("unsupported state type %T", state)
	}
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
	state := SnapshotState{
		CampaignID:              ids.CampaignID(strings.TrimSpace(string(campaignID))),
		GMFear:                  GMFearDefault,
		CharacterProfiles:       make(map[ids.CharacterID]CharacterProfile),
		CharacterStates:         make(map[ids.CharacterID]CharacterState),
		CharacterClassStates:    make(map[ids.CharacterID]CharacterClassState),
		CharacterSubclassStates: make(map[ids.CharacterID]CharacterSubclassState),
		CharacterCompanions:     make(map[ids.CharacterID]CharacterCompanionState),
		CharacterStatModifiers:  make(map[ids.CharacterID][]rules.StatModifierState),
		AdversaryStates:         make(map[dhids.AdversaryID]AdversaryState),
		EnvironmentStates:       make(map[dhids.EnvironmentEntityID]EnvironmentEntityState),
		SceneCountdownStates:    make(map[dhids.CountdownID]SceneCountdownState),
		CampaignCountdownStates: make(map[dhids.CountdownID]CampaignCountdownState),
		CountdownStates:         nil,
	}
	state.CountdownStates = state.CampaignCountdownStates
	return state
}

func defaultSnapshotState() SnapshotState {
	return NewSnapshotState("")
}
