package coreprojection

import "github.com/louisbranch/fracturing.space/internal/services/game/storage"

var (
	// Aggregate projection contract: includes campaign, session, and scene
	// read stores plus infrastructure concerns (snapshots, statistics,
	// watermarks).
	_ storage.ProjectionStore = (*Store)(nil)

	// Purpose-scoped projection composites — all satisfied by the same Store.
	_ storage.CampaignReadStores = (*Store)(nil)
	_ storage.SessionReadStores  = (*Store)(nil)
	_ storage.SceneReadStores    = (*Store)(nil)

	// Projection apply/runtime integrity contracts.
	_ storage.ProjectionApplyExactlyOnceStore = (*Store)(nil)
	_ storage.ProjectionApplyTxStore          = (*Store)(nil)
)
