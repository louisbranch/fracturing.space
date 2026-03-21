package gametools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

// Clients bundles gRPC service clients needed by the direct session.
type Clients struct {
	Interaction statev1.InteractionServiceClient
	Scene       statev1.SceneServiceClient
	Campaign    statev1.CampaignServiceClient
	Participant statev1.ParticipantServiceClient
	Character   statev1.CharacterServiceClient
	Session     statev1.SessionServiceClient
	Snapshot    statev1.SnapshotServiceClient
	Daggerheart pb.DaggerheartServiceClient
	Artifact    aiv1.CampaignArtifactServiceClient
	Reference   aiv1.SystemReferenceServiceClient
}

// DirectSession implements orchestration.Session by calling game gRPC
// services directly.
type DirectSession struct {
	clients Clients
	sc      sessionContext
}

// NewDirectSession creates a session bound to fixed campaign authority.
func NewDirectSession(clients Clients, sc sessionContext) *DirectSession {
	return &DirectSession{clients: clients, sc: sc}
}

// ListTools returns the 30 production tool definitions as static data.
func (s *DirectSession) ListTools(_ context.Context) ([]orchestration.Tool, error) {
	return productionTools(), nil
}

// CallTool dispatches a tool call by name to the correct gRPC handler.
func (s *DirectSession) CallTool(ctx context.Context, name string, args any) (orchestration.ToolResult, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("marshal tool arguments: %w", err)
	}

	handler, ok := s.toolHandler(strings.TrimSpace(name))
	if !ok {
		return orchestration.ToolResult{
			Output:  fmt.Sprintf("unknown tool %q", name),
			IsError: true,
		}, nil
	}

	result, err := handler(ctx, argsJSON)
	if err != nil {
		return orchestration.ToolResult{
			Output:  fmt.Sprintf("tool call failed: %v", err),
			IsError: true,
		}, nil
	}
	return result, nil
}

// ReadResource dispatches a resource URI to the correct gRPC reader.
func (s *DirectSession) ReadResource(ctx context.Context, uri string) (string, error) {
	return s.readResource(ctx, strings.TrimSpace(uri))
}

// Close is a no-op: gRPC connections are shared, not owned by the session.
func (s *DirectSession) Close() error { return nil }

// toolFunc is a typed tool handler that takes JSON args and returns a tool result.
type toolFunc func(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error)

func (s *DirectSession) toolHandler(name string) (toolFunc, bool) {
	handlers := map[string]toolFunc{
		// Interaction tools (10)
		"interaction_active_scene_set":               s.interactionSetActiveScene,
		"interaction_scene_player_phase_start":       s.interactionStartScenePlayerPhase,
		"interaction_scene_player_phase_accept":      s.interactionAcceptScenePlayerPhase,
		"interaction_scene_player_revisions_request": s.interactionRequestScenePlayerRevisions,
		"interaction_scene_player_phase_end":         s.interactionEndScenePlayerPhase,
		"interaction_scene_gm_output_commit":         s.interactionCommitSceneGMOutput,
		"interaction_ooc_pause":                      s.interactionPauseOOC,
		"interaction_ooc_post":                       s.interactionPostOOC,
		"interaction_ooc_ready_mark":                 s.interactionMarkOOCReady,
		"interaction_ooc_ready_clear":                s.interactionClearOOCReady,
		"interaction_ooc_resume":                     s.interactionResumeOOC,
		// Scene tools (6)
		"scene_create":           s.sceneCreate,
		"scene_update":           s.sceneUpdate,
		"scene_end":              s.sceneEnd,
		"scene_transition":       s.sceneTransition,
		"scene_add_character":    s.sceneAddCharacter,
		"scene_remove_character": s.sceneRemoveCharacter,
		// Daggerheart tools (6)
		"duality_action_roll":   s.dualityActionRoll,
		"roll_dice":             s.rollDice,
		"duality_outcome":       s.dualityOutcome,
		"duality_explain":       s.dualityExplain,
		"duality_probability":   s.dualityProbability,
		"duality_rules_version": s.dualityRulesVersion,
		// Artifact tools (3)
		"campaign_artifact_list":   s.artifactList,
		"campaign_artifact_get":    s.artifactGet,
		"campaign_artifact_upsert": s.artifactUpsert,
		// Memory section tools (2)
		"campaign_memory_section_read":   s.memorySectionRead,
		"campaign_memory_section_update": s.memorySectionUpdate,
		// Reference tools (2)
		"system_reference_search": s.referenceSearch,
		"system_reference_read":   s.referenceRead,
	}
	fn, ok := handlers[name]
	return fn, ok
}

// toolResultJSON marshals the result value as a JSON tool result.
func toolResultJSON(v any) (orchestration.ToolResult, error) {
	data, _ := json.Marshal(v)
	return orchestration.ToolResult{Output: string(data)}, nil
}
