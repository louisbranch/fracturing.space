package command

import "time"

// RequireNowFunc validates and returns the provided time function.
// It panics on nil because a missing clock is always a programming error:
// the engine handler boundary is responsible for providing a valid clock to
// all downstream deciders.
func RequireNowFunc(now func() time.Time) func() time.Time {
	if now == nil {
		panic("command: now function must not be nil")
	}
	return now
}
