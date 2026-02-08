package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// EventListInput represents the MCP tool input for listing events.
type EventListInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	PageSize   int    `json:"page_size,omitempty" jsonschema:"page size (default 50, max 200)"`
	PageToken  string `json:"page_token,omitempty" jsonschema:"opaque cursor for pagination"`
	OrderBy    string `json:"order_by,omitempty" jsonschema:"ordering: 'seq' (oldest first) or 'seq desc' (newest first)"`
	Filter     string `json:"filter,omitempty" jsonschema:"AIP-160 filter expression"`
}

// EventListEntry represents a single event in the list result.
type EventListEntry struct {
	CampaignID   string `json:"campaign_id"`
	Seq          uint64 `json:"seq"`
	Hash         string `json:"hash"`
	Timestamp    string `json:"ts"`
	Type         string `json:"type"`
	SessionID    string `json:"session_id,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
	InvocationID string `json:"invocation_id,omitempty"`
	ActorType    string `json:"actor_type"`
	ActorID      string `json:"actor_id,omitempty"`
	EntityType   string `json:"entity_type,omitempty"`
	EntityID     string `json:"entity_id,omitempty"`
	PayloadJSON  string `json:"payload_json,omitempty"`
}

// EventListResult represents the MCP tool output for listing events.
type EventListResult struct {
	Events            []EventListEntry `json:"events"`
	NextPageToken     string           `json:"next_page_token,omitempty"`
	PreviousPageToken string           `json:"previous_page_token,omitempty"`
	TotalSize         int              `json:"total_size"`
}

// EventListTool defines the MCP tool schema for listing events.
func EventListTool() *mcp.Tool {
	return &mcp.Tool{
		Name: "events_list",
		Description: `Lists events for a campaign with pagination, sorting, and filtering.

Supports AIP-160 filter expressions for flexible querying:
- session_id = "sess_123" - Filter by session
- type = "action.roll_resolved" - Filter by event type
- actor_id = "p_123" - Filter by actor
- ts >= timestamp("2024-01-15T00:00:00Z") - Filter by time
- Combine with AND/OR: session_id = "sess_123" AND type = "action.outcome_applied"

Ordering options:
- "seq" (default) - Oldest first
- "seq desc" - Newest first

Use page_token from response to paginate through results.`,
	}
}

// EventListHandler executes an event list request.
func EventListHandler(client statev1.EventServiceClient, getContext func() Context) mcp.ToolHandlerFor[EventListInput, EventListResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input EventListInput) (*mcp.CallToolResult, EventListResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, EventListResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// Use context campaign_id if not provided
		campaignID := input.CampaignID
		if campaignID == "" && getContext != nil {
			campaignID = getContext().CampaignID
		}
		if campaignID == "" {
			return nil, EventListResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, EventListResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.ListEvents(callCtx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   int32(input.PageSize),
			PageToken:  input.PageToken,
			OrderBy:    input.OrderBy,
			Filter:     input.Filter,
		}, grpc.Header(&header))
		if err != nil {
			return nil, EventListResult{}, fmt.Errorf("list events failed: %w", err)
		}
		if response == nil {
			return nil, EventListResult{}, fmt.Errorf("list events response is missing")
		}

		result := EventListResult{
			Events:            make([]EventListEntry, 0, len(response.GetEvents())),
			NextPageToken:     response.GetNextPageToken(),
			PreviousPageToken: response.GetPreviousPageToken(),
			TotalSize:         int(response.GetTotalSize()),
		}

		for _, evt := range response.GetEvents() {
			entry := EventListEntry{
				CampaignID:   evt.GetCampaignId(),
				Seq:          evt.GetSeq(),
				Hash:         evt.GetHash(),
				Timestamp:    formatTimestamp(evt.GetTs()),
				Type:         evt.GetType(),
				SessionID:    evt.GetSessionId(),
				RequestID:    evt.GetRequestId(),
				InvocationID: evt.GetInvocationId(),
				ActorType:    evt.GetActorType(),
				ActorID:      evt.GetActorId(),
				EntityType:   evt.GetEntityType(),
				EntityID:     evt.GetEntityId(),
			}
			if len(evt.GetPayloadJson()) > 0 {
				entry.PayloadJSON = string(evt.GetPayloadJson())
			}
			result.Events = append(result.Events, entry)
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// EventsListResourceTemplate defines the MCP resource template for campaign event listings.
func EventsListResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "campaign_events",
		Title:       "Campaign Events",
		Description: "Readable listing of all events for a campaign. URI format: campaign://{campaign_id}/events",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}/events",
	}
}

// EventsListPayload represents the MCP resource payload for campaign events.
type EventsListPayload struct {
	Events    []EventListEntry `json:"events"`
	TotalSize int              `json:"total_size"`
}

// EventsListResourceHandler returns a readable campaign events listing resource.
func EventsListResourceHandler(client statev1.EventServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("event service client is not configured")
		}

		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}/events")
		}
		uri := req.Params.URI

		campaignID, err := parseCampaignIDFromResourceURI(uri, "events")
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		// Fetch events with default pagination (newest first for readability)
		response, err := client.ListEvents(callCtx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   50,
			OrderBy:    "seq desc",
		})
		if err != nil {
			return nil, fmt.Errorf("list events failed: %w", err)
		}
		if response == nil {
			return nil, fmt.Errorf("list events response is missing")
		}

		payload := EventsListPayload{
			Events:    make([]EventListEntry, 0, len(response.GetEvents())),
			TotalSize: int(response.GetTotalSize()),
		}

		for _, evt := range response.GetEvents() {
			entry := EventListEntry{
				CampaignID:   evt.GetCampaignId(),
				Seq:          evt.GetSeq(),
				Hash:         evt.GetHash(),
				Timestamp:    formatTimestamp(evt.GetTs()),
				Type:         evt.GetType(),
				SessionID:    evt.GetSessionId(),
				RequestID:    evt.GetRequestId(),
				InvocationID: evt.GetInvocationId(),
				ActorType:    evt.GetActorType(),
				ActorID:      evt.GetActorId(),
				EntityType:   evt.GetEntityType(),
				EntityID:     evt.GetEntityId(),
			}
			if len(evt.GetPayloadJson()) > 0 {
				entry.PayloadJSON = string(evt.GetPayloadJson())
			}
			payload.Events = append(payload.Events, entry)
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal events: %w", err)
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
