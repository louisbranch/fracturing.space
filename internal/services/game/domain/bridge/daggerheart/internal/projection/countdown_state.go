package projection

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ApplyCountdownUpdate applies a countdown current-value update.
// The projection unconditionally trusts the event stream.
func ApplyCountdownUpdate(
	countdown storage.DaggerheartCountdown,
	after int,
) (storage.DaggerheartCountdown, error) {
	if after < 0 || after > countdown.Max {
		return storage.DaggerheartCountdown{}, fmt.Errorf("countdown after must be in range 0..%d", countdown.Max)
	}
	next := countdown
	next.Current = after
	return next, nil
}
