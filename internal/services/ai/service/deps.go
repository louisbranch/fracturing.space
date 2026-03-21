package service

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// Clock returns the current time. Used as a dependency for deterministic tests.
type Clock = func() time.Time

// IDGenerator returns a new unique identifier. Used as a dependency for
// deterministic tests.
type IDGenerator = func() (string, error)

func withDefaultClock(clock Clock) Clock {
	if clock != nil {
		return clock
	}
	return time.Now
}

func withDefaultIDGenerator(gen IDGenerator) IDGenerator {
	if gen != nil {
		return gen
	}
	return id.NewID
}
