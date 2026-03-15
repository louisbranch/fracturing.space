package eventjournal

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sqliteprojectionapplyoutbox "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/projectionapplyoutbox"
)

// ProjectionApplyOutboxStore binds the projection-apply outbox backend to this
// event store's SQLite database and event loader.
func (s *Store) ProjectionApplyOutboxStore() storage.ProjectionApplyOutboxStore {
	if s == nil {
		return nil
	}
	return sqliteprojectionapplyoutbox.Bind(s.sqlDB, s, s.eventRegistry)
}
