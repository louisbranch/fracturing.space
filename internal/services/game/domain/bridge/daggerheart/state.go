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

	GMFearMin     = 0
	GMFearMax     = 12
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
	CampaignID             ids.CampaignID
	GMFear                 int
	DowntimeMovesSinceRest int
	CharacterProfiles      map[ids.CharacterID]CharacterProfile
	CharacterStates        map[ids.CharacterID]CharacterState
	AdversaryStates        map[ids.AdversaryID]AdversaryState
	CountdownStates        map[ids.CountdownID]CountdownState
}

type TemporaryArmorBucket = mechanics.TemporaryArmorBucket
type CharacterState = mechanics.CharacterState

// AdversaryState captures Daggerheart adversary state for aggregate projections.
type AdversaryState struct {
	CampaignID  ids.CampaignID
	AdversaryID ids.AdversaryID
	Name        string
	Kind        string
	SessionID   ids.SessionID
	Notes       string
	HP          int
	HPMax       int
	Stress      int
	StressMax   int
	Evasion     int
	Major       int
	Severe      int
	Armor       int
	Conditions  []string
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
	if s.AdversaryStates == nil {
		s.AdversaryStates = make(map[ids.AdversaryID]AdversaryState)
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
