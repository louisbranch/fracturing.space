package coreprojection

import (
	"errors"
	"strings"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func isParticipantUserConflict(err error) bool {
	if !isConstraintLikeError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_participants_campaign_user") ||
		(strings.Contains(message, "participant") && strings.Contains(message, "user_id"))
}

func isParticipantClaimConflict(err error) bool {
	if !isConstraintLikeError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "participant_claims") ||
		strings.Contains(message, "idx_participant_claims")
}

func isConstraintLikeError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT || code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}

func isConstraintError(err error) bool {
	return isConstraintLikeError(err)
}
