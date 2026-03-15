package eventjournal

import "github.com/louisbranch/fracturing.space/internal/services/game/storage"

var (
	_ storage.EventStore             = (*Store)(nil)
	_ storage.AuditEventStore        = (*Store)(nil)
	_ storage.EventIntegrityVerifier = (*Store)(nil)
)
