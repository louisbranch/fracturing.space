package sqlite

import "github.com/louisbranch/fracturing.space/internal/services/game/storage"

var (
	// Aggregate store contracts used across app/bootstrap tooling.
	_ storage.Store           = (*Store)(nil)
	_ storage.ProjectionStore = (*Store)(nil)

	// Projection seams intentionally outside ProjectionStore aggregate.
	_ storage.SessionGateStore      = (*Store)(nil)
	_ storage.SessionSpotlightStore = (*Store)(nil)
	_ storage.SceneStore            = (*Store)(nil)
	_ storage.SceneCharacterStore   = (*Store)(nil)
	_ storage.SceneGateStore        = (*Store)(nil)
	_ storage.SceneSpotlightStore   = (*Store)(nil)

	// Projection apply/runtime integrity contracts.
	_ storage.EventIntegrityVerifier               = (*Store)(nil)
	_ storage.ProjectionApplyOutboxProcessor       = (*Store)(nil)
	_ storage.ProjectionApplyOutboxShadowProcessor = (*Store)(nil)
	_ storage.ProjectionApplyExactlyOnceStore      = (*Store)(nil)
	_ storage.ProjectionApplyTxStore               = (*Store)(nil)
)
