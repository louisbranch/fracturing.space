//go:build integration

package integration

import "testing"

// TestMCPStdioEndToEnd validates MCP stdio integration end-to-end.
func TestMCPStdioEndToEnd(t *testing.T) {
	fixture := newSuiteFixture(t)
	clientSession := fixture.newMCPClientSession(t)
	userID := fixture.newUserID(t, "test-creator")
	suite := &integrationSuite{client: clientSession, userID: userID}

	t.Run("duality tools", func(t *testing.T) {
		runDualityToolsTests(t, suite)
	})

	t.Run("campaign tools", func(t *testing.T) {
		runCampaignToolsTests(t, suite)
	})

	t.Run("fork tools", func(t *testing.T) {
		runForkToolsTests(t, suite)
	})

	t.Run("session outcomes", func(t *testing.T) {
		runSessionOutcomeTests(t, suite, fixture.grpcAddr)
	})

	t.Run("metadata", func(t *testing.T) {
		runMetadataTests(t, suite, fixture.grpcAddr)
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
		runMutationEventGuardrailTests(t, suite, fixture.grpcAddr, fixture.authAddr)
	})
}
