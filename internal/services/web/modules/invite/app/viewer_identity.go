package app

import "github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"

// RequireUserID validates and returns a normalized viewer user ID.
func RequireUserID(userID string) (string, error) {
	return userid.Require(userID)
}
