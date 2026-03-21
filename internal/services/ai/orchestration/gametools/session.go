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
	clients  Clients
	registry productionToolRegistry
	sc       SessionContext
}

// NewDirectSession creates a session bound to fixed campaign authority using the
// default production tool registry. Prefer NewDirectDialer for production use;
// this constructor is useful in tests that need a standalone session.
func NewDirectSession(clients Clients, sc SessionContext) *DirectSession {
	return newDirectSession(clients, defaultRegistry, sc)
}

// newDirectSession creates a session with an explicit registry.
func newDirectSession(clients Clients, reg productionToolRegistry, sc SessionContext) *DirectSession {
	return &DirectSession{clients: clients, registry: reg, sc: sc}
}

// ListTools returns the production tool definitions owned by the registry.
func (s *DirectSession) ListTools(_ context.Context) ([]orchestration.Tool, error) {
	return s.registry.tools(), nil
}

// CallTool dispatches a tool call by name to the correct gRPC handler.
func (s *DirectSession) CallTool(ctx context.Context, name string, args any) (orchestration.ToolResult, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("marshal tool arguments: %w", err)
	}

	definition, ok := s.registry.lookup(name)
	if !ok {
		return orchestration.ToolResult{
			Output:  fmt.Sprintf("unknown tool %q", name),
			IsError: true,
		}, nil
	}

	result, err := definition.Execute(s, ctx, argsJSON)
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

// toolResultJSON marshals the result value as a JSON tool result.
func toolResultJSON(v any) (orchestration.ToolResult, error) {
	data, _ := json.Marshal(v)
	return orchestration.ToolResult{Output: string(data)}, nil
}
