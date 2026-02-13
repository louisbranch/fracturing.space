//go:build integration

package integration

import "testing"

// TestMCPStdioEndToEnd validates MCP stdio integration end-to-end.
func TestMCPStdioEndToEnd(t *testing.T) {
	grpcAddr, authAddr, stopServer := startGRPCServer(t)
	defer stopServer()

	clientSession, closeClient := startMCPClient(t, grpcAddr)
	defer closeClient()

	userID := createAuthUser(t, authAddr, "Test Creator")
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
		runSessionOutcomeTests(t, suite, grpcAddr)
	})

	t.Run("metadata", func(t *testing.T) {
		runMetadataTests(t, suite, grpcAddr)
	})

	t.Run("session lock", func(t *testing.T) {
		runSessionLockTests(t, grpcAddr, authAddr)
	})

	t.Run("participant user link", func(t *testing.T) {
		runParticipantUserLinkTests(t, grpcAddr, authAddr)
	})

	t.Run("event list", func(t *testing.T) {
		runEventListTests(t, grpcAddr, authAddr)
	})

	t.Run("mutation event guardrails", func(t *testing.T) {
		runMutationEventGuardrailTests(t, suite, grpcAddr)
	})
}
