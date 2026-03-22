package projection

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// ApplyCountdownUpdate applies a countdown remaining-value update. The
// projection unconditionally trusts the event stream.
func ApplyCountdownUpdate(
	countdown projectionstore.DaggerheartCountdown,
	after int,
) (projectionstore.DaggerheartCountdown, error) {
	if after < 0 || after > countdown.StartingValue {
		return projectionstore.DaggerheartCountdown{}, fmt.Errorf("countdown after must be in range 0..%d", countdown.StartingValue)
	}
	next := countdown
	next.RemainingValue = after
	return next, nil
}
