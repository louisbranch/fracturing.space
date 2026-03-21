package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// NewProjectionStores builds the root projection concern from one projection
// bundle plus the built-in system read models that share that backend.
func NewProjectionStores(config StoresProjectionConfig) ProjectionStores {
	return ProjectionStores{
		Campaign:           config.ProjectionStore,
		Participant:        config.ProjectionStore,
		ClaimIndex:         config.ProjectionStore,
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
		SceneGMInteraction: config.ProjectionStore,
		CampaignFork:       config.ProjectionStore,
	}
}

// NewInfrastructureStores builds the operational store concern from the exact
// infrastructure dependencies that root transport and projection apply use.
func NewInfrastructureStores(
	projectionStore storage.ProjectionStore,
	config StoresInfrastructureConfig,
) InfrastructureStores {
	return InfrastructureStores{
		Event:      config.EventStore,
		Watermarks: projectionStore,
		Audit:      config.AuditStore,
		Statistics: projectionStore,
		Snapshot:   projectionStore,
	}
}

// NewContentStores builds the read-only content and external client concern
// used by root transport handlers.
func NewContentStores(config StoresContentConfig) ContentStores {
	return ContentStores{
		DaggerheartContent: config.ContentStore,
		Social:             config.SocialClient,
	}
}

// NewRuntimeStores builds the write-path runtime concern from the exact
// executor/runtime collaborators owned by startup.
func NewRuntimeStores(
	config StoresRuntimeConfig,
	auditStore storage.AuditEventStore,
) RuntimeStores {
	return RuntimeStores{
		Write: domainwrite.WritePath{
			Executor: config.Domain,
			Runtime:  config.WriteRuntime,
			Audit:    auditStore,
		},
	}
}
