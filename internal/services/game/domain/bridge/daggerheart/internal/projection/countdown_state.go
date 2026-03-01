package projection

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ApplyCountdownUpdate validates and applies a countdown current-value update.
func ApplyCountdownUpdate(
	countdown storage.DaggerheartCountdown,
	before int,
	after int,
) (storage.DaggerheartCountdown, error) {
	if before != countdown.Current {
		return storage.DaggerheartCountdown{}, fmt.Errorf("countdown before mismatch")
	}
	if after < 0 || after > countdown.Max {
		return storage.DaggerheartCountdown{}, fmt.Errorf("countdown after must be in range 0..%d", countdown.Max)
	}
	next := countdown
	next.Current = after
	return next, nil
}
