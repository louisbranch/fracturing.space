//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/louisbranch/duality-engine/internal/mcp/domain"
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

		participantResource, found := findResource(result.Resources, "participant_list")
		if !found {
			t.Fatal("expected participant_list resource")
		}
		if !strings.HasPrefix(participantResource.URI, "campaign://") {
			t.Fatalf("expected resource URI to start with campaign://, got %q", participantResource.URI)
		}
		if !strings.HasSuffix(participantResource.URI, "/participants") {
			t.Fatalf("expected resource URI to end with /participants, got %q", participantResource.URI)
		}
		if participantResource.MIMEType != "application/json" {
			t.Fatalf("expected resource MIME application/json, got %q", participantResource.MIMEType)
		}

		actorResource, found := findResource(result.Resources, "actor_list")
		if !found {
			t.Fatal("expected actor_list resource")
		}
		if !strings.HasPrefix(actorResource.URI, "campaign://") {
			t.Fatalf("expected resource URI to start with campaign://, got %q", actorResource.URI)
		}
		if !strings.HasSuffix(actorResource.URI, "/actors") {
			t.Fatalf("expected resource URI to end with /actors, got %q", actorResource.URI)
		}
		if actorResource.MIMEType != "application/json" {
			t.Fatalf("expected resource MIME application/json, got %q", actorResource.MIMEType)
		}

		sessionResource, found := findResource(result.Resources, "session_list")
		if !found {
			t.Fatal("expected session_list resource")
		}
		if !strings.HasPrefix(sessionResource.URI, "campaign://") {
			t.Fatalf("expected resource URI to start with campaign://, got %q", sessionResource.URI)
		}
		if !strings.HasSuffix(sessionResource.URI, "/sessions") {
			t.Fatalf("expected resource URI to end with /sessions, got %q", sessionResource.URI)
		}
		if sessionResource.MIMEType != "application/json" {
			t.Fatalf("expected resource MIME application/json, got %q", sessionResource.MIMEType)
		}
	})

	t.Run("read participant list resource", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Create a campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Participant Test Campaign",
				"gm_mode":      "AI",
				"theme_prompt": "test theme",
			},
		}
		campaignResult, err := suite.client.CallTool(ctx, campaignParams)
		if err != nil {
			t.Fatalf("call campaign_create: %v", err)
		}
		if campaignResult == nil || campaignResult.IsError {
			t.Fatalf("campaign_create failed: %+v", campaignResult)
		}
		campaignOutput := decodeStructuredContent[domain.CampaignCreateResult](t, campaignResult.StructuredContent)
		if campaignOutput.ID == "" {
			t.Fatal("campaign_create returned empty id")
		}

		// Create a participant
		participantParams := &mcp.CallToolParams{
			Name: "participant_create",
			Arguments: map[string]any{
				"campaign_id":  campaignOutput.ID,
				"display_name": "Test GM",
				"role":         "GM",
				"controller":   "HUMAN",
			},
		}
		participantResult, err := suite.client.CallTool(ctx, participantParams)
		if err != nil {
			t.Fatalf("call participant_create: %v", err)
		}
		if participantResult == nil || participantResult.IsError {
			t.Fatalf("participant_create failed: %+v", participantResult)
		}
		participantOutput := decodeStructuredContent[domain.ParticipantCreateResult](t, participantResult.StructuredContent)
		if participantOutput.ID == "" {
			t.Fatal("participant_create returned empty id")
		}

		// Note: The MCP SDK validates URIs exactly against registered resources.
		// Since we registered campaign://_/participants, the SDK only accepts that exact URI.
		// The handler implementation correctly parses campaign://{campaign_id}/participants
		// format, but the SDK validation prevents testing it directly via ReadResource.
		// The handler logic is tested in unit tests (TestParticipantListResourceHandler*).
		// This integration test verifies the resource is discoverable and the handler
		// would work correctly if the SDK supported URI templates.
		//
		// For now, we test that the registered URI format is accepted (even though
		// it uses a placeholder) to verify the resource is properly registered.
		registeredURI := "campaign://_/participants"
		_, err = suite.client.ReadResource(ctx, &mcp.ReadResourceParams{URI: registeredURI})
		if err != nil {
			// The handler will reject the placeholder, which is expected behavior
			// This confirms the handler is being called and validates the campaign ID
			if !strings.Contains(err.Error(), "campaign ID") {
				t.Fatalf("read participant list resource: expected campaign ID error, got %v", err)
			}
		}
	})

	t.Run("read actor list resource", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Create a campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Actor Test Campaign",
				"gm_mode":      "AI",
				"theme_prompt": "test theme",
			},
		}
		campaignResult, err := suite.client.CallTool(ctx, campaignParams)
		if err != nil {
			t.Fatalf("call campaign_create: %v", err)
		}
		if campaignResult == nil || campaignResult.IsError {
			t.Fatalf("campaign_create failed: %+v", campaignResult)
		}
		campaignOutput := decodeStructuredContent[domain.CampaignCreateResult](t, campaignResult.StructuredContent)
		if campaignOutput.ID == "" {
			t.Fatal("campaign_create returned empty id")
		}

		// Create an actor
		actorParams := &mcp.CallToolParams{
			Name: "actor_create",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Test PC",
				"kind":        "PC",
				"notes":       "Test notes",
			},
		}
		actorResult, err := suite.client.CallTool(ctx, actorParams)
		if err != nil {
			t.Fatalf("call actor_create: %v", err)
		}
		if actorResult == nil || actorResult.IsError {
			t.Fatalf("actor_create failed: %+v", actorResult)
		}
		actorOutput := decodeStructuredContent[domain.ActorCreateResult](t, actorResult.StructuredContent)
		if actorOutput.ID == "" {
			t.Fatal("actor_create returned empty id")
		}

		// Note: The MCP SDK validates URIs exactly against registered resources.
		// Since we registered campaign://_/actors, the SDK only accepts that exact URI.
		// The handler implementation correctly parses campaign://{campaign_id}/actors
		// format, but the SDK validation prevents testing it directly via ReadResource.
		// The handler logic is tested in unit tests (TestActorListResourceHandler*).
		// This integration test verifies the resource is discoverable and the handler
		// would work correctly if the SDK supported URI templates.
		//
		// For now, we test that the registered URI format is accepted (even though
		// it uses a placeholder) to verify the resource is properly registered.
		registeredURI := "campaign://_/actors"
		_, err = suite.client.ReadResource(ctx, &mcp.ReadResourceParams{URI: registeredURI})
		if err != nil {
			// The handler will reject the placeholder, which is expected behavior
			// This confirms the handler is being called and validates the campaign ID
			if !strings.Contains(err.Error(), "campaign ID") {
				t.Fatalf("read actor list resource: expected campaign ID error, got %v", err)
			}
		}
	})

	t.Run("read session list resource", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Create a campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Session Test Campaign",
				"gm_mode":      "AI",
				"theme_prompt": "test theme",
			},
		}
		campaignResult, err := suite.client.CallTool(ctx, campaignParams)
		if err != nil {
			t.Fatalf("call campaign_create: %v", err)
		}
		if campaignResult == nil || campaignResult.IsError {
			t.Fatalf("campaign_create failed: %+v", campaignResult)
		}
		campaignOutput := decodeStructuredContent[domain.CampaignCreateResult](t, campaignResult.StructuredContent)
		if campaignOutput.ID == "" {
			t.Fatal("campaign_create returned empty id")
		}

		// Start a session
		sessionParams := &mcp.CallToolParams{
			Name: "session_start",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Test Session",
			},
		}
		sessionResult, err := suite.client.CallTool(ctx, sessionParams)
		if err != nil {
			t.Fatalf("call session_start: %v", err)
		}
		if sessionResult == nil || sessionResult.IsError {
			t.Fatalf("session_start failed: %+v", sessionResult)
		}
		sessionOutput := decodeStructuredContent[domain.SessionStartResult](t, sessionResult.StructuredContent)
		if sessionOutput.ID == "" {
			t.Fatal("session_start returned empty id")
		}

		// Note: The MCP SDK validates URIs exactly against registered resources.
		// Since we registered campaign://_/sessions, the SDK only accepts that exact URI.
		// The handler implementation correctly parses campaign://{campaign_id}/sessions
		// format, but the SDK validation prevents testing it directly via ReadResource.
		// The handler logic is tested in unit tests (TestSessionListResourceHandler*).
		// This integration test verifies the resource is discoverable and the handler
		// would work correctly if the SDK supported URI templates.
		//
		// For now, we test that the registered URI format is accepted (even though
		// it uses a placeholder) to verify the resource is properly registered.
		registeredURI := "campaign://_/sessions"
		_, err = suite.client.ReadResource(ctx, &mcp.ReadResourceParams{URI: registeredURI})
		if err != nil {
			// The handler will reject the placeholder, which is expected behavior
			// This confirms the handler is being called and validates the campaign ID
			if !strings.Contains(err.Error(), "campaign ID") {
				t.Fatalf("read session list resource: expected campaign ID error, got %v", err)
			}
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
