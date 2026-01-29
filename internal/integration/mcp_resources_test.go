//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
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

		contextResource, found := findResource(result.Resources, "context_current")
		if !found {
			t.Fatal("expected context_current resource")
		}
		if contextResource.URI != "context://current" {
			t.Fatalf("expected resource URI context://current, got %q", contextResource.URI)
		}
		if contextResource.MIMEType != "application/json" {
			t.Fatalf("expected resource MIME application/json, got %q", contextResource.MIMEType)
		}

		templateResult, err := suite.client.ListResourceTemplates(ctx, nil)
		if err != nil {
			t.Fatalf("list resource templates: %v", err)
		}
		if templateResult == nil {
			t.Fatal("list resource templates returned nil result")
		}

		campaignTemplate, found := findResourceTemplate(templateResult.ResourceTemplates, "campaign")
		if !found {
			t.Fatal("expected campaign resource template")
		}
		if campaignTemplate.URITemplate != "campaign://{campaign_id}" {
			t.Fatalf("expected campaign template URI campaign://{campaign_id}, got %q", campaignTemplate.URITemplate)
		}
		if campaignTemplate.MIMEType != "application/json" {
			t.Fatalf("expected campaign template MIME application/json, got %q", campaignTemplate.MIMEType)
		}

		participantTemplate, found := findResourceTemplate(templateResult.ResourceTemplates, "participant_list")
		if !found {
			t.Fatal("expected participant_list resource template")
		}
		if participantTemplate.URITemplate != "campaign://{campaign_id}/participants" {
			t.Fatalf("expected participant template URI campaign://{campaign_id}/participants, got %q", participantTemplate.URITemplate)
		}
		if participantTemplate.MIMEType != "application/json" {
			t.Fatalf("expected participant template MIME application/json, got %q", participantTemplate.MIMEType)
		}

		characterTemplate, found := findResourceTemplate(templateResult.ResourceTemplates, "character_list")
		if !found {
			t.Fatal("expected character_list resource template")
		}
		if characterTemplate.URITemplate != "campaign://{campaign_id}/characters" {
			t.Fatalf("expected character template URI campaign://{campaign_id}/characters, got %q", characterTemplate.URITemplate)
		}
		if characterTemplate.MIMEType != "application/json" {
			t.Fatalf("expected character template MIME application/json, got %q", characterTemplate.MIMEType)
		}

		sessionTemplate, found := findResourceTemplate(templateResult.ResourceTemplates, "session_list")
		if !found {
			t.Fatal("expected session_list resource template")
		}
		if sessionTemplate.URITemplate != "campaign://{campaign_id}/sessions" {
			t.Fatalf("expected session template URI campaign://{campaign_id}/sessions, got %q", sessionTemplate.URITemplate)
		}
		if sessionTemplate.MIMEType != "application/json" {
			t.Fatalf("expected session template MIME application/json, got %q", sessionTemplate.MIMEType)
		}

		sessionEventsTemplate, found := findResourceTemplate(templateResult.ResourceTemplates, "session_events")
		if !found {
			t.Fatal("expected session_events resource template")
		}
		if sessionEventsTemplate.URITemplate != "session://{session_id}/events" {
			t.Fatalf("expected session events template URI session://{session_id}/events, got %q", sessionEventsTemplate.URITemplate)
		}
		if sessionEventsTemplate.MIMEType != "application/json" {
			t.Fatalf("expected session events template MIME application/json, got %q", sessionEventsTemplate.MIMEType)
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
		if campaignOutput.GmFear != 0 {
			t.Fatalf("expected gm_fear 0, got %d", campaignOutput.GmFear)
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

		resourceURI := fmt.Sprintf("campaign://%s/participants", campaignOutput.ID)
		resourceResult, err := suite.client.ReadResource(ctx, &mcp.ReadResourceParams{URI: resourceURI})
		if err != nil {
			t.Fatalf("read participant list resource: %v", err)
		}
		if resourceResult == nil || len(resourceResult.Contents) != 1 {
			t.Fatalf("expected 1 content item, got %v", resourceResult)
		}

		var payload domain.ParticipantListPayload
		if err := json.Unmarshal([]byte(resourceResult.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal participant list JSON: %v", err)
		}
		if len(payload.Participants) == 0 {
			t.Fatal("expected participants in list")
		}
		found := false
		for _, participant := range payload.Participants {
			if participant.ID == participantOutput.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected participant %q in list", participantOutput.ID)
		}
	})

	t.Run("read character list resource", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Create a campaign
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Character Test Campaign",
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

		// Create a character
		characterParams := &mcp.CallToolParams{
			Name: "character_create",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"name":        "Test PC",
				"kind":        "PC",
				"notes":       "Test notes",
			},
		}
		characterResult, err := suite.client.CallTool(ctx, characterParams)
		if err != nil {
			t.Fatalf("call character_create: %v", err)
		}
		if characterResult == nil || characterResult.IsError {
			t.Fatalf("character_create failed: %+v", characterResult)
		}
		characterOutput := decodeStructuredContent[domain.CharacterCreateResult](t, characterResult.StructuredContent)
		if characterOutput.ID == "" {
			t.Fatal("character_create returned empty id")
		}

		resourceURI := fmt.Sprintf("campaign://%s/characters", campaignOutput.ID)
		resourceResult, err := suite.client.ReadResource(ctx, &mcp.ReadResourceParams{URI: resourceURI})
		if err != nil {
			t.Fatalf("read character list resource: %v", err)
		}
		if resourceResult == nil || len(resourceResult.Contents) != 1 {
			t.Fatalf("expected 1 content item, got %v", resourceResult)
		}

		var payload domain.CharacterListPayload
		if err := json.Unmarshal([]byte(resourceResult.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal character list JSON: %v", err)
		}
		if len(payload.Characters) == 0 {
			t.Fatal("expected characters in list")
		}
		found := false
		for _, character := range payload.Characters {
			if character.ID == characterOutput.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected character %q in list", characterOutput.ID)
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

		endSessionParams := &mcp.CallToolParams{
			Name: "session_end",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
				"session_id":  sessionOutput.ID,
			},
		}
		endSessionResult, err := suite.client.CallTool(ctx, endSessionParams)
		if err != nil {
			t.Fatalf("call session_end: %v", err)
		}
		if endSessionResult == nil || endSessionResult.IsError {
			t.Fatalf("session_end failed: %+v", endSessionResult)
		}
		endSessionOutput := decodeStructuredContent[domain.SessionEndResult](t, endSessionResult.StructuredContent)
		if endSessionOutput.ID != sessionOutput.ID {
			t.Fatalf("expected session_end id %q, got %q", sessionOutput.ID, endSessionOutput.ID)
		}
		if endSessionOutput.Status != "ENDED" {
			t.Fatalf("expected ended status, got %q", endSessionOutput.Status)
		}
		if endSessionOutput.EndedAt == "" {
			t.Fatal("expected ended_at to be set")
		}

		resourceURI := fmt.Sprintf("campaign://%s/sessions", campaignOutput.ID)
		resourceResult, err := suite.client.ReadResource(ctx, &mcp.ReadResourceParams{URI: resourceURI})
		if err != nil {
			t.Fatalf("read session list resource: %v", err)
		}
		if resourceResult == nil || len(resourceResult.Contents) != 1 {
			t.Fatalf("expected 1 content item, got %v", resourceResult)
		}

		var payload domain.SessionListPayload
		if err := json.Unmarshal([]byte(resourceResult.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal session list JSON: %v", err)
		}
		if len(payload.Sessions) == 0 {
			t.Fatal("expected sessions in list")
		}
		found := false
		for _, session := range payload.Sessions {
			if session.ID == sessionOutput.ID {
				found = true
				if session.Status != "ENDED" {
					t.Fatalf("expected session status ENDED, got %q", session.Status)
				}
				break
			}
		}
		if !found {
			t.Fatalf("expected session %q in list", sessionOutput.ID)
		}
	})

	t.Run("read context resource", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		// Read context resource with empty context (should return all null fields)
		result, err := suite.client.ReadResource(ctx, &mcp.ReadResourceParams{URI: "context://current"})
		if err != nil {
			t.Fatalf("read context resource: %v", err)
		}
		if result == nil || len(result.Contents) != 1 {
			t.Fatalf("expected 1 content item, got %v", result)
		}
		if result.Contents[0].URI != "context://current" {
			t.Fatalf("expected URI context://current, got %q", result.Contents[0].URI)
		}
		if result.Contents[0].MIMEType != "application/json" {
			t.Fatalf("expected MIME application/json, got %q", result.Contents[0].MIMEType)
		}

		// Verify JSON structure with all null fields
		var payload domain.ContextResourcePayload
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal context JSON: %v", err)
		}
		if payload.Context.CampaignID != nil {
			t.Fatalf("expected null campaign_id for empty context, got %v", payload.Context.CampaignID)
		}
		if payload.Context.SessionID != nil {
			t.Fatalf("expected null session_id for empty context, got %v", payload.Context.SessionID)
		}
		if payload.Context.ParticipantID != nil {
			t.Fatalf("expected null participant_id for empty context, got %v", payload.Context.ParticipantID)
		}

		// Set context and read again
		campaignParams := &mcp.CallToolParams{
			Name: "campaign_create",
			Arguments: map[string]any{
				"name":         "Context Test Campaign",
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

		// Set context
		setContextParams := &mcp.CallToolParams{
			Name: "set_context",
			Arguments: map[string]any{
				"campaign_id": campaignOutput.ID,
			},
		}
		_, err = suite.client.CallTool(ctx, setContextParams)
		if err != nil {
			t.Fatalf("call set_context: %v", err)
		}

		// Read context resource again (should return campaign_id)
		result2, err := suite.client.ReadResource(ctx, &mcp.ReadResourceParams{URI: "context://current"})
		if err != nil {
			t.Fatalf("read context resource after set: %v", err)
		}
		if result2 == nil || len(result2.Contents) != 1 {
			t.Fatalf("expected 1 content item, got %v", result2)
		}

		var payload2 domain.ContextResourcePayload
		if err := json.Unmarshal([]byte(result2.Contents[0].Text), &payload2); err != nil {
			t.Fatalf("unmarshal context JSON: %v", err)
		}
		if payload2.Context.CampaignID == nil || *payload2.Context.CampaignID != campaignOutput.ID {
			t.Fatalf("expected campaign_id %q, got %v", campaignOutput.ID, payload2.Context.CampaignID)
		}
		if payload2.Context.SessionID != nil {
			t.Fatalf("expected null session_id, got %v", payload2.Context.SessionID)
		}
		if payload2.Context.ParticipantID != nil {
			t.Fatalf("expected null participant_id, got %v", payload2.Context.ParticipantID)
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

// findResourceTemplate searches a resource template list for a matching name.
func findResourceTemplate(templates []*mcp.ResourceTemplate, name string) (*mcp.ResourceTemplate, bool) {
	for _, template := range templates {
		if template != nil && template.Name == name {
			return template, true
		}
	}
	return nil, false
}
