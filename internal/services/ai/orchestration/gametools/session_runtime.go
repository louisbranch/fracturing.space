package gametools

import (
	"context"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaignartifact"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/referencecorpus"
)

// ArtifactManager provides campaign artifact operations for tool execution.
// Satisfied by *campaigncontext.Manager.
type ArtifactManager interface {
	ListArtifacts(ctx context.Context, campaignID string) ([]campaignartifact.Artifact, error)
	GetArtifact(ctx context.Context, campaignID string, path string) (campaignartifact.Artifact, error)
	UpsertArtifact(ctx context.Context, campaignID string, path string, content string) (campaignartifact.Artifact, error)
}

// ReferenceCorpus provides system reference search and read for tool execution.
// Satisfied by *referencecorpus.Corpus.
type ReferenceCorpus interface {
	Search(ctx context.Context, system, query string, maxResults int) ([]referencecorpus.SearchResult, error)
	Read(ctx context.Context, system, documentID string) (referencecorpus.Document, error)
}

// Clients bundles service clients needed by the direct session.
type Clients struct {
	Interaction statev1.InteractionServiceClient
	CampaignAI  statev1.CampaignAIOrchestrationServiceClient
	Scene       statev1.SceneServiceClient
	Campaign    statev1.CampaignServiceClient
	Participant statev1.ParticipantServiceClient
	Character   statev1.CharacterServiceClient
	Session     statev1.SessionServiceClient
	Snapshot    statev1.SnapshotServiceClient
	Daggerheart pb.DaggerheartServiceClient
	Artifact    ArtifactManager
	Reference   ReferenceCorpus
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

// ReadResource dispatches a resource URI to the correct gRPC reader.
func (s *DirectSession) ReadResource(ctx context.Context, uri string) (string, error) {
	return s.readResource(ctx, uri)
}

// Close is a no-op: gRPC connections are shared, not owned by the session.
func (s *DirectSession) Close() error { return nil }
