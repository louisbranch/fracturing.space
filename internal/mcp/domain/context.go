package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
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
	campaignClient campaignv1.CampaignServiceClient,
	sessionClient sessionv1.SessionServiceClient,
	setContextFunc func(ctx Context),
	getContextFunc func() Context,
) mcp.ToolHandlerFor[SetContextInput, SetContextResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SetContextInput) (*mcp.CallToolResult, SetContextResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Validate campaign_id is not empty
		campaignID := strings.TrimSpace(input.CampaignID)
		if campaignID == "" {
			return nil, SetContextResult{}, fmt.Errorf("campaign_id is required")
		}

		// Validate campaign exists
		if err := validateCampaignExists(runCtx, campaignClient, campaignID); err != nil {
			return nil, SetContextResult{}, fmt.Errorf("validate campaign: %w", err)
		}

		// Build new context starting with campaign_id
		newCtx := Context{
			CampaignID: campaignID,
		}

		// Validate and set session_id if provided (treat whitespace-only as omitted)
		if input.SessionID != "" {
			sessionID := strings.TrimSpace(input.SessionID)
			if sessionID != "" {
				if err := validateSessionExists(runCtx, sessionClient, campaignID, sessionID); err != nil {
					return nil, SetContextResult{}, fmt.Errorf("validate session: %w", err)
				}
				newCtx.SessionID = sessionID
			}
		}

		// Validate and set participant_id if provided (treat whitespace-only as omitted)
		if input.ParticipantID != "" {
			participantID := strings.TrimSpace(input.ParticipantID)
			if participantID != "" {
				if err := validateParticipantExists(runCtx, campaignClient, campaignID, participantID); err != nil {
					return nil, SetContextResult{}, fmt.Errorf("validate participant: %w", err)
				}
				newCtx.ParticipantID = participantID
			}
		}

		// Update server context
		setContextFunc(newCtx)

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

		return nil, result, nil
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
func validateCampaignExists(ctx context.Context, client campaignv1.CampaignServiceClient, campaignID string) error {
	_, err := client.GetCampaign(ctx, &campaignv1.GetCampaignRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.NotFound {
				return fmt.Errorf("campaign not found")
			}
		}
		return fmt.Errorf("get campaign: %w", err)
	}
	return nil
}

// validateSessionExists checks if a session exists and belongs to the campaign.
// The GetSession gRPC method validates that the session belongs to the campaign.
func validateSessionExists(ctx context.Context, client sessionv1.SessionServiceClient, campaignID, sessionID string) error {
	_, err := client.GetSession(ctx, &sessionv1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.NotFound {
				return fmt.Errorf("session not found")
			}
			if s.Code() == codes.InvalidArgument {
				return fmt.Errorf("session not found or does not belong to campaign")
			}
		}
		return fmt.Errorf("get session: %w", err)
	}
	return nil
}

// validateParticipantExists checks if a participant exists and belongs to the campaign.
// The GetParticipant gRPC method validates that the participant belongs to the campaign.
func validateParticipantExists(ctx context.Context, client campaignv1.CampaignServiceClient, campaignID, participantID string) error {
	_, err := client.GetParticipant(ctx, &campaignv1.GetParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.NotFound {
				return fmt.Errorf("participant not found")
			}
			if s.Code() == codes.InvalidArgument {
				return fmt.Errorf("participant not found or does not belong to campaign")
			}
		}
		return fmt.Errorf("get participant: %w", err)
	}
	return nil
}
