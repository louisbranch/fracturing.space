package eventjournal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// AppendAuditEvent records an operational audit event.
func (s *Store) AppendAuditEvent(ctx context.Context, evt storage.AuditEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(evt.EventName) == "" {
		return fmt.Errorf("event name is required")
	}
	if strings.TrimSpace(evt.Severity) == "" {
		return fmt.Errorf("severity is required")
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	if len(evt.AttributesJSON) == 0 && len(evt.Attributes) > 0 {
		payload, err := json.Marshal(evt.Attributes)
		if err != nil {
			return fmt.Errorf("marshal audit attributes: %w", err)
		}
		evt.AttributesJSON = payload
	}

	return s.q.AppendAuditEvent(ctx, db.AppendAuditEventParams{
		Timestamp:      sqliteutil.ToMillis(evt.Timestamp),
		EventName:      evt.EventName,
		Severity:       evt.Severity,
		CampaignID:     sqliteutil.ToNullString(evt.CampaignID),
		SessionID:      sqliteutil.ToNullString(evt.SessionID),
		ActorType:      sqliteutil.ToNullString(evt.ActorType),
		ActorID:        sqliteutil.ToNullString(evt.ActorID),
		RequestID:      sqliteutil.ToNullString(evt.RequestID),
		InvocationID:   sqliteutil.ToNullString(evt.InvocationID),
		TraceID:        sqliteutil.ToNullString(evt.TraceID),
		SpanID:         sqliteutil.ToNullString(evt.SpanID),
		AttributesJson: evt.AttributesJSON,
	})
}
