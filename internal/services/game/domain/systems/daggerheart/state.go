package daggerheart

const (
	// SystemID identifies the Daggerheart system for system modules.
	SystemID = "daggerheart"
	// SystemVersion tracks the Daggerheart ruleset version for system modules.
	SystemVersion = "1.0.0"

	GMFearMin     = 0
	GMFearMax     = 12
	GMFearDefault = 0

	HPDefault        = 6
	HPMaxDefault     = 6
	HopeDefault      = 2
	HopeMaxDefault   = 6
	StressDefault    = 0
	StressMaxDefault = 6
	ArmorDefault     = 0
	ArmorMaxDefault  = 0
	LifeStateAlive   = "alive"
)

// SnapshotState captures campaign-level Daggerheart state.
type SnapshotState struct {
	CampaignID      string
	GMFear          int
	CharacterStates map[string]CharacterState
	AdversaryStates map[string]AdversaryState
	CountdownStates map[string]CountdownState
}

// TemporaryArmorBucket tracks temporary armor contributions on a character.
type TemporaryArmorBucket struct {
	Source   string `json:"source"`
	Duration string `json:"duration"`
	SourceID string `json:"source_id,omitempty"`
	Amount   int    `json:"amount"`
}

// CharacterState captures Daggerheart character state.
type CharacterState struct {
	CampaignID  string
	CharacterID string
	Kind        string
	HP          int
	HPMax       int
	Hope        int
	HopeMax     int
	Stress      int
	StressMax   int
	Armor       int
	ArmorMax    int
	ArmorBonus  []TemporaryArmorBucket
	LifeState   string
	Conditions  []string
}

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
	CampaignID  string
	CountdownID string
	Name        string
	Kind        string
	Current     int
	Max         int
	Direction   string
	Looping     bool
}
