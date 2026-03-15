package coreprojection

import "github.com/louisbranch/fracturing.space/internal/services/game/storage"

var (
	// Aggregate projection contracts used across app/bootstrap tooling.
	_ storage.ProjectionStore = (*Store)(nil)

	// Projection seams intentionally outside ProjectionStore aggregate.
	_ storage.SessionGateStore        = (*Store)(nil)
	_ storage.SessionSpotlightStore   = (*Store)(nil)
	_ storage.SessionInteractionStore = (*Store)(nil)
	_ storage.SceneStore              = (*Store)(nil)
	_ storage.SceneCharacterStore     = (*Store)(nil)
	_ storage.SceneGateStore          = (*Store)(nil)
	_ storage.SceneSpotlightStore     = (*Store)(nil)
	_ storage.SceneInteractionStore   = (*Store)(nil)

	// Projection apply/runtime integrity contracts.
	_ storage.ProjectionApplyExactlyOnceStore = (*Store)(nil)
	_ storage.ProjectionApplyTxStore          = (*Store)(nil)
)
