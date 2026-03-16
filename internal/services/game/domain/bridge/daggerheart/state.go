package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

const (
	// SystemID identifies the Daggerheart system for system modules.
	SystemID = "daggerheart"
	// SystemVersion tracks the Daggerheart ruleset version for system modules.
	SystemVersion = "1.0.0"

	GMFearMin = 0
	GMFearMax = 12
	// GMFearDefault is the neutral pre-activation value for synthetic or newly
	// created snapshots. First-session bootstrap seeds the campaign's actual
	// starting Fear from the count of created PCs when the campaign becomes
	// active.
	GMFearDefault = 0

	HPDefault        = mechanics.HPDefault
	HPMaxDefault     = mechanics.HPMaxDefault
	HopeDefault      = mechanics.HopeDefault
	HopeMaxDefault   = mechanics.HopeMaxDefault
	StressDefault    = mechanics.StressDefault
	StressMaxDefault = mechanics.StressMaxDefault
	ArmorDefault     = mechanics.ArmorDefault
	ArmorMaxDefault  = mechanics.ArmorMaxDefault
	LifeStateAlive   = mechanics.LifeStateAlive
)

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

type TemporaryArmorBucket = mechanics.TemporaryArmorBucket
type CharacterState = mechanics.CharacterState

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
	FeatureStates     []AdversaryFeatureState
	PendingExperience *AdversaryPendingExperience
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
