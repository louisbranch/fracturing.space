//go:build integration

package integration

import "testing"

// TestMCPStdioEndToEnd validates MCP stdio integration end-to-end.
func TestMCPStdioEndToEnd(t *testing.T) {
	grpcAddr, stopServer := startGRPCServer(t)
	defer stopServer()

	clientSession, closeClient := startMCPClient(t, grpcAddr)
	defer closeClient()

	suite := &integrationSuite{client: clientSession}

	t.Run("duality tools", func(t *testing.T) {
		runDualityToolsTests(t, suite)
	})

	t.Run("campaign tools", func(t *testing.T) {
		runCampaignToolsTests(t, suite)
	})

	t.Run("session outcomes", func(t *testing.T) {
		runSessionOutcomeTests(t, suite, grpcAddr)
	})

	t.Run("metadata", func(t *testing.T) {
		runMetadataTests(t, suite, grpcAddr)
	})

	t.Run("session lock", func(t *testing.T) {
		runSessionLockTests(t, grpcAddr)
	})
}
