package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
		Timestamp:      toMillis(evt.Timestamp),
		EventName:      evt.EventName,
		Severity:       evt.Severity,
		CampaignID:     toNullString(evt.CampaignID),
		SessionID:      toNullString(evt.SessionID),
		ActorType:      toNullString(evt.ActorType),
		ActorID:        toNullString(evt.ActorID),
		RequestID:      toNullString(evt.RequestID),
		InvocationID:   toNullString(evt.InvocationID),
		TraceID:        toNullString(evt.TraceID),
		SpanID:         toNullString(evt.SpanID),
		AttributesJson: evt.AttributesJSON,
	})
}

// GetGameStatistics returns aggregate counts across the game data set.
func (s *Store) GetGameStatistics(ctx context.Context, since *time.Time) (storage.GameStatistics, error) {
	if err := ctx.Err(); err != nil {
		return storage.GameStatistics{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.GameStatistics{}, fmt.Errorf("storage is not configured")
	}

	sinceValue := toNullMillis(since)

	row, err := s.q.GetGameStatistics(ctx, sinceValue)
	if err != nil {
		return storage.GameStatistics{}, fmt.Errorf("get game statistics: %w", err)
	}

	return storage.GameStatistics{
		CampaignCount:    row.CampaignCount,
		SessionCount:     row.SessionCount,
		CharacterCount:   row.CharacterCount,
		ParticipantCount: row.ParticipantCount,
	}, nil
}

func toNullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
