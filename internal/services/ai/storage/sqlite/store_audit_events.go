package sqlite

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/ai/auditevent"
)

func (s *Store) PutAuditEvent(ctx context.Context, record auditevent.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if record.EventName == "" {
		return fmt.Errorf("event name is required")
	}
	if record.ActorUserID == "" {
		return fmt.Errorf("actor user id is required")
	}
	if record.OwnerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if record.RequesterUserID == "" {
		return fmt.Errorf("requester user id is required")
	}
	if record.AgentID == "" {
		return fmt.Errorf("agent id is required")
	}
	if record.AccessRequestID == "" {
		return fmt.Errorf("access request id is required")
	}
	if record.Outcome == "" {
		return fmt.Errorf("outcome is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("created at is required")
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_audit_events (
	event_name, actor_user_id, owner_user_id, requester_user_id, agent_id, access_request_id, outcome, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
		record.EventName,
		record.ActorUserID,
		record.OwnerUserID,
		record.RequesterUserID,
		record.AgentID,
		record.AccessRequestID,
		record.Outcome,
		sqliteutil.ToMillis(record.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("put audit event: %w", err)
	}
	return nil
}

// ListAuditEventsByOwner returns a page of audit events scoped to one owner.
func (s *Store) ListAuditEventsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter auditevent.Filter) (auditevent.Page, error) {
	if err := ctx.Err(); err != nil {
		return auditevent.Page{}, err
	}
	db, err := requireStoreDB(s)
	if err != nil {
		return auditevent.Page{}, err
	}
	if ownerUserID == "" {
		return auditevent.Page{}, fmt.Errorf("owner user id is required")
	}
	limit, err := keysetPageLimit(pageSize)
	if err != nil {
		return auditevent.Page{}, err
	}
	eventName := filter.EventName
	agentID := filter.AgentID

	var (
		createdAfterMillis  *int64
		createdBeforeMillis *int64
	)
	if filter.CreatedAfter != nil {
		value := sqliteutil.ToMillis(filter.CreatedAfter.UTC())
		createdAfterMillis = &value
	}
	if filter.CreatedBefore != nil {
		value := sqliteutil.ToMillis(filter.CreatedBefore.UTC())
		createdBeforeMillis = &value
	}
	if createdAfterMillis != nil && createdBeforeMillis != nil && *createdAfterMillis > *createdBeforeMillis {
		return auditevent.Page{}, fmt.Errorf("created_after must be before or equal to created_before")
	}

	whereParts := []string{"owner_user_id = ?"}
	args := []any{ownerUserID}
	if eventName != "" {
		whereParts = append(whereParts, "event_name = ?")
		args = append(args, eventName)
	}
	if agentID != "" {
		whereParts = append(whereParts, "agent_id = ?")
		args = append(args, agentID)
	}
	if createdAfterMillis != nil {
		whereParts = append(whereParts, "created_at >= ?")
		args = append(args, *createdAfterMillis)
	}
	if createdBeforeMillis != nil {
		whereParts = append(whereParts, "created_at <= ?")
		args = append(args, *createdBeforeMillis)
	}
	if pageToken != "" {
		tokenValue, parseErr := strconv.ParseInt(pageToken, 10, 64)
		if parseErr != nil || tokenValue < 0 {
			return auditevent.Page{}, fmt.Errorf("invalid page token")
		}
		whereParts = append(whereParts, "id > ?")
		args = append(args, tokenValue)
	}
	args = append(args, limit)

	// Owner scope is always included in WHERE before optional filters so callers
	// can only narrow their own visibility, never expand to another tenant.
	query := fmt.Sprintf(`
SELECT id, event_name, actor_user_id, owner_user_id, requester_user_id, agent_id, access_request_id, outcome, created_at
FROM ai_audit_events
WHERE %s
ORDER BY id
LIMIT ?
`, strings.Join(whereParts, " AND "))
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return auditevent.Page{}, fmt.Errorf("list audit events by owner: %w", err)
	}
	defer rows.Close()

	auditEvents, nextPageToken, err := scanIDKeysetPage(rows, pageSize, scanAuditEventRecord, "audit event", func(event auditevent.Event) string {
		return event.ID
	})
	if err != nil {
		return auditevent.Page{}, err
	}
	return auditevent.Page{AuditEvents: auditEvents, NextPageToken: nextPageToken}, nil
}

func scanAuditEventRecord(s scanner) (auditevent.Event, error) {
	var (
		idValue      int64
		eventName    string
		actorUserID  string
		ownerUser    string
		requesterID  string
		agentID      string
		requestID    string
		outcome      string
		createdAtRaw int64
	)
	if err := s.Scan(&idValue, &eventName, &actorUserID, &ownerUser, &requesterID, &agentID, &requestID, &outcome, &createdAtRaw); err != nil {
		return auditevent.Event{}, err
	}
	return auditevent.Event{
		ID:              strconv.FormatInt(idValue, 10),
		EventName:       auditevent.Name(eventName),
		ActorUserID:     actorUserID,
		OwnerUserID:     ownerUser,
		RequesterUserID: requesterID,
		AgentID:         agentID,
		AccessRequestID: requestID,
		Outcome:         outcome,
		CreatedAt:       sqliteutil.FromMillis(createdAtRaw),
	}, nil
}

// PutProviderGrant persists a provider grant record.
