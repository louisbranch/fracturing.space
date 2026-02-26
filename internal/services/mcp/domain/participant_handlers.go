package domain

import (
	"context"
	"fmt"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func ParticipantCreateHandler(client statev1.ParticipantServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[ParticipantCreateInput, ParticipantCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ParticipantCreateInput) (*mcp.CallToolResult, ParticipantCreateResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &statev1.CreateParticipantRequest{
			CampaignId: input.CampaignID,
			UserId:     input.UserID,
			Name:       input.Name,
			Role:       participantRoleFromString(input.Role),
			Pronouns:   input.Pronouns,
		}

		// Controller is optional; only set if provided
		if input.Controller != "" {
			req.Controller = controllerFromString(input.Controller)
		}

		response, err := client.CreateParticipant(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("participant create failed: %w", err)
		}
		if response == nil || response.Participant == nil {
			return nil, ParticipantCreateResult{}, fmt.Errorf("participant create response is missing")
		}

		result := ParticipantCreateResult{
			ID:         response.Participant.GetId(),
			CampaignID: response.Participant.GetCampaignId(),
			Name:       response.Participant.GetName(),
			Role:       participantRoleToString(response.Participant.GetRole()),
			Controller: controllerToString(response.Participant.GetController()),
			Pronouns:   response.Participant.GetPronouns(),
			CreatedAt:  formatTimestamp(response.Participant.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Participant.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/participants", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// ParticipantUpdateHandler executes a participant update request.
func ParticipantUpdateHandler(client statev1.ParticipantServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[ParticipantUpdateInput, ParticipantUpdateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ParticipantUpdateInput) (*mcp.CallToolResult, ParticipantUpdateResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		if input.CampaignID == "" {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("campaign_id is required")
		}
		if input.ParticipantID == "" {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("participant_id is required")
		}

		req := &statev1.UpdateParticipantRequest{
			CampaignId:    input.CampaignID,
			ParticipantId: input.ParticipantID,
		}
		if input.Name != nil {
			req.Name = wrapperspb.String(*input.Name)
		}
		if input.Role != nil {
			role := participantRoleFromString(*input.Role)
			if role == statev1.ParticipantRole_ROLE_UNSPECIFIED {
				return nil, ParticipantUpdateResult{}, fmt.Errorf("role must be GM or PLAYER")
			}
			req.Role = role
		}
		if input.Controller != nil {
			controller := controllerFromString(*input.Controller)
			if controller == statev1.Controller_CONTROLLER_UNSPECIFIED {
				return nil, ParticipantUpdateResult{}, fmt.Errorf("controller must be HUMAN or AI")
			}
			req.Controller = controller
		}
		if input.Pronouns != nil {
			req.Pronouns = wrapperspb.String(*input.Pronouns)
		}
		if req.Name == nil && req.Role == statev1.ParticipantRole_ROLE_UNSPECIFIED && req.Controller == statev1.Controller_CONTROLLER_UNSPECIFIED && req.Pronouns == nil {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("at least one field must be provided")
		}

		var header metadata.MD
		response, err := client.UpdateParticipant(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("participant update failed: %w", err)
		}
		if response == nil || response.Participant == nil {
			return nil, ParticipantUpdateResult{}, fmt.Errorf("participant update response is missing")
		}

		result := ParticipantUpdateResult{
			ID:         response.Participant.GetId(),
			CampaignID: response.Participant.GetCampaignId(),
			Name:       response.Participant.GetName(),
			Role:       participantRoleToString(response.Participant.GetRole()),
			Controller: controllerToString(response.Participant.GetController()),
			Pronouns:   response.Participant.GetPronouns(),
			CreatedAt:  formatTimestamp(response.Participant.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Participant.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/participants", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// ParticipantDeleteHandler executes a participant delete request.
func ParticipantDeleteHandler(client statev1.ParticipantServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[ParticipantDeleteInput, ParticipantDeleteResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ParticipantDeleteInput) (*mcp.CallToolResult, ParticipantDeleteResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		if input.CampaignID == "" {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("campaign_id is required")
		}
		if input.ParticipantID == "" {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("participant_id is required")
		}

		var header metadata.MD
		response, err := client.DeleteParticipant(callCtx, &statev1.DeleteParticipantRequest{
			CampaignId:    input.CampaignID,
			ParticipantId: input.ParticipantID,
			Reason:        input.Reason,
		}, grpc.Header(&header))
		if err != nil {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("participant delete failed: %w", err)
		}
		if response == nil || response.Participant == nil {
			return nil, ParticipantDeleteResult{}, fmt.Errorf("participant delete response is missing")
		}

		result := ParticipantDeleteResult{
			ID:         response.Participant.GetId(),
			CampaignID: response.Participant.GetCampaignId(),
			Name:       response.Participant.GetName(),
			Role:       participantRoleToString(response.Participant.GetRole()),
			Controller: controllerToString(response.Participant.GetController()),
			Pronouns:   response.Participant.GetPronouns(),
			CreatedAt:  formatTimestamp(response.Participant.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Participant.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/participants", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// participantRoleFromString maps MCP role names to domain enums while tolerating common casing.
func participantRoleFromString(value string) statev1.ParticipantRole {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "GM":
		return statev1.ParticipantRole_GM
	case "PLAYER":
		return statev1.ParticipantRole_PLAYER
	default:
		return statev1.ParticipantRole_ROLE_UNSPECIFIED
	}
}

// participantRoleToString converts participant roles back to stable MCP-visible values.
func participantRoleToString(role statev1.ParticipantRole) string {
	switch role {
	case statev1.ParticipantRole_GM:
		return "GM"
	case statev1.ParticipantRole_PLAYER:
		return "PLAYER"
	default:
		return "UNSPECIFIED"
	}
}

// controllerFromString parses controller kinds from MCP payload strings.
func controllerFromString(value string) statev1.Controller {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return statev1.Controller_CONTROLLER_HUMAN
	case "AI":
		return statev1.Controller_CONTROLLER_AI
	default:
		return statev1.Controller_CONTROLLER_UNSPECIFIED
	}
}

// controllerToString converts internal controller enums into MCP output form.
func controllerToString(controller statev1.Controller) string {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return "HUMAN"
	case statev1.Controller_CONTROLLER_AI:
		return "AI"
	default:
		return "UNSPECIFIED"
	}
}

// CharacterCreateHandler executes a character creation request.
