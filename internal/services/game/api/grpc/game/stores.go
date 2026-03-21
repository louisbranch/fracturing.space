package game

import (
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// SystemStores groups system-specific projection stores consumed by the core
// game transport. This package keeps its explicit Daggerheart dependency
// because snapshot and profile reads are part of the product surface, not the
// manifest extension platform.
type SystemStores = systemmanifest.ProjectionStores

// WritePath exposes the root transport write-path contract without forcing
// startup callers to import the grpc/internal package directly.
type WritePath = domainwrite.WritePath

// ProjectionStores groups the core projection-backed read models used by the
// root game transport and projection applier.
type ProjectionStores struct {
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	ClaimIndex         storage.ClaimIndexStore
	Character          storage.CharacterStore
	Session            storage.SessionStore
	SessionGate        storage.SessionGateStore
	SessionSpotlight   storage.SessionSpotlightStore
	SessionInteraction storage.SessionInteractionStore
	Scene              storage.SceneStore
	SceneCharacter     storage.SceneCharacterStore
	SceneGate          storage.SceneGateStore
	SceneSpotlight     storage.SceneSpotlightStore
	SceneInteraction   storage.SceneInteractionStore
	SceneGMInteraction storage.SceneGMInteractionStore
	CampaignFork       storage.CampaignForkStore
}

// InfrastructureStores groups non-projection stores used by root transport
// infrastructure and projection application.
type InfrastructureStores struct {
	Event      storage.EventStore
	Watermarks storage.ProjectionWatermarkStore
	Audit      storage.AuditEventStore
	Statistics storage.StatisticsStore
	Snapshot   storage.SnapshotStore
}

// ContentStores groups read-only content and external service clients consumed
// by the root game transport.
type ContentStores struct {
	DaggerheartContent contentstore.DaggerheartContentReadStore
	Social             socialv1.SocialServiceClient
}

// RuntimeStores groups runtime-owned collaborators used by the write path.
type RuntimeStores struct {
	Write domainwrite.WritePath
}

// StoresProjectionConfig groups the projection-backed contracts used to build
// the root game transport projection concern. The ProjectionStore satisfies
// all purpose-scoped read store interfaces (CampaignReadStores,
// SessionReadStores, SceneReadStores) plus infrastructure concerns.
type StoresProjectionConfig struct {
	ProjectionStore storage.ProjectionStore
	SystemStores    SystemStores
}

// StoresInfrastructureConfig groups infrastructure stores that are not part of
// the projection bundle and must be wired explicitly by startup.
type StoresInfrastructureConfig struct {
	EventStore storage.EventStore
	AuditStore storage.AuditEventStore
}

// StoresContentConfig groups read-only external content and service clients
// consumed by the root game transport.
type StoresContentConfig struct {
	ContentStore contentstore.DaggerheartContentReadStore
	SocialClient socialv1.SocialServiceClient
}

// StoresRuntimeConfig groups write-path collaborators consumed by the root game
// transport.
type StoresRuntimeConfig struct {
	Domain       handler.Domain
	WriteRuntime *domainwrite.Runtime
}

// NewWriteRuntime creates a new write-path runtime for use by service startup.
// This factory avoids leaking the internal domainwrite package to callers
// outside the gRPC package tree.
func NewWriteRuntime() *domainwrite.Runtime {
	return domainwrite.NewRuntime()
}
