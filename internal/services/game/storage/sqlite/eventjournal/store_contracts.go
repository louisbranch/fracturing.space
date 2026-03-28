package eventjournal

import "github.com/louisbranch/fracturing.space/internal/services/game/storage"

var (
	_ storage.EventAppender          = (*Store)(nil)
	_ storage.BatchEventAppender     = (*Store)(nil)
	_ storage.EventReadStore         = (*Store)(nil)
	_ storage.EventHistoryStore      = (*Store)(nil)
	_ storage.EventLookupStore       = (*Store)(nil)
	_ storage.EventStore             = (*Store)(nil)
	_ storage.AuditEventStore        = (*Store)(nil)
	_ storage.EventIntegrityVerifier = (*Store)(nil)
)
