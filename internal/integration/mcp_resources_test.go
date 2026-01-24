//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runMCPResourcesTests exercises MCP resource discovery.
func runMCPResourcesTests(t *testing.T, suite *integrationSuite) {
	t.Helper()

	t.Run("list resources", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		result, err := suite.client.ListResources(ctx, nil)
		if err != nil {
			t.Fatalf("list resources: %v", err)
		}
		if result == nil {
			t.Fatal("list resources returned nil result")
		}

		resource, found := findResource(result.Resources, "campaign_list")
		if !found {
			t.Fatal("expected campaign_list resource")
		}
		if resource.URI != "campaigns://list" {
			t.Fatalf("expected resource URI campaigns://list, got %q", resource.URI)
		}
		if resource.MIMEType != "application/json" {
			t.Fatalf("expected resource MIME application/json, got %q", resource.MIMEType)
		}
	})
}

// findResource searches a resource list for a matching name.
func findResource(resources []*mcp.Resource, name string) (*mcp.Resource, bool) {
	for _, resource := range resources {
		if resource != nil && resource.Name == name {
			return resource, true
		}
	}
	return nil, false
}
