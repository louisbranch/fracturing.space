package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/mechanics"

// Compatibility constants re-exported from internal/mechanics for gRPC
// application mappings. Remove when gRPC transport packages import
// internal/mechanics directly or define their own boundary constants.
const (
	HPMin        = mechanics.HPMin
	HPMaxCap     = mechanics.HPMaxCap
	HopeMin      = mechanics.HopeMin
	HopeMax      = mechanics.HopeMax
	StressMin    = mechanics.StressMin
	StressMaxCap = mechanics.StressMaxCap
	ArmorMin     = mechanics.ArmorMin
	ArmorMaxCap  = mechanics.ArmorMaxCap

	LifeStateUnconscious  = mechanics.LifeStateUnconscious
	LifeStateBlazeOfGlory = mechanics.LifeStateBlazeOfGlory
	LifeStateDead         = mechanics.LifeStateDead
)
