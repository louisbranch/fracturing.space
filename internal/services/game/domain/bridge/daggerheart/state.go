package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/mechanics"

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
	CampaignID             string
	GMFear                 int
	DowntimeMovesSinceRest int
	CharacterStates        map[string]CharacterState
	AdversaryStates        map[string]AdversaryState
	CountdownStates        map[string]CountdownState
}

type TemporaryArmorBucket = mechanics.TemporaryArmorBucket
type CharacterState = mechanics.CharacterState

// AdversaryState captures Daggerheart adversary state for aggregate projections.
type AdversaryState struct {
	CampaignID  string
	AdversaryID string
	Name        string
	Kind        string
	SessionID   string
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
	if s.CharacterStates == nil {
		s.CharacterStates = make(map[string]CharacterState)
	}
	if s.AdversaryStates == nil {
		s.AdversaryStates = make(map[string]AdversaryState)
	}
	if s.CountdownStates == nil {
		s.CountdownStates = make(map[string]CountdownState)
	}
}

// CountdownState captures Daggerheart countdown state for aggregate projections.
type CountdownState struct {
	CampaignID        string
	CountdownID       string
	Name              string
	Kind              string
	Current           int
	Max               int
	Direction         string
	Looping           bool
	Variant           string
	TriggerEventType  string
	LinkedCountdownID string
}
