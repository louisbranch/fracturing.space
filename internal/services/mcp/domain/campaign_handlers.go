package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CampaignCreateHandler(client statev1.CampaignServiceClient, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignCreateInput, CampaignCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignCreateInput) (*mcp.CallToolResult, CampaignCreateResult, error) {
		callContext, err := newToolInvocationContext(ctx, nil)
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContext(callContext.RunCtx, callContext.InvocationID)
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		if strings.TrimSpace(input.UserID) != "" {
			callCtx = metadata.AppendToOutgoingContext(callCtx, grpcmeta.UserIDHeader, strings.TrimSpace(input.UserID))
		}

		var header metadata.MD

		response, err := client.CreateCampaign(callCtx, &statev1.CreateCampaignRequest{
			Name:         input.Name,
			System:       gameSystemFromString(input.System),
			GmMode:       gmModeFromString(input.GmMode),
			Intent:       campaignIntentFromString(input.Intent),
			AccessPolicy: campaignAccessPolicyFromString(input.AccessPolicy),
			ThemePrompt:  input.ThemePrompt,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create response is missing")
		}
		if response.OwnerParticipant == nil {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create response missing owner participant")
		}
		ownerParticipantID := response.OwnerParticipant.GetId()
		if ownerParticipantID == "" {
			return nil, CampaignCreateResult{}, fmt.Errorf("campaign create response missing owner participant id")
		}

		result := CampaignCreateResult{
			ID:                 response.Campaign.GetId(),
			OwnerParticipantID: ownerParticipantID,
			Name:               response.Campaign.GetName(),
			GmMode:             gmModeToString(response.Campaign.GetGmMode()),
			Intent:             campaignIntentToString(response.Campaign.GetIntent()),
			AccessPolicy:       campaignAccessPolicyToString(response.Campaign.GetAccessPolicy()),
			ParticipantCount:   int(response.Campaign.GetParticipantCount()),
			CharacterCount:     int(response.Campaign.GetCharacterCount()),
			GmFear:             0, // GM Fear is now in Snapshot, not Campaign
			ThemePrompt:        response.Campaign.GetThemePrompt(),
			Status:             campaignStatusToString(response.Campaign.GetStatus()),
			CreatedAt:          formatTimestamp(response.Campaign.GetCreatedAt()),
			UpdatedAt:          formatTimestamp(response.Campaign.GetUpdatedAt()),
			CompletedAt:        formatTimestamp(response.Campaign.GetCompletedAt()),
			ArchivedAt:         formatTimestamp(response.Campaign.GetArchivedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.ID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CampaignEndHandler executes a campaign end request.
func CampaignEndHandler(client statev1.CampaignServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignStatusChangeInput, CampaignStatusResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignStatusChangeInput) (*mcp.CallToolResult, CampaignStatusResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := input.CampaignID
		if campaignID == "" {
			campaignID = callContext.MCPContext.CampaignID
		}
		if campaignID == "" {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		response, err := client.EndCampaign(callCtx, &statev1.EndCampaignRequest{
			CampaignId: campaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign end failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign end response is missing")
		}

		result := campaignStatusResultFromProto(response.Campaign)
		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.ID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CampaignArchiveHandler executes a campaign archive request.
func CampaignArchiveHandler(client statev1.CampaignServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignStatusChangeInput, CampaignStatusResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignStatusChangeInput) (*mcp.CallToolResult, CampaignStatusResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := input.CampaignID
		if campaignID == "" {
			campaignID = callContext.MCPContext.CampaignID
		}
		if campaignID == "" {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		response, err := client.ArchiveCampaign(callCtx, &statev1.ArchiveCampaignRequest{
			CampaignId: campaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign archive failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign archive response is missing")
		}

		result := campaignStatusResultFromProto(response.Campaign)
		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.ID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CampaignRestoreHandler executes a campaign restore request.
func CampaignRestoreHandler(client statev1.CampaignServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CampaignStatusChangeInput, CampaignStatusResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CampaignStatusChangeInput) (*mcp.CallToolResult, CampaignStatusResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := input.CampaignID
		if campaignID == "" {
			campaignID = callContext.MCPContext.CampaignID
		}
		if campaignID == "" {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		response, err := client.RestoreCampaign(callCtx, &statev1.RestoreCampaignRequest{
			CampaignId: campaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign restore failed: %w", err)
		}
		if response == nil || response.Campaign == nil {
			return nil, CampaignStatusResult{}, fmt.Errorf("campaign restore response is missing")
		}

		result := campaignStatusResultFromProto(response.Campaign)
		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.ID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CampaignListResourceHandler returns a readable campaign listing resource.
// The resource intentionally returns one consolidated page while we migrate MCP clients
// to explicit paging controls; callers should not assume cursor-based pagination exists yet.
func CampaignListResourceHandler(client statev1.CampaignServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("campaign list client is not configured")
		}

		uri := CampaignListResource().URI
		if req != nil && req.Params != nil && req.Params.URI != "" {
			uri = req.Params.URI
		}

		runCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		// Campaign listings currently return a single snapshot page so MCP callers can
		// inspect canonical campaign metadata quickly during onboarding and local scripting.
		// TODO: Support page_size/page_token inputs and return next_page_token.
		// Without this, clients cannot resume listing after this first page and may
		// over-fetch when campaign catalogs grow.
		//
		// This is a temporary compromise to keep catalog responses stable while
		// richer UX pagination support is implemented in a later revision.
		// Pagination is intentionally deferred to keep MCP responses deterministic while
		// first-party surfaces agree on cursor semantics.
		payload := CampaignListPayload{}
		response, err := client.ListCampaigns(callCtx, &statev1.ListCampaignsRequest{
			PageSize: 10,
		})
		if err != nil {
			return nil, fmt.Errorf("campaign list failed: %w", err)
		}
		if response == nil {
			return nil, fmt.Errorf("campaign list response is missing")
		}

		for _, campaign := range response.GetCampaigns() {
			payload.Campaigns = append(payload.Campaigns, CampaignListEntry{
				ID:               campaign.GetId(),
				Name:             campaign.GetName(),
				Status:           campaignStatusToString(campaign.GetStatus()),
				GmMode:           gmModeToString(campaign.GetGmMode()),
				Intent:           campaignIntentToString(campaign.GetIntent()),
				AccessPolicy:     campaignAccessPolicyToString(campaign.GetAccessPolicy()),
				ParticipantCount: int(campaign.GetParticipantCount()),
				CharacterCount:   int(campaign.GetCharacterCount()),
				CanStartSession:  campaign.GetCanStartSession(),
				GmFear:           0, // GM Fear is now in Snapshot, not Campaign
				ThemePrompt:      campaign.GetThemePrompt(),
				CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
				UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
				CompletedAt:      formatTimestamp(campaign.GetCompletedAt()),
				ArchivedAt:       formatTimestamp(campaign.GetArchivedAt()),
			})
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal campaign list: %w", err)
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

// formatTimestamp returns an RFC3339 timestamp or empty string.
// Empty values are treated as missing fields for compact API responses.
func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format(time.RFC3339)
}

// gameSystemFromString normalizes MCP input into a canonical game-system enum.
// Unknown values degrade to UNSPECIFIED so input validation can reject unsupported
// systems in a controlled and observable way.
func gameSystemFromString(value string) commonv1.GameSystem {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "DAGGERHEART", "GAME_SYSTEM_DAGGERHEART":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
}

// gmModeFromString parses MCP input into gm mode, accepting loose variants and
// normalizing case/spacing so user-facing callers stay ergonomic.
func gmModeFromString(value string) statev1.GmMode {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return statev1.GmMode_HUMAN
	case "AI":
		return statev1.GmMode_AI
	case "HYBRID":
		return statev1.GmMode_HYBRID
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED
	}
}

// gmModeToString converts internal gm-mode enums into deterministic MCP output values.
func gmModeToString(mode statev1.GmMode) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return "HUMAN"
	case statev1.GmMode_AI:
		return "AI"
	case statev1.GmMode_HYBRID:
		return "HYBRID"
	default:
		return "UNSPECIFIED"
	}
}

// campaignIntentFromString parses MCP campaign intent values while accepting common aliases.
func campaignIntentFromString(value string) statev1.CampaignIntent {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "STANDARD", "CAMPAIGN_INTENT_STANDARD":
		return statev1.CampaignIntent_STANDARD
	case "STARTER", "CAMPAIGN_INTENT_STARTER":
		return statev1.CampaignIntent_STARTER
	case "SANDBOX", "CAMPAIGN_INTENT_SANDBOX":
		return statev1.CampaignIntent_SANDBOX
	default:
		return statev1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED
	}
}

// campaignIntentToString converts internal campaign intent enums into MCP wire format.
func campaignIntentToString(intent statev1.CampaignIntent) string {
	switch intent {
	case statev1.CampaignIntent_STANDARD:
		return "STANDARD"
	case statev1.CampaignIntent_STARTER:
		return "STARTER"
	case statev1.CampaignIntent_SANDBOX:
		return "SANDBOX"
	default:
		return "UNSPECIFIED"
	}
}

// campaignAccessPolicyFromString maps MCP text to the campaign access policy enum.
func campaignAccessPolicyFromString(value string) statev1.CampaignAccessPolicy {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PRIVATE", "CAMPAIGN_ACCESS_POLICY_PRIVATE":
		return statev1.CampaignAccessPolicy_PRIVATE
	case "RESTRICTED", "CAMPAIGN_ACCESS_POLICY_RESTRICTED":
		return statev1.CampaignAccessPolicy_RESTRICTED
	case "PUBLIC", "CAMPAIGN_ACCESS_POLICY_PUBLIC":
		return statev1.CampaignAccessPolicy_PUBLIC
	default:
		return statev1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED
	}
}

// campaignAccessPolicyToString converts internal campaign access policy enums for output.
func campaignAccessPolicyToString(policy statev1.CampaignAccessPolicy) string {
	switch policy {
	case statev1.CampaignAccessPolicy_PRIVATE:
		return "PRIVATE"
	case statev1.CampaignAccessPolicy_RESTRICTED:
		return "RESTRICTED"
	case statev1.CampaignAccessPolicy_PUBLIC:
		return "PUBLIC"
	default:
		return "UNSPECIFIED"
	}
}

// campaignStatusToString converts internal campaign status enums for MCP consumers.
func campaignStatusToString(status statev1.CampaignStatus) string {
	switch status {
	case statev1.CampaignStatus_DRAFT:
		return "DRAFT"
	case statev1.CampaignStatus_ACTIVE:
		return "ACTIVE"
	case statev1.CampaignStatus_COMPLETED:
		return "COMPLETED"
	case statev1.CampaignStatus_ARCHIVED:
		return "ARCHIVED"
	default:
		return "UNSPECIFIED"
	}
}

func campaignStatusResultFromProto(campaign *statev1.Campaign) CampaignStatusResult {
	return CampaignStatusResult{
		ID:               campaign.GetId(),
		Name:             campaign.GetName(),
		GmMode:           gmModeToString(campaign.GetGmMode()),
		Intent:           campaignIntentToString(campaign.GetIntent()),
		AccessPolicy:     campaignAccessPolicyToString(campaign.GetAccessPolicy()),
		ParticipantCount: int(campaign.GetParticipantCount()),
		CharacterCount:   int(campaign.GetCharacterCount()),
		CanStartSession:  campaign.GetCanStartSession(),
		GmFear:           0, // GM Fear is now in Snapshot, not Campaign
		ThemePrompt:      campaign.GetThemePrompt(),
		Status:           campaignStatusToString(campaign.GetStatus()),
		CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
		UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
		CompletedAt:      formatTimestamp(campaign.GetCompletedAt()),
		ArchivedAt:       formatTimestamp(campaign.GetArchivedAt()),
	}
}

// ParticipantCreateHandler executes a participant creation request.
