package maintenance

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// closableEventStore extends EventStore with a Close method for resource cleanup.
type closableEventStore interface {
	storage.EventStore
	Close() error
}

// closableProjectionStore extends ProjectionStore with a Close method for resource cleanup.
type closableProjectionStore interface {
	storage.ProjectionStore
	Close() error
}
