package userid

import (
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// Normalize returns a canonical user-id value for request and transport seams.
func Normalize(userID string) string {
	return strings.TrimSpace(userID)
}

// Require validates and returns a trimmed user id.
func Require(userID string) (string, error) {
	resolvedUserID := Normalize(userID)
	if resolvedUserID == "" {
		return "", apperrors.EK(apperrors.KindUnauthorized, "error.web.message.user_id_is_required", "user id is required")
	}
	return resolvedUserID, nil
}
