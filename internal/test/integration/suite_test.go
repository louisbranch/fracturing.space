//go:build integration

package integration

import "testing"

// TestMCPEndToEnd validates the MCP bridge end-to-end against the shared fixture stack.
func TestMCPEndToEnd(t *testing.T) {
	fixture := newSuiteFixture(t)
	newSuite := func(t *testing.T, label string) *integrationSuite {
		t.Helper()
		clientSession := fixture.newMCPClientSession(t)
		userID := fixture.newUserID(t, uniqueTestUsername(t, label))
		return &integrationSuite{client: clientSession, userID: userID}
	}

	t.Run("duality tools", func(t *testing.T) {
		runDualityToolsTests(t, newSuite(t, "duality-tools"))
	})

	t.Run("campaign tools", func(t *testing.T) {
		runCampaignToolsTests(t, newSuite(t, "campaign-tools"))
	})

	t.Run("fork tools", func(t *testing.T) {
		runForkToolsTests(t, newSuite(t, "fork-tools"))
	})

	t.Run("session outcomes", func(t *testing.T) {
		runSessionOutcomeTests(t, newSuite(t, "session-outcomes"), fixture.grpcAddr)
	})

	t.Run("metadata", func(t *testing.T) {
		runMetadataTests(t, newSuite(t, "metadata"), fixture.grpcAddr)
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
		runMutationEventGuardrailTests(t, newSuite(t, "mutation-guardrails"), fixture.grpcAddr, fixture.authAddr)
	})
}
