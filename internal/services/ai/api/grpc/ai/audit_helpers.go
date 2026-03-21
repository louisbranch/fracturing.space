package ai

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// putAuditEvent centralizes audit persistence so capability handlers can share
// one write path without keeping audit ownership on the broad transport root.
func putAuditEvent(ctx context.Context, store storage.AuditEventStore, record storage.AuditEventRecord) error {
	if store == nil {
		return fmt.Errorf("audit event store is not configured")
	}
	record.CreatedAt = record.CreatedAt.UTC()
	return store.PutAuditEvent(ctx, record)
}
