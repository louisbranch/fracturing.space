package projection

import (
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStores groups campaign lifecycle and fork projection dependencies.
type CampaignStores struct {
	Campaign     storage.CampaignStore
	CampaignFork storage.CampaignForkStore
}

// ParticipantStores groups participant and claim-index projection dependencies.
type ParticipantStores struct {
	Participant storage.ParticipantStore
	ClaimIndex  storage.ClaimIndexStore
}

// CharacterStores groups character projection dependencies.
type CharacterStores struct {
	Character storage.CharacterStore
}

// InviteStores groups invite projection dependencies.
type InviteStores struct {
	Invite storage.InviteStore
}

// SessionStores groups session and session-interaction projection dependencies.
type SessionStores struct {
	Session            storage.SessionStore
	SessionGate        storage.SessionGateStore
	SessionSpotlight   storage.SessionSpotlightStore
	SessionInteraction storage.SessionInteractionStore
}

// SceneStores groups scene and scene-interaction projection dependencies.
type SceneStores struct {
	Scene            storage.SceneStore
	SceneCharacter   storage.SceneCharacterStore
	SceneGate        storage.SceneGateStore
	SceneSpotlight   storage.SceneSpotlightStore
	SceneInteraction storage.SceneInteractionStore
}

// SupportStores groups projection support concerns such as watermarks.
type SupportStores struct {
	Watermarks storage.ProjectionWatermarkStore
}

// StoreGroups is the projection-owned dependency surface for core projection
// handlers. It keeps construction grouped by concern instead of as one large
// flat bag.
type StoreGroups struct {
	CampaignStores
	ParticipantStores
	CharacterStores
	InviteStores
	SessionStores
	SceneStores
	SupportStores
}

// StoreBundle is the projection-owned store contract for the full core read
// model surface plus watermark persistence. It composes the purpose-scoped
// interfaces from the storage package so consumers import only what they need
// while the SQLite implementation satisfies the full surface.
type StoreBundle interface {
	storage.CampaignReadStores
	storage.SessionReadStores
	storage.SceneReadStores
	storage.ProjectionWatermarkStore
}

// StoreGroupsFromBundle expands a full projection bundle into concern-local
// store groups for applier construction.
func StoreGroupsFromBundle(bundle StoreBundle) StoreGroups {
	return StoreGroups{
		CampaignStores: CampaignStores{
			Campaign:     bundle,
			CampaignFork: bundle,
		},
		ParticipantStores: ParticipantStores{
			Participant: bundle,
			ClaimIndex:  bundle,
		},
		CharacterStores: CharacterStores{
			Character: bundle,
		},
		InviteStores: InviteStores{
			Invite: bundle,
		},
		SessionStores: SessionStores{
			Session:            bundle,
			SessionGate:        bundle,
			SessionSpotlight:   bundle,
			SessionInteraction: bundle,
		},
		SceneStores: SceneStores{
			Scene:            bundle,
			SceneCharacter:   bundle,
			SceneGate:        bundle,
			SceneSpotlight:   bundle,
			SceneInteraction: bundle,
		},
		SupportStores: SupportStores{
			Watermarks: bundle,
		},
	}
}

// ApplierConfig defines the exact projection-owned collaborators needed to
// build a fully bound applier, including system adapter extraction and audit
// policy.
type ApplierConfig struct {
	Stores       StoreGroups
	SystemStores systemmanifest.ProjectionStores
	Events       *event.Registry
	AuditPolicy  audit.Policy
	Now          func() time.Time
}

// BundleApplierConfig builds an applier from a full projection store bundle.
type BundleApplierConfig struct {
	StoreBundle  StoreBundle
	SystemStores systemmanifest.ProjectionStores
	Events       *event.Registry
	AuditPolicy  audit.Policy
	Now          func() time.Time
}

// BoundApplierConfig defines the exact collaborators needed once system
// adapters are already available.
type BoundApplierConfig struct {
	Stores   StoreGroups
	Events   *event.Registry
	Adapters *bridge.AdapterRegistry
	Auditor  *audit.Emitter
	Now      func() time.Time
}

// NewApplier builds a fully bound projection applier from grouped projection
// stores and system seams, failing fast if the system adapter registry cannot
// be constructed.
func NewApplier(config ApplierConfig) (Applier, error) {
	adapters, err := systemmanifest.AdapterRegistry(config.SystemStores)
	if err != nil {
		return Applier{}, fmt.Errorf("build adapter registry: %w", err)
	}
	return NewBoundApplier(BoundApplierConfig{
		Stores:   config.Stores,
		Events:   config.Events,
		Adapters: adapters,
		Auditor:  audit.NewEmitter(config.AuditPolicy),
		Now:      config.Now,
	}), nil
}

// NewApplierFromBundle expands a full projection store bundle into concern
// groups and then constructs a fully bound applier.
func NewApplierFromBundle(config BundleApplierConfig) (Applier, error) {
	return NewApplier(ApplierConfig{
		Stores:       StoreGroupsFromBundle(config.StoreBundle),
		SystemStores: config.SystemStores,
		Events:       config.Events,
		AuditPolicy:  config.AuditPolicy,
		Now:          config.Now,
	})
}

// NewBoundApplier builds an applier once system adapters are already resolved.
func NewBoundApplier(config BoundApplierConfig) Applier {
	return Applier{
		Events:             config.Events,
		Campaign:           config.Stores.Campaign,
		Character:          config.Stores.Character,
		CampaignFork:       config.Stores.CampaignFork,
		ClaimIndex:         config.Stores.ClaimIndex,
		Invite:             config.Stores.Invite,
		Participant:        config.Stores.Participant,
		Session:            config.Stores.Session,
		SessionGate:        config.Stores.SessionGate,
		SessionSpotlight:   config.Stores.SessionSpotlight,
		SessionInteraction: config.Stores.SessionInteraction,
		Scene:              config.Stores.Scene,
		SceneCharacter:     config.Stores.SceneCharacter,
		SceneGate:          config.Stores.SceneGate,
		SceneSpotlight:     config.Stores.SceneSpotlight,
		SceneInteraction:   config.Stores.SceneInteraction,
		Adapters:           config.Adapters,
		Watermarks:         config.Stores.Watermarks,
		Now:                config.Now,
		Auditor:            config.Auditor,
	}
}
