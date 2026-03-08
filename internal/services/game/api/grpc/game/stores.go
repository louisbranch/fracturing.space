package game

import (
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Stores groups all campaign-related storage interfaces for service injection.
type Stores struct {
	// Core projection stores — used by the projection applier for core events.
	Campaign         storage.CampaignStore
	Participant      storage.ParticipantStore
	ClaimIndex       storage.ClaimIndexStore
	Invite           storage.InviteStore
	Character        storage.CharacterStore
	Session          storage.SessionStore
	SessionGate      storage.SessionGateStore
	SessionSpotlight storage.SessionSpotlightStore
	Scene            storage.SceneStore
	SceneCharacter   storage.SceneCharacterStore
	SceneGate        storage.SceneGateStore
	SceneSpotlight   storage.SceneSpotlightStore
	CampaignFork     storage.CampaignForkStore

	// SystemStores groups system-specific projection stores used by
	// AdapterRegistry. When adding a new game system, add its store here
	// and in manifest.ProjectionStores — no other Stores fields need changing.
	SystemStores systemmanifest.ProjectionStores

	// Infrastructure stores — event journal, snapshots, audit.
	Event      storage.EventStore
	Watermarks storage.ProjectionWatermarkStore
	Audit      storage.AuditEventStore
	Statistics storage.StatisticsStore
	Snapshot   storage.SnapshotStore

	// System content stores — read-only content used by gRPC handlers.
	DaggerheartContent storage.DaggerheartContentReadStore
	Social             socialv1.SocialServiceClient

	// Write groups the domain executor, runtime controls, and audit store
	// used by the write path. It satisfies domainwriteexec.Deps so handlers
	// can pass it directly to executeAndApplyDomainCommand.
	Write domainwriteexec.WritePath

	// Events is the event registry used for intent filtering and applier
	// construction at request time.
	Events *event.Registry

	// adapters is built eagerly during Validate and cached for Applier.
	adapters adapterRegistry
}

// ProjectionStoreBundle is the projection dependency contract for game gRPC
// handlers and appliers. Startup wires one projection implementation into this
// bundle so callers avoid assigning each projection interface manually.
type ProjectionStoreBundle interface {
	storage.ProjectionStore
	storage.SessionGateStore
	storage.SessionSpotlightStore
	storage.SceneStore
	storage.SceneCharacterStore
	storage.SceneGateStore
	storage.SceneSpotlightStore
}

// StoresFromProjectionConfig configures NewStoresFromProjection.
type StoresFromProjectionConfig struct {
	ProjectionStore ProjectionStoreBundle
	SystemStores    systemmanifest.ProjectionStores
	EventStore      storage.EventStore
	AuditStore      storage.AuditEventStore
	ContentStore    storage.DaggerheartContentReadStore
	SocialClient    socialv1.SocialServiceClient
	Domain          Domain
	WriteRuntime    *domainwrite.Runtime
	Events          *event.Registry
}

// NewWriteRuntime creates a new write-path runtime for use by service startup.
// This factory avoids leaking the internal domainwrite package to callers
// outside the gRPC package tree.
func NewWriteRuntime() *domainwrite.Runtime {
	return domainwrite.NewRuntime()
}
