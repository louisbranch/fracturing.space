package projection

import "github.com/louisbranch/fracturing.space/internal/services/game/storage"

// TxStoreBundle is the transaction-scoped projection contract needed by
// exactly-once projection apply callbacks.
type TxStoreBundle = storage.ProjectionApplyTxStore

// ExactlyOnceStore applies one event to projections exactly once per
// campaign/sequence checkpoint.
type ExactlyOnceStore = storage.ProjectionApplyExactlyOnceStore
