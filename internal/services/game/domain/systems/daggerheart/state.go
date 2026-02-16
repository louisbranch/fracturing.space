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
	CampaignID string
	GMFear     int
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
}
