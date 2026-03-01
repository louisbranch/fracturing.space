package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/mechanics"

// Compatibility constants used by gRPC application mappings during migration.
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
