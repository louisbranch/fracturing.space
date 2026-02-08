package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// SetContextInput represents the MCP tool input for setting context.
type SetContextInput struct {
	CampaignID    string `json:"campaign_id" jsonschema:"campaign identifier (required)"`
	SessionID     string `json:"session_id,omitempty" jsonschema:"optional session identifier"`
	ParticipantID string `json:"participant_id,omitempty" jsonschema:"optional participant identifier"`
}

// SetContextResult represents the MCP tool output for setting context.
type SetContextResult struct {
	Context struct {
		CampaignID    string `json:"campaign_id" jsonschema:"campaign identifier"`
		SessionID     string `json:"session_id,omitempty" jsonschema:"optional session identifier"`
		ParticipantID string `json:"participant_id,omitempty" jsonschema:"optional participant identifier"`
	} `json:"context" jsonschema:"current context"`
}

// SetContextTool defines the MCP tool schema for setting context.
func SetContextTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "set_context",
		Description: "Sets the current context (campaign_id, optional session_id, optional participant_id) for subsequent tool calls",
	}
}

// SetContextHandler executes a context set request.
// The handler needs access to the Server instance to update context state,
// so it takes both clients and a function to update the server's context.
func SetContextHandler(
	campaignClient statev1.CampaignServiceClient,
	sessionClient statev1.SessionServiceClient,
	participantClient statev1.ParticipantServiceClient,
	setContextFunc func(ctx Context),
	getContextFunc func() Context,
	notify ResourceUpdateNotifier,
) mcp.ToolHandlerFor[SetContextInput, SetContextResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SetContextInput) (*mcp.CallToolResult, SetContextResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, SetContextResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Validate campaign_id is not empty
		campaignID := strings.TrimSpace(input.CampaignID)
		if campaignID == "" {
			return nil, SetContextResult{}, fmt.Errorf("campaign_id is required")
		}

		// Validate campaign exists
		responseMeta, err := validateCampaignExists(runCtx, campaignClient, campaignID, invocationID)
		if err != nil {
			return nil, SetContextResult{}, fmt.Errorf("validate campaign: %w", err)
		}
		lastMeta := responseMeta

		// Build new context starting with campaign_id
		newCtx := Context{
			CampaignID: campaignID,
		}

		// Validate and set session_id if provided (treat whitespace-only as omitted)
		if input.SessionID != "" {
			sessionID := strings.TrimSpace(input.SessionID)
			if sessionID != "" {
				responseMeta, err := validateSessionExists(runCtx, sessionClient, campaignID, sessionID, invocationID)
				if err != nil {
					return nil, SetContextResult{}, fmt.Errorf("validate session: %w", err)
				}
				lastMeta = responseMeta
				newCtx.SessionID = sessionID
			}
		}

		// Validate and set participant_id if provided (treat whitespace-only as omitted)
		if input.ParticipantID != "" {
			participantID := strings.TrimSpace(input.ParticipantID)
			if participantID != "" {
				responseMeta, err := validateParticipantExists(runCtx, participantClient, campaignID, participantID, invocationID)
				if err != nil {
					return nil, SetContextResult{}, fmt.Errorf("validate participant: %w", err)
				}
				lastMeta = responseMeta
				newCtx.ParticipantID = participantID
			}
		}

		// Update server context
		setContextFunc(newCtx)

		NotifyResourceUpdates(ctx, notify, ContextResource().URI)

		// Return current context
		currentCtx := getContextFunc()
		result := SetContextResult{}
		result.Context.CampaignID = currentCtx.CampaignID
		if currentCtx.SessionID != "" {
			result.Context.SessionID = currentCtx.SessionID
		}
		if currentCtx.ParticipantID != "" {
			result.Context.ParticipantID = currentCtx.ParticipantID
		}

		return CallToolResultWithMetadata(lastMeta), result, nil
	}
}

// Context represents the current MCP context.
// This is a duplicate of the one in service package to avoid circular imports.
type Context struct {
	CampaignID    string
	SessionID     string
	ParticipantID string
}

// validateCampaignExists checks if a campaign exists by calling GetCampaign.
func validateCampaignExists(ctx context.Context, client statev1.CampaignServiceClient, campaignID string, invocationID string) (ToolCallMetadata, error) {
	callCtx, callMeta, err := NewOutgoingContext(ctx, invocationID)
	if err != nil {
		return ToolCallMetadata{}, fmt.Errorf("create request metadata: %w", err)
	}

	var header metadata.MD
	_, err = client.GetCampaign(callCtx, &statev1.GetCampaignRequest{
		CampaignId: campaignID,
	}, grpc.Header(&header))
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.NotFound {
				return ToolCallMetadata{}, fmt.Errorf("campaign not found")
			}
		}
		return ToolCallMetadata{}, fmt.Errorf("get campaign: %w", err)
	}

	return MergeResponseMetadata(callMeta, header), nil
}

// validateSessionExists checks if a session exists and belongs to the campaign.
// The GetSession gRPC method validates that the session belongs to the campaign.
func validateSessionExists(ctx context.Context, client statev1.SessionServiceClient, campaignID, sessionID, invocationID string) (ToolCallMetadata, error) {
	callCtx, callMeta, err := NewOutgoingContext(ctx, invocationID)
	if err != nil {
		return ToolCallMetadata{}, fmt.Errorf("create request metadata: %w", err)
	}

	var header metadata.MD
	_, err = client.GetSession(callCtx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	}, grpc.Header(&header))
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.NotFound {
				return ToolCallMetadata{}, fmt.Errorf("session not found")
			}
			if s.Code() == codes.InvalidArgument {
				return ToolCallMetadata{}, fmt.Errorf("session not found or does not belong to campaign")
			}
		}
		return ToolCallMetadata{}, fmt.Errorf("get session: %w", err)
	}

	return MergeResponseMetadata(callMeta, header), nil
}

// validateParticipantExists checks if a participant exists and belongs to the campaign.
// The GetParticipant gRPC method validates that the participant belongs to the campaign.
func validateParticipantExists(ctx context.Context, client statev1.ParticipantServiceClient, campaignID, participantID, invocationID string) (ToolCallMetadata, error) {
	callCtx, callMeta, err := NewOutgoingContext(ctx, invocationID)
	if err != nil {
		return ToolCallMetadata{}, fmt.Errorf("create request metadata: %w", err)
	}

	var header metadata.MD
	_, err = client.GetParticipant(callCtx, &statev1.GetParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	}, grpc.Header(&header))
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.NotFound {
				return ToolCallMetadata{}, fmt.Errorf("participant not found")
			}
			if s.Code() == codes.InvalidArgument {
				return ToolCallMetadata{}, fmt.Errorf("participant not found or does not belong to campaign")
			}
		}
		return ToolCallMetadata{}, fmt.Errorf("get participant: %w", err)
	}

	return MergeResponseMetadata(callMeta, header), nil
}

// ContextResourcePayload represents the MCP resource payload for the current context.
type ContextResourcePayload struct {
	Context struct {
		CampaignID    *string `json:"campaign_id"`
		SessionID     *string `json:"session_id"`
		ParticipantID *string `json:"participant_id"`
	} `json:"context"`
}

// ContextResource defines the MCP resource for the current context.
func ContextResource() *mcp.Resource {
	return &mcp.Resource{
		Name:        "context_current",
		Title:       "Current Context",
		Description: "Readable current MCP context (campaign_id, session_id, participant_id)",
		MIMEType:    "application/json",
		URI:         "context://current",
	}
}

// ContextResourceHandler returns a readable current context resource.
func ContextResourceHandler(getContextFunc func() Context) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if getContextFunc == nil {
			return nil, fmt.Errorf("context getter function is not configured")
		}

		uri := ContextResource().URI
		if req != nil && req.Params != nil && req.Params.URI != "" {
			uri = req.Params.URI
		}

		// Validate URI matches context://current
		if uri != "context://current" {
			return nil, fmt.Errorf("invalid URI: expected context://current, got %q", uri)
		}

		// Get current context
		currentCtx := getContextFunc()

		// Build payload with null for empty strings
		payload := ContextResourcePayload{}
		if currentCtx.CampaignID != "" {
			payload.Context.CampaignID = &currentCtx.CampaignID
		}
		if currentCtx.SessionID != "" {
			payload.Context.SessionID = &currentCtx.SessionID
		}
		if currentCtx.ParticipantID != "" {
			payload.Context.ParticipantID = &currentCtx.ParticipantID
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal context: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	}
}
