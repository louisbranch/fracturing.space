package generator

// Preset defines a named configuration for scenario generation.
type Preset string

const (
	// PresetDemo creates a rich single campaign with full party and active session.
	PresetDemo Preset = "demo"

	// PresetVariety creates multiple campaigns across all statuses and GM modes.
	PresetVariety Preset = "variety"

	// PresetSessionHeavy creates few campaigns with many sessions and events.
	PresetSessionHeavy Preset = "session-heavy"

	// PresetStressTest creates many minimal campaigns for load testing.
	PresetStressTest Preset = "stress-test"
)

// PresetConfig holds the generation parameters for a preset.
type PresetConfig struct {
	// Number of campaigns to generate
	Campaigns int

	// Participants per campaign (min, max)
	ParticipantsMin int
	ParticipantsMax int

	// Characters per campaign (min, max)
	CharactersMin int
	CharactersMax int

	// Sessions per campaign (min, max)
	SessionsMin int
	SessionsMax int

	// Events per session (min, max)
	EventsMin int
	EventsMax int

	// Whether to vary campaign statuses
	VaryStatuses bool

	// Whether to vary GM modes
	VaryGmModes bool

	// Whether to include ended sessions
	IncludeEndedSessions bool
}

// GetPresetConfig returns the configuration for a preset.
func GetPresetConfig(preset Preset) PresetConfig {
	switch preset {
	case PresetDemo:
		return PresetConfig{
			Campaigns:            1,
			ParticipantsMin:      4,
			ParticipantsMax:      4,
			CharactersMin:        5,
			CharactersMax:        6,
			SessionsMin:          1,
			SessionsMax:          1,
			EventsMin:            10,
			EventsMax:            20,
			VaryStatuses:         false,
			VaryGmModes:          false,
			IncludeEndedSessions: false,
		}

	case PresetVariety:
		return PresetConfig{
			Campaigns:            8,
			ParticipantsMin:      2,
			ParticipantsMax:      4,
			CharactersMin:        2,
			CharactersMax:        5,
			SessionsMin:          0,
			SessionsMax:          2,
			EventsMin:            0,
			EventsMax:            5,
			VaryStatuses:         true,
			VaryGmModes:          true,
			IncludeEndedSessions: true,
		}

	case PresetSessionHeavy:
		return PresetConfig{
			Campaigns:            2,
			ParticipantsMin:      4,
			ParticipantsMax:      5,
			CharactersMin:        5,
			CharactersMax:        6,
			SessionsMin:          5,
			SessionsMax:          5,
			EventsMin:            10,
			EventsMax:            15,
			VaryStatuses:         false,
			VaryGmModes:          false,
			IncludeEndedSessions: true,
		}

	case PresetStressTest:
		return PresetConfig{
			Campaigns:            50,
			ParticipantsMin:      1,
			ParticipantsMax:      2,
			CharactersMin:        1,
			CharactersMax:        2,
			SessionsMin:          1,
			SessionsMax:          1,
			EventsMin:            2,
			EventsMax:            2,
			VaryStatuses:         true,
			VaryGmModes:          true,
			IncludeEndedSessions: false,
		}

	default:
		// Default to demo preset
		return GetPresetConfig(PresetDemo)
	}
}
