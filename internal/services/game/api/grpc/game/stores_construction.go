package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// NewStoresFromProjection constructs Stores from a projection-oriented store
// bundle plus runtime dependencies. This reduces startup constructor coupling
// to individual store interfaces while preserving explicit overrides.
func NewStoresFromProjection(config StoresFromProjectionConfig) Stores {
	return Stores{
		Campaign:           config.ProjectionStore,
		Participant:        config.ProjectionStore,
		ClaimIndex:         config.ProjectionStore,
		Invite:             config.ProjectionStore,
		Character:          config.ProjectionStore,
		Session:            config.ProjectionStore,
		SessionGate:        config.ProjectionStore,
		SessionSpotlight:   config.ProjectionStore,
		SessionInteraction: config.ProjectionStore,
		Scene:              config.ProjectionStore,
		SceneCharacter:     config.ProjectionStore,
		SceneGate:          config.ProjectionStore,
		SceneSpotlight:     config.ProjectionStore,
		SceneInteraction:   config.ProjectionStore,
		CampaignFork:       config.ProjectionStore,
		SystemStores:       config.SystemStores,
		Event:              config.EventStore,
		Watermarks:         config.ProjectionStore,
		Audit:              inferAuditStore(config),
		Statistics:         config.ProjectionStore,
		Snapshot:           config.ProjectionStore,
		DaggerheartContent: config.ContentStore,
		Social:             config.SocialClient,
		Write: domainwriteexec.WritePath{
			Executor: config.Domain,
			Runtime:  config.WriteRuntime,
			Audit:    inferAuditStore(config),
		},
		Events: config.Events,
	}
}

func inferAuditStore(config StoresFromProjectionConfig) storage.AuditEventStore {
	if config.AuditStore != nil {
		return config.AuditStore
	}
	if config.EventStore == nil {
		return nil
	}
	inferredAuditStore, ok := config.EventStore.(storage.AuditEventStore)
	if !ok {
		return nil
	}
	return inferredAuditStore
}
