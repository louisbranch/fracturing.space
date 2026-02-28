package domain

import (
	"context"
	"fmt"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func CharacterCreateHandler(client statev1.CharacterServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterCreateInput, CharacterCreateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterCreateInput) (*mcp.CallToolResult, CharacterCreateResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &statev1.CreateCharacterRequest{
			CampaignId: input.CampaignID,
			Name:       input.Name,
			Kind:       characterKindFromString(input.Kind),
			Notes:      input.Notes,
			Pronouns:   input.Pronouns,
			Aliases:    append([]string(nil), input.Aliases...),
		}

		response, err := client.CreateCharacter(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("character create failed: %w", err)
		}
		if response == nil || response.Character == nil {
			return nil, CharacterCreateResult{}, fmt.Errorf("character create response is missing")
		}

		result := CharacterCreateResult{
			ID:         response.Character.GetId(),
			CampaignID: response.Character.GetCampaignId(),
			Name:       response.Character.GetName(),
			Kind:       characterKindToString(response.Character.GetKind()),
			Notes:      response.Character.GetNotes(),
			Pronouns:   response.Character.GetPronouns(),
			Aliases:    mcpAliases(response.Character.GetAliases()),
			CreatedAt:  formatTimestamp(response.Character.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Character.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/characters", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterUpdateHandler executes a character update request.
func CharacterUpdateHandler(client statev1.CharacterServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterUpdateInput, CharacterUpdateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterUpdateInput) (*mcp.CallToolResult, CharacterUpdateResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		if input.CampaignID == "" {
			return nil, CharacterUpdateResult{}, fmt.Errorf("campaign_id is required")
		}
		if input.CharacterID == "" {
			return nil, CharacterUpdateResult{}, fmt.Errorf("character_id is required")
		}

		req := &statev1.UpdateCharacterRequest{
			CampaignId:  input.CampaignID,
			CharacterId: input.CharacterID,
		}
		if input.Name != nil {
			req.Name = wrapperspb.String(*input.Name)
		}
		if input.Kind != nil {
			kind := characterKindFromString(*input.Kind)
			if kind == statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
				return nil, CharacterUpdateResult{}, fmt.Errorf("kind must be PC or NPC")
			}
			req.Kind = kind
		}
		if input.Notes != nil {
			req.Notes = wrapperspb.String(*input.Notes)
		}
		if input.Pronouns != nil {
			req.Pronouns = wrapperspb.String(*input.Pronouns)
		}
		if input.Aliases != nil {
			req.Aliases = mcpAliases(*input.Aliases)
		}
		if req.Name == nil && req.Kind == statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED && req.Notes == nil && req.Pronouns == nil && input.Aliases == nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("at least one field must be provided")
		}

		var header metadata.MD
		response, err := client.UpdateCharacter(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("character update failed: %w", err)
		}
		if response == nil || response.Character == nil {
			return nil, CharacterUpdateResult{}, fmt.Errorf("character update response is missing")
		}

		result := CharacterUpdateResult{
			ID:         response.Character.GetId(),
			CampaignID: response.Character.GetCampaignId(),
			Name:       response.Character.GetName(),
			Kind:       characterKindToString(response.Character.GetKind()),
			Notes:      response.Character.GetNotes(),
			Pronouns:   response.Character.GetPronouns(),
			Aliases:    mcpAliases(response.Character.GetAliases()),
			CreatedAt:  formatTimestamp(response.Character.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Character.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/characters", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterDeleteHandler executes a character delete request.
func CharacterDeleteHandler(client statev1.CharacterServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterDeleteInput, CharacterDeleteResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterDeleteInput) (*mcp.CallToolResult, CharacterDeleteResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CharacterDeleteResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CharacterDeleteResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		if input.CampaignID == "" {
			return nil, CharacterDeleteResult{}, fmt.Errorf("campaign_id is required")
		}
		if input.CharacterID == "" {
			return nil, CharacterDeleteResult{}, fmt.Errorf("character_id is required")
		}

		var header metadata.MD
		response, err := client.DeleteCharacter(callCtx, &statev1.DeleteCharacterRequest{
			CampaignId:  input.CampaignID,
			CharacterId: input.CharacterID,
			Reason:      input.Reason,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CharacterDeleteResult{}, fmt.Errorf("character delete failed: %w", err)
		}
		if response == nil || response.Character == nil {
			return nil, CharacterDeleteResult{}, fmt.Errorf("character delete response is missing")
		}

		result := CharacterDeleteResult{
			ID:         response.Character.GetId(),
			CampaignID: response.Character.GetCampaignId(),
			Name:       response.Character.GetName(),
			Kind:       characterKindToString(response.Character.GetKind()),
			Notes:      response.Character.GetNotes(),
			Pronouns:   response.Character.GetPronouns(),
			Aliases:    mcpAliases(response.Character.GetAliases()),
			CreatedAt:  formatTimestamp(response.Character.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(response.Character.GetUpdatedAt()),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			CampaignListResource().URI,
			fmt.Sprintf("campaign://%s", result.CampaignID),
			fmt.Sprintf("campaign://%s/characters", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// characterKindFromString parses MCP character kinds and maps unknown kinds to UNSPECIFIED.
func characterKindFromString(value string) statev1.CharacterKind {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PC":
		return statev1.CharacterKind_PC
	case "NPC":
		return statev1.CharacterKind_NPC
	default:
		return statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}

// characterKindToString converts internal character kinds into MCP output values.
func characterKindToString(kind statev1.CharacterKind) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return "PC"
	case statev1.CharacterKind_NPC:
		return "NPC"
	default:
		return "UNSPECIFIED"
	}
}

// CharacterControlSetHandler executes a character control set request.
func CharacterControlSetHandler(client statev1.CharacterServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterControlSetInput, CharacterControlSetResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterControlSetInput) (*mcp.CallToolResult, CharacterControlSetResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &statev1.SetDefaultControlRequest{
			CampaignId:    input.CampaignID,
			CharacterId:   input.CharacterID,
			ParticipantId: wrapperspb.String(input.ParticipantID),
		}

		response, err := client.SetDefaultControl(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("character control set failed: %w", err)
		}
		if response == nil {
			return nil, CharacterControlSetResult{}, fmt.Errorf("character control set response is missing")
		}

		participantID := ""
		if response.GetParticipantId() != nil {
			participantID = response.GetParticipantId().GetValue()
		}
		result := CharacterControlSetResult{
			CampaignID:    response.GetCampaignId(),
			CharacterID:   response.GetCharacterId(),
			ParticipantID: participantID,
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			fmt.Sprintf("campaign://%s/characters", result.CampaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterSheetGetHandler executes a character sheet get request.
func CharacterSheetGetHandler(client statev1.CharacterServiceClient, getContext func() Context) mcp.ToolHandlerFor[CharacterSheetGetInput, CharacterSheetGetResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterSheetGetInput) (*mcp.CallToolResult, CharacterSheetGetResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := callContext.MCPContext.CampaignID
		if campaignID == "" {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("campaign context is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(callContext.RunCtx, callContext.InvocationID)
		if err != nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.GetCharacterSheet(callCtx, &statev1.GetCharacterSheetRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("character sheet get failed: %w", err)
		}
		if response == nil {
			return nil, CharacterSheetGetResult{}, fmt.Errorf("character sheet response is missing")
		}

		result := CharacterSheetGetResult{
			Character: CharacterCreateResult{
				ID:         response.Character.GetId(),
				CampaignID: response.Character.GetCampaignId(),
				Name:       response.Character.GetName(),
				Kind:       characterKindToString(response.Character.GetKind()),
				Notes:      response.Character.GetNotes(),
				Pronouns:   response.Character.GetPronouns(),
				Aliases:    mcpAliases(response.Character.GetAliases()),
				CreatedAt:  formatTimestamp(response.Character.GetCreatedAt()),
				UpdatedAt:  formatTimestamp(response.Character.GetUpdatedAt()),
			},
			Profile: characterProfileResultFromProto(response.Profile),
			State:   characterStateResultFromProto(response.State),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

func mcpAliases(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

// CharacterProfilePatchHandler executes a character profile patch request.
func CharacterProfilePatchHandler(client statev1.CharacterServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterProfilePatchInput, CharacterProfilePatchResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterProfilePatchInput) (*mcp.CallToolResult, CharacterProfilePatchResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := callContext.MCPContext.CampaignID
		if campaignID == "" {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("campaign context is required")
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &statev1.PatchCharacterProfileRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
		}

		// All profile fields are now Daggerheart-specific (including hp_max)
		hasDaggerheartPatch := input.HpMax != nil || input.StressMax != nil || input.Evasion != nil ||
			input.MajorThreshold != nil || input.SevereThreshold != nil
		if hasDaggerheartPatch {
			dhProfile := &daggerheartv1.DaggerheartProfile{}
			if input.HpMax != nil {
				dhProfile.HpMax = int32(*input.HpMax)
			}
			if input.StressMax != nil {
				dhProfile.StressMax = wrapperspb.Int32(int32(*input.StressMax))
			}
			if input.Evasion != nil {
				dhProfile.Evasion = wrapperspb.Int32(int32(*input.Evasion))
			}
			if input.MajorThreshold != nil {
				dhProfile.MajorThreshold = wrapperspb.Int32(int32(*input.MajorThreshold))
			}
			if input.SevereThreshold != nil {
				dhProfile.SevereThreshold = wrapperspb.Int32(int32(*input.SevereThreshold))
			}
			req.SystemProfilePatch = &statev1.PatchCharacterProfileRequest_Daggerheart{
				Daggerheart: dhProfile,
			}
		}

		response, err := client.PatchCharacterProfile(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("character profile patch failed: %w", err)
		}
		if response == nil || response.Profile == nil {
			return nil, CharacterProfilePatchResult{}, fmt.Errorf("character profile patch response is missing")
		}

		result := CharacterProfilePatchResult{
			Profile: characterProfileResultFromProto(response.Profile),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			fmt.Sprintf("campaign://%s/characters", campaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// CharacterCreationWorkflowApplyHandler executes an atomic creation workflow apply.
func CharacterCreationWorkflowApplyHandler(client statev1.CharacterServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterCreationWorkflowApplyInput, CharacterCreationWorkflowApplyResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterCreationWorkflowApplyInput) (*mcp.CallToolResult, CharacterCreationWorkflowApplyResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CharacterCreationWorkflowApplyResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := callContext.MCPContext.CampaignID
		if campaignID == "" {
			return nil, CharacterCreationWorkflowApplyResult{}, fmt.Errorf("campaign context is required")
		}

		callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, CharacterCreationWorkflowApplyResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		experiences := make([]*daggerheartv1.DaggerheartExperience, 0, len(input.Experiences))
		for _, experience := range input.Experiences {
			experiences = append(experiences, &daggerheartv1.DaggerheartExperience{
				Name:     experience.Name,
				Modifier: int32(experience.Modifier),
			})
		}

		req := &statev1.ApplyCharacterCreationWorkflowRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
			SystemWorkflow: &statev1.ApplyCharacterCreationWorkflowRequest_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartCreationWorkflowInput{
					ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{
						ClassId:    input.ClassID,
						SubclassId: input.SubclassID,
					},
					HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{
						AncestryId:  input.AncestryID,
						CommunityId: input.CommunityID,
					},
					TraitsInput:  &daggerheartv1.DaggerheartCreationStepTraitsInput{Agility: int32(input.Agility), Strength: int32(input.Strength), Finesse: int32(input.Finesse), Instinct: int32(input.Instinct), Presence: int32(input.Presence), Knowledge: int32(input.Knowledge)},
					DetailsInput: &daggerheartv1.DaggerheartCreationStepDetailsInput{},
					EquipmentInput: &daggerheartv1.DaggerheartCreationStepEquipmentInput{
						WeaponIds:    append([]string(nil), input.WeaponIDs...),
						ArmorId:      input.ArmorID,
						PotionItemId: input.PotionItemID,
					},
					BackgroundInput: &daggerheartv1.DaggerheartCreationStepBackgroundInput{Background: input.Background},
					ExperiencesInput: &daggerheartv1.DaggerheartCreationStepExperiencesInput{
						Experiences: experiences,
					},
					DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{DomainCardIds: append([]string(nil), input.DomainCardIDs...)},
					ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{Connections: input.Connections},
				},
			},
		}

		var header metadata.MD
		response, err := client.ApplyCharacterCreationWorkflow(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterCreationWorkflowApplyResult{}, fmt.Errorf("character creation workflow apply failed: %w", err)
		}
		if response == nil || response.Profile == nil || response.Progress == nil {
			return nil, CharacterCreationWorkflowApplyResult{}, fmt.Errorf("character creation workflow apply response is missing")
		}

		result := CharacterCreationWorkflowApplyResult{
			Profile:  characterProfileResultFromProto(response.Profile),
			Progress: characterCreationProgressResultFromProto(response.Progress),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			fmt.Sprintf("campaign://%s/characters", campaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

func characterCreationProgressResultFromProto(progress *statev1.CharacterCreationProgress) CharacterCreationProgressResult {
	if progress == nil {
		return CharacterCreationProgressResult{UnmetReasons: []string{}, Steps: []CharacterCreationStepProgressResult{}}
	}
	steps := make([]CharacterCreationStepProgressResult, 0, len(progress.GetSteps()))
	for _, step := range progress.GetSteps() {
		steps = append(steps, CharacterCreationStepProgressResult{
			Step:     int(step.GetStep()),
			Key:      step.GetKey(),
			Complete: step.GetComplete(),
		})
	}
	unmetReasons := make([]string, 0, len(progress.GetUnmetReasons()))
	unmetReasons = append(unmetReasons, progress.GetUnmetReasons()...)
	return CharacterCreationProgressResult{
		CharacterID:  progress.GetCharacterId(),
		Ready:        progress.GetReady(),
		NextStep:     int(progress.GetNextStep()),
		UnmetReasons: unmetReasons,
		Steps:        steps,
	}
}

// CharacterStatePatchHandler executes a character state patch request.
func CharacterStatePatchHandler(client statev1.SnapshotServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[CharacterStatePatchInput, CharacterStatePatchResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CharacterStatePatchInput) (*mcp.CallToolResult, CharacterStatePatchResult, error) {
		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		campaignID := callContext.MCPContext.CampaignID
		if campaignID == "" {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("campaign context is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(callContext.RunCtx, callContext.InvocationID)
		if err != nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		req := &statev1.PatchCharacterStateRequest{
			CampaignId:  campaignID,
			CharacterId: input.CharacterID,
		}

		// All state fields are now Daggerheart-specific (including HP)
		if input.Hp != nil || input.Hope != nil || input.Stress != nil {
			dhState := &daggerheartv1.DaggerheartCharacterState{}
			if input.Hp != nil {
				dhState.Hp = int32(*input.Hp)
			}
			if input.Hope != nil {
				dhState.Hope = int32(*input.Hope)
			}
			if input.Stress != nil {
				dhState.Stress = int32(*input.Stress)
			}
			req.SystemStatePatch = &statev1.PatchCharacterStateRequest_Daggerheart{
				Daggerheart: dhState,
			}
		}

		response, err := client.PatchCharacterState(callCtx, req, grpc.Header(&header))
		if err != nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("character state patch failed: %w", err)
		}
		if response == nil || response.State == nil {
			return nil, CharacterStatePatchResult{}, fmt.Errorf("character state patch response is missing")
		}

		result := CharacterStatePatchResult{
			State: characterStateResultFromProto(response.State),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		NotifyResourceUpdates(
			ctx,
			notify,
			fmt.Sprintf("campaign://%s/characters", campaignID),
		)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// ParticipantListResourceHandler returns a readable participant listing resource.
