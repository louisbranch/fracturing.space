package domain

import (
	"context"
	"fmt"
	"time"

	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SessionStartInput represents the MCP tool input for starting a session.
type SessionStartInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name,omitempty" jsonschema:"optional free-form name for the session"`
}

// SessionStartResult represents the MCP tool output for starting a session.
type SessionStartResult struct {
	ID         string `json:"id" jsonschema:"session identifier"`
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"session name"`
	Status     string `json:"status" jsonschema:"session status (ACTIVE, PAUSED, ENDED)"`
	StartedAt  string `json:"started_at" jsonschema:"RFC3339 timestamp when session was started"`
	UpdatedAt  string `json:"updated_at" jsonschema:"RFC3339 timestamp when session was last updated"`
	EndedAt    string `json:"ended_at,omitempty" jsonschema:"RFC3339 timestamp when session ended, if applicable"`
}

// SessionStartTool defines the MCP tool schema for starting a session.
func SessionStartTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "session_start",
		Description: "Starts a new session for a campaign. Enforces at most one ACTIVE session per campaign.",
	}
}

// SessionStartHandler executes a session start request.
func SessionStartHandler(client sessionv1.SessionServiceClient) mcp.ToolHandlerFor[SessionStartInput, SessionStartResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SessionStartInput) (*mcp.CallToolResult, SessionStartResult, error) {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		response, err := client.StartSession(runCtx, &sessionv1.StartSessionRequest{
			CampaignId: input.CampaignID,
			Name:       input.Name,
		})
		if err != nil {
			return nil, SessionStartResult{}, fmt.Errorf("session start failed: %w", err)
		}
		if response == nil || response.Session == nil {
			return nil, SessionStartResult{}, fmt.Errorf("session start response is missing")
		}

		result := SessionStartResult{
			ID:         response.Session.GetId(),
			CampaignID: response.Session.GetCampaignId(),
			Name:       response.Session.GetName(),
			Status:     sessionStatusToString(response.Session.GetStatus()),
			StartedAt:  formatTimestamp(response.Session.GetStartedAt()),
			UpdatedAt:  formatTimestamp(response.Session.GetUpdatedAt()),
		}

		if response.Session.GetEndedAt() != nil {
			result.EndedAt = formatTimestamp(response.Session.GetEndedAt())
		}

		return nil, result, nil
	}
}

// sessionStatusToString converts a protobuf SessionStatus to a string representation.
func sessionStatusToString(status sessionv1.SessionStatus) string {
	switch status {
	case sessionv1.SessionStatus_ACTIVE:
		return "ACTIVE"
	case sessionv1.SessionStatus_PAUSED:
		return "PAUSED"
	case sessionv1.SessionStatus_ENDED:
		return "ENDED"
	case sessionv1.SessionStatus_STATUS_UNSPECIFIED:
		return "UNSPECIFIED"
	default:
		return "UNSPECIFIED"
	}
}
