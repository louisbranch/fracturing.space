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
