package app

import "github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"

// RequireUserID validates and returns a trimmed user ID, or returns an
// unauthorized error if it is blank.
func RequireUserID(userID string) (string, error) {
	return userid.Require(userID)
}
