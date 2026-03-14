package command

import "time"

// NowFunc returns the provided time function or time.Now as a default.
// This normalizes the nil-check pattern used across all deciders so each
// aggregate does not need to repeat the guard inline.
func NowFunc(now func() time.Time) func() time.Time {
	if now == nil {
		return time.Now
	}
	return now
}
