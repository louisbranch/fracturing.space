package snapstate

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
)

const (
	// SystemID identifies the Daggerheart system for system modules.
	SystemID = "daggerheart"
	// SystemVersion tracks the Daggerheart ruleset version for system modules.
	SystemVersion = "1.0.0"

	GMFearMin = rules.GMFearMin
	GMFearMax = rules.GMFearMax
	// GMFearDefault is the neutral pre-activation value for synthetic or newly
	// created snapshots. First-session bootstrap seeds the campaign's actual
	// starting Fear from the count of created PCs when the campaign becomes
	// active.
	GMFearDefault = rules.GMFearDefault

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
