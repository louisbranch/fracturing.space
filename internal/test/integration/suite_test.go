//go:build integration

package integration

import "testing"

// TestGameEndToEnd validates game gRPC behavior end-to-end against the shared fixture stack.
func TestGameEndToEnd(t *testing.T) {
	fixture := newSuiteFixture(t)
	newSuite := func(t *testing.T, label string) *integrationSuite {
		t.Helper()
		userID := fixture.newUserID(t, uniqueTestUsername(t, label))
		return fixture.newGameSuite(t, userID)
	}

	t.Run("campaign tools", func(t *testing.T) {
		runCampaignToolsTests(t, newSuite(t, "campaign-tools"))
	})

	t.Run("fork tools", func(t *testing.T) {
		runForkToolsTests(t, newSuite(t, "fork-tools"))
	})

	t.Run("session lock", func(t *testing.T) {
		runSessionLockTests(t, fixture.grpcAddr, fixture.authAddr)
	})

	t.Run("participant user link", func(t *testing.T) {
		runParticipantUserLinkTests(t, fixture.grpcAddr, fixture.authAddr)
	})

	t.Run("event list", func(t *testing.T) {
		runEventListTests(t, fixture.grpcAddr, fixture.authAddr)
	})

	t.Run("mutation event guardrails", func(t *testing.T) {
		runMutationEventGuardrailTests(t, newSuite(t, "mutation-guardrails"), fixture.grpcAddr)
	})

	t.Run("invite lifecycle", func(t *testing.T) {
		runInviteLifecycleTests(t, fixture)
	})
}
