package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/mechanics"

// Compatibility constants re-exported from internal/mechanics for gRPC
// transport and snapshot packages.
//
// Removal criteria: these can be removed when the transport packages under
// api/grpc/systems/daggerheart/ and api/grpc/game/ (18+ consumers) either
// import internal/mechanics directly or define their own boundary constants.
// Until then, this file is the stable public interface for game-mechanic
// constants used outside the domain layer.
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
