package domain

import (
	"context"
	"fmt"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// CampaignForkInput represents the MCP tool input for forking a campaign.
type CampaignForkInput struct {
	SourceCampaignID string `json:"source_campaign_id" jsonschema:"source campaign identifier to fork from"`
	EventSeq         uint64 `json:"event_seq,omitempty" jsonschema:"optional event sequence to fork at (0 or omitted = latest)"`
	SessionID        string `json:"session_id,omitempty" jsonschema:"optional session ID to fork at the end of"`
	NewCampaignName  string `json:"new_campaign_name,omitempty" jsonschema:"optional name for the forked campaign"`
	CopyParticipants bool   `json:"copy_participants,omitempty" jsonschema:"whether to copy participants from source"`
}

// CampaignForkResult represents the MCP tool output for forking a campaign.
type CampaignForkResult struct {
	CampaignID       string `json:"campaign_id" jsonschema:"new forked campaign identifier"`
	Name             string `json:"name" jsonschema:"forked campaign name"`
	ForkEventSeq     uint64 `json:"fork_event_seq" jsonschema:"event sequence at which fork occurred"`
	ParentCampaignID string `json:"parent_campaign_id" jsonschema:"parent campaign identifier"`
	OriginCampaignID string `json:"origin_campaign_id" jsonschema:"root of the fork lineage"`
	Depth            int    `json:"depth" jsonschema:"fork depth (0 = original)"`
	Status           string `json:"status" jsonschema:"campaign status"`
	CreatedAt        string `json:"created_at" jsonschema:"RFC3339 timestamp when campaign was created"`
}

// CampaignLineageInput represents the MCP tool input for getting campaign lineage.
type CampaignLineageInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
}

// CampaignLineageResult represents the MCP tool output for campaign lineage.
type CampaignLineageResult struct {
	CampaignID       string `json:"campaign_id" jsonschema:"campaign identifier"`
	ParentCampaignID string `json:"parent_campaign_id,omitempty" jsonschema:"parent campaign identifier (empty if original)"`
	ForkEventSeq     uint64 `json:"fork_event_seq,omitempty" jsonschema:"event sequence at which fork occurred"`
	OriginCampaignID string `json:"origin_campaign_id" jsonschema:"root of the fork lineage"`
	Depth            int    `json:"depth" jsonschema:"fork depth (0 = original)"`
	IsOriginal       bool   `json:"is_original" jsonschema:"whether this is an original campaign"`
}

// CampaignForkTool defines the MCP tool schema for forking campaigns.
func CampaignForkTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_fork",
		Description: "Fork a campaign at a specific point in its history, creating a new independent campaign with the same state",
	}
}

// CampaignLineageTool defines the MCP tool schema for getting campaign lineage.
func CampaignLineageTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "campaign_lineage",
		Description: "Get the lineage (ancestry) of a campaign, showing its fork history",
	}
}

// CampaignForkHandler executes a campaign fork request.
func CampaignForkHandler(client statev1.ForkServiceClient, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignForkInput, CampaignForkResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignForkInput) (*mcp.CallToolResult, CampaignForkResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CampaignForkResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CampaignForkResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		forkPoint := &statev1.ForkPoint{}
		if input.SessionID != "" {
			forkPoint.SessionId = input.SessionID
		} else if input.EventSeq > 0 {
			forkPoint.EventSeq = input.EventSeq
		}

		response, err := client.ForkCampaign(callCtx, &statev1.ForkCampaignRequest{
			SourceCampaignId: input.SourceCampaignID,
			ForkPoint:        forkPoint,
			NewCampaignName:  input.NewCampaignName,
			CopyParticipants: input.CopyParticipants,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignForkResult{}, fmt.Errorf("campaign fork failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignForkResult{}, fmt.Errorf("campaign fork response is missing")
		}

		result := CampaignForkResult{
			CampaignID:       response.Campaign.GetId(),
			Name:             response.Campaign.GetName(),
			ForkEventSeq:     response.GetForkEventSeq(),
			ParentCampaignID: input.SourceCampaignID, // fallback
			Status:           campaignStatusToString(response.Campaign.GetStatus()),
			CreatedAt:        formatTimestamp(response.Campaign.GetCreatedAt()),
		}

		// Use authoritative values from server response
		if response.Lineage != nil {
			if pid := response.Lineage.GetParentCampaignId(); pid != "" {
				result.ParentCampaignID = pid
			}
			result.OriginCampaignID = response.Lineage.GetOriginCampaignId()
			result.Depth = int(response.Lineage.GetDepth())
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CampaignLineageHandler executes a campaign lineage request.
func CampaignLineageHandler(client statev1.ForkServiceClient) mcp.ToolHandlerFor[CampaignLineageInput, CampaignLineageResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignLineageInput) (*mcp.CallToolResult, CampaignLineageResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, CampaignLineageResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, CampaignLineageResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.GetLineage(callCtx, &statev1.GetLineageRequest{
			CampaignId: input.CampaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignLineageResult{}, fmt.Errorf("get lineage failed: %w", err)
		}
		if response == nil || response.Lineage == nil {
			return nil, CampaignLineageResult{}, fmt.Errorf("get lineage response is missing")
		}

		lineage := response.Lineage
		result := CampaignLineageResult{
			CampaignID:       lineage.GetCampaignId(),
			ParentCampaignID: lineage.GetParentCampaignId(),
			ForkEventSeq:     lineage.GetForkEventSeq(),
			OriginCampaignID: lineage.GetOriginCampaignId(),
			Depth:            int(lineage.GetDepth()),
			IsOriginal:       lineage.GetParentCampaignId() == "",
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}
