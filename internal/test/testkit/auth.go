package testkit

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/auth/authtest"
)

// CreateAuthUser delegates auth test bootstrap to the auth service boundary so
// generic testkit helpers do not own auth storage details directly.
func CreateAuthUser(t *testing.T, authAddr, username string) string {
	t.Helper()
	return authtest.EnsureUser(t, authAddr, username)
}

// CreateAuthWebSession issues a durable auth-owned web session for one
// existing user.
func CreateAuthWebSession(t *testing.T, authAddr, userID string) string {
	t.Helper()
	return authtest.CreateWebSession(t, authAddr, userID)
}
