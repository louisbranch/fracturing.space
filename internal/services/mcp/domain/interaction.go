package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const interactionStateResourceURITemplate = "campaign://{campaign_id}/interaction"

// InteractionViewerResult exposes the active interaction viewer identity.
type InteractionViewerResult struct {
	ParticipantID string `json:"participant_id,omitempty"`
	Name          string `json:"name,omitempty"`
	Role          string `json:"role,omitempty"`
}

// InteractionSessionResult exposes the active session summary.
type InteractionSessionResult struct {
	SessionID string `json:"session_id,omitempty"`
	Name      string `json:"name,omitempty"`
}

// InteractionCharacterResult exposes a scene character entry.
type InteractionCharacterResult struct {
	CharacterID        string `json:"character_id,omitempty"`
	Name               string `json:"name,omitempty"`
	OwnerParticipantID string `json:"owner_participant_id,omitempty"`
}

// InteractionSceneResult exposes the active scene summary.
type InteractionSceneResult struct {
	SceneID     string                       `json:"scene_id,omitempty"`
	SessionID   string                       `json:"session_id,omitempty"`
	Name        string                       `json:"name,omitempty"`
	Description string                       `json:"description,omitempty"`
	Characters  []InteractionCharacterResult `json:"characters,omitempty"`
}

// InteractionPlayerSlotResult exposes one participant-owned scene slot.
type InteractionPlayerSlotResult struct {
	ParticipantID      string   `json:"participant_id,omitempty"`
	SummaryText        string   `json:"summary_text,omitempty"`
	CharacterIDs       []string `json:"character_ids,omitempty"`
	UpdatedAt          string   `json:"updated_at,omitempty"`
	Yielded            bool     `json:"yielded,omitempty"`
	ReviewStatus       string   `json:"review_status,omitempty"`
	ReviewReason       string   `json:"review_reason,omitempty"`
	ReviewCharacterIDs []string `json:"review_character_ids,omitempty"`
}

// InteractionPlayerPhaseResult exposes the current player-phase state.
type InteractionPlayerPhaseResult struct {
	PhaseID              string                        `json:"phase_id,omitempty"`
	Status               string                        `json:"status,omitempty"`
	FrameText            string                        `json:"frame_text,omitempty"`
	ActingCharacterIDs   []string                      `json:"acting_character_ids,omitempty"`
	ActingParticipantIDs []string                      `json:"acting_participant_ids,omitempty"`
	Slots                []InteractionPlayerSlotResult `json:"slots,omitempty"`
}

// InteractionOOCPostResult exposes one OOC transcript post.
type InteractionOOCPostResult struct {
	PostID        string `json:"post_id,omitempty"`
	ParticipantID string `json:"participant_id,omitempty"`
	Body          string `json:"body,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
}

// InteractionOOCStateResult exposes the session OOC overlay.
type InteractionOOCStateResult struct {
	Open                        bool                       `json:"open"`
	Posts                       []InteractionOOCPostResult `json:"posts,omitempty"`
	ReadyToResumeParticipantIDs []string                   `json:"ready_to_resume_participant_ids,omitempty"`
}

// InteractionStateResult exposes the authoritative scene interaction state.
type InteractionStateResult struct {
	CampaignID    string                       `json:"campaign_id,omitempty"`
	CampaignName  string                       `json:"campaign_name,omitempty"`
	Locale        string                       `json:"locale,omitempty"`
	Viewer        InteractionViewerResult      `json:"viewer"`
	ActiveSession InteractionSessionResult     `json:"active_session"`
	ActiveScene   InteractionSceneResult       `json:"active_scene"`
	PlayerPhase   InteractionPlayerPhaseResult `json:"player_phase"`
	OOC           InteractionOOCStateResult    `json:"ooc"`
}

// InteractionStateResourceTemplate defines the MCP resource template for the
// authoritative interaction snapshot.
func InteractionStateResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "interaction_state",
		Title:       "Interaction State",
		Description: "Readable active-play interaction snapshot. URI format: campaign://{campaign_id}/interaction",
		MIMEType:    "application/json",
		URITemplate: interactionStateResourceURITemplate,
	}
}

// InteractionSetActiveSceneInput sets the active scene for the current session.
type InteractionSetActiveSceneInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SceneID    string `json:"scene_id" jsonschema:"scene identifier"`
}

// InteractionStartScenePlayerPhaseInput starts a player phase on the active scene.
type InteractionStartScenePlayerPhaseInput struct {
	CampaignID   string   `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SceneID      string   `json:"scene_id,omitempty" jsonschema:"scene identifier (defaults to active scene)"`
	FrameText    string   `json:"frame_text" jsonschema:"GM frame text shown to acting players"`
	CharacterIDs []string `json:"character_ids" jsonschema:"acting character identifiers"`
}

// InteractionSubmitScenePlayerPostInput updates the caller's committed post.
type InteractionSubmitScenePlayerPostInput struct {
	CampaignID     string   `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SceneID        string   `json:"scene_id,omitempty" jsonschema:"scene identifier (defaults to active scene)"`
	SummaryText    string   `json:"summary_text" jsonschema:"committed summary of the participant action"`
	CharacterIDs   []string `json:"character_ids,omitempty" jsonschema:"character identifiers referenced by the post"`
	YieldAfterPost bool     `json:"yield_after_post,omitempty" jsonschema:"whether the participant also yields after posting"`
}

// InteractionScenePhaseInput targets the active scene phase.
type InteractionScenePhaseInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SceneID    string `json:"scene_id,omitempty" jsonschema:"scene identifier (defaults to active scene)"`
}

// InteractionEndScenePlayerPhaseInput ends the current player phase early.
type InteractionEndScenePlayerPhaseInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SceneID    string `json:"scene_id,omitempty" jsonschema:"scene identifier (defaults to active scene)"`
	Reason     string `json:"reason,omitempty" jsonschema:"optional GM-supplied reason"`
}

// InteractionAcceptScenePlayerPhaseInput accepts the current reviewed phase.
type InteractionAcceptScenePlayerPhaseInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SceneID    string `json:"scene_id,omitempty" jsonschema:"scene identifier (defaults to active scene)"`
}

// InteractionScenePlayerRevisionInput targets one participant slot for revisions.
type InteractionScenePlayerRevisionInput struct {
	ParticipantID string   `json:"participant_id" jsonschema:"participant identifier that must revise their slot"`
	Reason        string   `json:"reason" jsonschema:"GM review reason shown to the participant"`
	CharacterIDs  []string `json:"character_ids,omitempty" jsonschema:"optional character identifiers affected by the review request"`
}

// InteractionRequestScenePlayerRevisionsInput requests participant slot revisions.
type InteractionRequestScenePlayerRevisionsInput struct {
	CampaignID string                                `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SceneID    string                                `json:"scene_id,omitempty" jsonschema:"scene identifier (defaults to active scene)"`
	Revisions  []InteractionScenePlayerRevisionInput `json:"revisions" jsonschema:"participant-scoped revision requests"`
}

// InteractionPauseOOCInput opens the session OOC overlay.
type InteractionPauseOOCInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	Reason     string `json:"reason,omitempty" jsonschema:"optional OOC pause reason"`
}

// InteractionPostOOCInput posts one OOC transcript line.
type InteractionPostOOCInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	Body       string `json:"body" jsonschema:"out-of-character message body"`
}

// InteractionSetActiveSceneTool defines the MCP tool schema.
func InteractionSetActiveSceneTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_active_scene_set",
		Description: "Sets the authoritative active scene for the current session",
	}
}

// InteractionStartScenePlayerPhaseTool defines the MCP tool schema.
func InteractionStartScenePlayerPhaseTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_scene_player_phase_start",
		Description: "Starts a new player phase on the active scene from a GM frame",
	}
}

// InteractionSubmitScenePlayerPostTool defines the MCP tool schema.
func InteractionSubmitScenePlayerPostTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_scene_player_post_submit",
		Description: "Commits one participant action summary in the active scene player phase",
	}
}

// InteractionYieldScenePlayerPhaseTool defines the MCP tool schema.
func InteractionYieldScenePlayerPhaseTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_scene_player_phase_yield",
		Description: "Marks the caller as yielded in the active scene player phase",
	}
}

// InteractionUnyieldScenePlayerPhaseTool defines the MCP tool schema.
func InteractionUnyieldScenePlayerPhaseTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_scene_player_phase_unyield",
		Description: "Clears the caller's yielded state in the active scene player phase",
	}
}

// InteractionEndScenePlayerPhaseTool defines the MCP tool schema.
func InteractionEndScenePlayerPhaseTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_scene_player_phase_end",
		Description: "Ends the active scene player phase early under GM control",
	}
}

// InteractionAcceptScenePlayerPhaseTool defines the MCP tool schema.
func InteractionAcceptScenePlayerPhaseTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_scene_player_phase_accept",
		Description: "Accepts the active scene player phase after GM review and returns authority to the GM",
	}
}

// InteractionRequestScenePlayerRevisionsTool defines the MCP tool schema.
func InteractionRequestScenePlayerRevisionsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_scene_player_revisions_request",
		Description: "Requests revisions for one or more participant slots in the active scene player phase",
	}
}

// InteractionPauseOOCTool defines the MCP tool schema.
func InteractionPauseOOCTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_ooc_pause",
		Description: "Opens the session-level out-of-character pause overlay",
	}
}

// InteractionPostOOCTool defines the MCP tool schema.
func InteractionPostOOCTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_ooc_post",
		Description: "Posts one append-only out-of-character transcript message",
	}
}

// InteractionMarkOOCReadyTool defines the MCP tool schema.
func InteractionMarkOOCReadyTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_ooc_ready_mark",
		Description: "Marks the caller as ready to resume from the current OOC pause",
	}
}

// InteractionClearOOCReadyTool defines the MCP tool schema.
func InteractionClearOOCReadyTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_ooc_ready_clear",
		Description: "Clears the caller's ready-to-resume state for the current OOC pause",
	}
}

// InteractionResumeOOCTool defines the MCP tool schema.
func InteractionResumeOOCTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "interaction_ooc_resume",
		Description: "Resumes in-character scene play from the current OOC pause",
	}
}

// InteractionSetActiveSceneHandler executes an active-scene update.
func InteractionSetActiveSceneHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionSetActiveSceneInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionSetActiveSceneInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		var header metadata.MD
		response, err := client.SetActiveScene(callCtx.RunCtx, &statev1.SetActiveSceneRequest{
			CampaignId: campaignID,
			SceneId:    strings.TrimSpace(input.SceneID),
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("set active scene failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("set active scene response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionStartScenePlayerPhaseHandler starts a player phase through MCP.
func InteractionStartScenePlayerPhaseHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionStartScenePlayerPhaseInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionStartScenePlayerPhaseInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		sceneID, err := resolveInteractionSceneID(callCtx.RunCtx, client, campaignID, input.SceneID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}

		var header metadata.MD
		response, err := client.StartScenePlayerPhase(callCtx.RunCtx, &statev1.StartScenePlayerPhaseRequest{
			CampaignId:   campaignID,
			SceneId:      sceneID,
			FrameText:    strings.TrimSpace(input.FrameText),
			CharacterIds: input.CharacterIDs,
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("start scene player phase failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("start scene player phase response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionSubmitScenePlayerPostHandler commits one participant post.
func InteractionSubmitScenePlayerPostHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionSubmitScenePlayerPostInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionSubmitScenePlayerPostInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		sceneID, err := resolveInteractionSceneID(callCtx.RunCtx, client, campaignID, input.SceneID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}

		var header metadata.MD
		response, err := client.SubmitScenePlayerPost(callCtx.RunCtx, &statev1.SubmitScenePlayerPostRequest{
			CampaignId:     campaignID,
			SceneId:        sceneID,
			SummaryText:    strings.TrimSpace(input.SummaryText),
			CharacterIds:   input.CharacterIDs,
			YieldAfterPost: input.YieldAfterPost,
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("submit scene player post failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("submit scene player post response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionYieldScenePlayerPhaseHandler yields the active player phase.
func InteractionYieldScenePlayerPhaseHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionScenePhaseInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionScenePhaseInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		sceneID, err := resolveInteractionSceneID(callCtx.RunCtx, client, campaignID, input.SceneID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}

		var header metadata.MD
		response, err := client.YieldScenePlayerPhase(callCtx.RunCtx, &statev1.YieldScenePlayerPhaseRequest{
			CampaignId: campaignID,
			SceneId:    sceneID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("yield scene player phase failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("yield scene player phase response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionUnyieldScenePlayerPhaseHandler clears the yielded state.
func InteractionUnyieldScenePlayerPhaseHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionScenePhaseInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionScenePhaseInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		sceneID, err := resolveInteractionSceneID(callCtx.RunCtx, client, campaignID, input.SceneID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}

		var header metadata.MD
		response, err := client.UnyieldScenePlayerPhase(callCtx.RunCtx, &statev1.UnyieldScenePlayerPhaseRequest{
			CampaignId: campaignID,
			SceneId:    sceneID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("unyield scene player phase failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("unyield scene player phase response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionEndScenePlayerPhaseHandler ends the player phase.
func InteractionEndScenePlayerPhaseHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionEndScenePlayerPhaseInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionEndScenePlayerPhaseInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		sceneID, err := resolveInteractionSceneID(callCtx.RunCtx, client, campaignID, input.SceneID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}

		var header metadata.MD
		response, err := client.EndScenePlayerPhase(callCtx.RunCtx, &statev1.EndScenePlayerPhaseRequest{
			CampaignId: campaignID,
			SceneId:    sceneID,
			Reason:     strings.TrimSpace(input.Reason),
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("end scene player phase failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("end scene player phase response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionAcceptScenePlayerPhaseHandler accepts the current reviewed phase.
func InteractionAcceptScenePlayerPhaseHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionAcceptScenePlayerPhaseInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionAcceptScenePlayerPhaseInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		sceneID, err := resolveInteractionSceneID(callCtx.RunCtx, client, campaignID, input.SceneID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}

		var header metadata.MD
		response, err := client.AcceptScenePlayerPhase(callCtx.RunCtx, &statev1.AcceptScenePlayerPhaseRequest{
			CampaignId: campaignID,
			SceneId:    sceneID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("accept scene player phase failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("accept scene player phase response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionRequestScenePlayerRevisionsHandler requests revisions for one or more slots.
func InteractionRequestScenePlayerRevisionsHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionRequestScenePlayerRevisionsInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionRequestScenePlayerRevisionsInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		sceneID, err := resolveInteractionSceneID(callCtx.RunCtx, client, campaignID, input.SceneID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}

		revisions := make([]*statev1.ScenePlayerRevisionRequest, 0, len(input.Revisions))
		for _, revision := range input.Revisions {
			revisions = append(revisions, &statev1.ScenePlayerRevisionRequest{
				ParticipantId: strings.TrimSpace(revision.ParticipantID),
				Reason:        strings.TrimSpace(revision.Reason),
				CharacterIds:  append([]string(nil), revision.CharacterIDs...),
			})
		}

		var header metadata.MD
		response, err := client.RequestScenePlayerRevisions(callCtx.RunCtx, &statev1.RequestScenePlayerRevisionsRequest{
			CampaignId: campaignID,
			SceneId:    sceneID,
			Revisions:  revisions,
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("request scene player revisions failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("request scene player revisions response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionPauseOOCHandler opens the OOC overlay.
func InteractionPauseOOCHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionPauseOOCInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionPauseOOCInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		var header metadata.MD
		response, err := client.PauseSessionForOOC(callCtx.RunCtx, &statev1.PauseSessionForOOCRequest{
			CampaignId: campaignID,
			Reason:     strings.TrimSpace(input.Reason),
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("pause session for ooc failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("pause session for ooc response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionPostOOCHandler posts an OOC message.
func InteractionPostOOCHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionPostOOCInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionPostOOCInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		var header metadata.MD
		response, err := client.PostSessionOOC(callCtx.RunCtx, &statev1.PostSessionOOCRequest{
			CampaignId: campaignID,
			Body:       strings.TrimSpace(input.Body),
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("post session ooc failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("post session ooc response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionMarkOOCReadyHandler marks the caller ready to resume.
func InteractionMarkOOCReadyHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionPauseOOCInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionPauseOOCInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		var header metadata.MD
		response, err := client.MarkOOCReadyToResume(callCtx.RunCtx, &statev1.MarkOOCReadyToResumeRequest{
			CampaignId: campaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("mark ooc ready failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("mark ooc ready response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionClearOOCReadyHandler clears the caller's ready state.
func InteractionClearOOCReadyHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionPauseOOCInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionPauseOOCInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		var header metadata.MD
		response, err := client.ClearOOCReadyToResume(callCtx.RunCtx, &statev1.ClearOOCReadyToResumeRequest{
			CampaignId: campaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("clear ooc ready failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("clear ooc ready response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionResumeOOCHandler resumes in-character play.
func InteractionResumeOOCHandler(client statev1.InteractionServiceClient, getContext func() Context, notify ResourceUpdateNotifier) mcp.ToolHandlerFor[InteractionPauseOOCInput, InteractionStateResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input InteractionPauseOOCInput) (*mcp.CallToolResult, InteractionStateResult, error) {
		campaignID, callCtx, callMeta, err := interactionCallContext(ctx, getContext, input.CampaignID)
		if err != nil {
			return nil, InteractionStateResult{}, err
		}
		defer callCtx.Cancel()

		var header metadata.MD
		response, err := client.ResumeFromOOC(callCtx.RunCtx, &statev1.ResumeFromOOCRequest{
			CampaignId: campaignID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, InteractionStateResult{}, fmt.Errorf("resume from ooc failed: %w", err)
		}
		if response == nil {
			return nil, InteractionStateResult{}, fmt.Errorf("resume from ooc response is missing")
		}
		return interactionToolResult(ctx, notify, campaignID, response.GetState(), callMeta, header)
	}
}

// InteractionStateResourceHandler returns a readable interaction snapshot.
func InteractionStateResourceHandler(client statev1.InteractionServiceClient, getContext func() Context) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("interaction service client is not configured")
		}
		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format %s", interactionStateResourceURITemplate)
		}

		campaignID, err := parseCampaignIDFromResourceURI(req.Params.URI, "interaction")
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		callContext, err := newToolInvocationContext(ctx, getContext)
		if err != nil {
			return nil, fmt.Errorf("generate invocation id: %w", err)
		}
		defer callContext.Cancel()

		callCtx, _, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		response, err := client.GetInteractionState(callCtx, &statev1.GetInteractionStateRequest{CampaignId: campaignID})
		if err != nil {
			return nil, fmt.Errorf("get interaction state failed: %w", err)
		}
		if response == nil || response.State == nil {
			return nil, fmt.Errorf("get interaction state response is missing")
		}

		payloadJSON, err := json.MarshalIndent(interactionStateResultFromProto(response.State), "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal interaction state: %w", err)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(payloadJSON),
				},
			},
		}, nil
	}
}

func interactionToolResult(ctx context.Context, notify ResourceUpdateNotifier, campaignID string, state *statev1.InteractionState, callMeta ToolCallMetadata, header metadata.MD) (*mcp.CallToolResult, InteractionStateResult, error) {
	if state == nil {
		return nil, InteractionStateResult{}, fmt.Errorf("interaction state response is missing")
	}
	NotifyResourceUpdates(ctx, notify, fmt.Sprintf("campaign://%s/interaction", campaignID))
	return CallToolResultWithMetadata(MergeResponseMetadata(callMeta, header)), interactionStateResultFromProto(state), nil
}

type interactionToolContext struct {
	RunCtx       context.Context
	Cancel       context.CancelFunc
	InvocationID string
	MCPContext   Context
}

func interactionCallContext(ctx context.Context, getContext func() Context, explicitCampaignID string) (string, interactionToolContext, ToolCallMetadata, error) {
	callContext, err := newToolInvocationContext(ctx, getContext)
	if err != nil {
		return "", interactionToolContext{}, ToolCallMetadata{}, fmt.Errorf("generate invocation id: %w", err)
	}
	campaignID := strings.TrimSpace(explicitCampaignID)
	if campaignID == "" {
		campaignID = strings.TrimSpace(callContext.MCPContext.CampaignID)
	}
	if campaignID == "" {
		callContext.Cancel()
		return "", interactionToolContext{}, ToolCallMetadata{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, callMeta, err := NewOutgoingContextWithContext(callContext.RunCtx, callContext.InvocationID, callContext.MCPContext)
	if err != nil {
		callContext.Cancel()
		return "", interactionToolContext{}, ToolCallMetadata{}, fmt.Errorf("create request metadata: %w", err)
	}
	return campaignID, interactionToolContext{
		RunCtx:       callCtx,
		Cancel:       callContext.Cancel,
		InvocationID: callContext.InvocationID,
		MCPContext:   callContext.MCPContext,
	}, callMeta, nil
}

func resolveInteractionSceneID(ctx context.Context, client statev1.InteractionServiceClient, campaignID, explicitSceneID string) (string, error) {
	sceneID := strings.TrimSpace(explicitSceneID)
	if sceneID != "" {
		return sceneID, nil
	}
	response, err := client.GetInteractionState(ctx, &statev1.GetInteractionStateRequest{CampaignId: campaignID})
	if err != nil {
		return "", fmt.Errorf("get interaction state failed: %w", err)
	}
	if response == nil || response.State == nil {
		return "", fmt.Errorf("get interaction state response is missing")
	}
	sceneID = strings.TrimSpace(response.GetState().GetActiveScene().GetSceneId())
	if sceneID == "" {
		return "", fmt.Errorf("scene_id is required when no active scene is set")
	}
	return sceneID, nil
}

func interactionStateResultFromProto(state *statev1.InteractionState) InteractionStateResult {
	if state == nil {
		return InteractionStateResult{}
	}
	result := InteractionStateResult{
		CampaignID:   state.GetCampaignId(),
		CampaignName: state.GetCampaignName(),
		Locale:       state.GetLocale().String(),
		Viewer: InteractionViewerResult{
			ParticipantID: state.GetViewer().GetParticipantId(),
			Name:          state.GetViewer().GetName(),
			Role:          participantRoleToString(state.GetViewer().GetRole()),
		},
		ActiveSession: InteractionSessionResult{
			SessionID: state.GetActiveSession().GetSessionId(),
			Name:      state.GetActiveSession().GetName(),
		},
		ActiveScene: InteractionSceneResult{
			SceneID:     state.GetActiveScene().GetSceneId(),
			SessionID:   state.GetActiveScene().GetSessionId(),
			Name:        state.GetActiveScene().GetName(),
			Description: state.GetActiveScene().GetDescription(),
			Characters:  make([]InteractionCharacterResult, 0, len(state.GetActiveScene().GetCharacters())),
		},
		PlayerPhase: InteractionPlayerPhaseResult{
			PhaseID:              state.GetPlayerPhase().GetPhaseId(),
			Status:               scenePhaseStatusToString(state.GetPlayerPhase().GetStatus()),
			FrameText:            state.GetPlayerPhase().GetFrameText(),
			ActingCharacterIDs:   append([]string(nil), state.GetPlayerPhase().GetActingCharacterIds()...),
			ActingParticipantIDs: append([]string(nil), state.GetPlayerPhase().GetActingParticipantIds()...),
			Slots:                make([]InteractionPlayerSlotResult, 0, len(state.GetPlayerPhase().GetSlots())),
		},
		OOC: InteractionOOCStateResult{
			Open:                        state.GetOoc().GetOpen(),
			Posts:                       make([]InteractionOOCPostResult, 0, len(state.GetOoc().GetPosts())),
			ReadyToResumeParticipantIDs: append([]string(nil), state.GetOoc().GetReadyToResumeParticipantIds()...),
		},
	}

	for _, character := range state.GetActiveScene().GetCharacters() {
		result.ActiveScene.Characters = append(result.ActiveScene.Characters, InteractionCharacterResult{
			CharacterID:        character.GetCharacterId(),
			Name:               character.GetName(),
			OwnerParticipantID: character.GetOwnerParticipantId(),
		})
	}
	for _, slot := range state.GetPlayerPhase().GetSlots() {
		result.PlayerPhase.Slots = append(result.PlayerPhase.Slots, InteractionPlayerSlotResult{
			ParticipantID:      slot.GetParticipantId(),
			SummaryText:        slot.GetSummaryText(),
			CharacterIDs:       append([]string(nil), slot.GetCharacterIds()...),
			UpdatedAt:          formatTimestamp(slot.GetUpdatedAt()),
			Yielded:            slot.GetYielded(),
			ReviewStatus:       scenePlayerSlotReviewStatusToString(slot.GetReviewStatus()),
			ReviewReason:       slot.GetReviewReason(),
			ReviewCharacterIDs: append([]string(nil), slot.GetReviewCharacterIds()...),
		})
	}
	for _, post := range state.GetOoc().GetPosts() {
		result.OOC.Posts = append(result.OOC.Posts, InteractionOOCPostResult{
			PostID:        post.GetPostId(),
			ParticipantID: post.GetParticipantId(),
			Body:          post.GetBody(),
			CreatedAt:     formatTimestamp(post.GetCreatedAt()),
		})
	}
	return result
}

func scenePhaseStatusToString(status statev1.ScenePhaseStatus) string {
	switch status {
	case statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM:
		return "GM"
	case statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS:
		return "PLAYERS"
	case statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW:
		return "GM_REVIEW"
	default:
		return "UNSPECIFIED"
	}
}

func scenePlayerSlotReviewStatusToString(status statev1.ScenePlayerSlotReviewStatus) string {
	switch status {
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_OPEN:
		return "OPEN"
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNDER_REVIEW:
		return "UNDER_REVIEW"
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED:
		return "ACCEPTED"
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED:
		return "CHANGES_REQUESTED"
	default:
		return "UNSPECIFIED"
	}
}
