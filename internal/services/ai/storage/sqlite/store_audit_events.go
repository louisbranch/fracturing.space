package sqlite

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func (s *Store) PutAuditEvent(ctx context.Context, record storage.AuditEventRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.EventName) == "" {
		return fmt.Errorf("event name is required")
	}
	if strings.TrimSpace(record.ActorUserID) == "" {
		return fmt.Errorf("actor user id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.RequesterUserID) == "" {
		return fmt.Errorf("requester user id is required")
	}
	if strings.TrimSpace(record.AgentID) == "" {
		return fmt.Errorf("agent id is required")
	}
	if strings.TrimSpace(record.AccessRequestID) == "" {
		return fmt.Errorf("access request id is required")
	}
	if strings.TrimSpace(record.Outcome) == "" {
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
		strings.TrimSpace(record.EventName),
		strings.TrimSpace(record.ActorUserID),
		strings.TrimSpace(record.OwnerUserID),
		strings.TrimSpace(record.RequesterUserID),
		strings.TrimSpace(record.AgentID),
		strings.TrimSpace(record.AccessRequestID),
		strings.TrimSpace(record.Outcome),
		toMillis(record.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("put audit event: %w", err)
	}
	return nil
}

// ListAuditEventsByOwner returns a page of audit events scoped to one owner.
func (s *Store) ListAuditEventsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter storage.AuditEventFilter) (storage.AuditEventPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.AuditEventPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AuditEventPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.AuditEventPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.AuditEventPage{}, fmt.Errorf("page size must be greater than zero")
	}
	eventName := strings.TrimSpace(filter.EventName)
	agentID := strings.TrimSpace(filter.AgentID)

	var (
		createdAfterMillis  *int64
		createdBeforeMillis *int64
	)
	if filter.CreatedAfter != nil {
		value := toMillis(filter.CreatedAfter.UTC())
		createdAfterMillis = &value
	}
	if filter.CreatedBefore != nil {
		value := toMillis(filter.CreatedBefore.UTC())
		createdBeforeMillis = &value
	}
	if createdAfterMillis != nil && createdBeforeMillis != nil && *createdAfterMillis > *createdBeforeMillis {
		return storage.AuditEventPage{}, fmt.Errorf("created_after must be before or equal to created_before")
	}

	limit := pageSize + 1
	pageToken = strings.TrimSpace(pageToken)
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
			return storage.AuditEventPage{}, fmt.Errorf("invalid page token")
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
	rows, err := s.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return storage.AuditEventPage{}, fmt.Errorf("list audit events by owner: %w", err)
	}
	defer rows.Close()

	page := storage.AuditEventPage{AuditEvents: make([]storage.AuditEventRecord, 0, pageSize)}
	for rows.Next() {
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
		if err := rows.Scan(&idValue, &eventName, &actorUserID, &ownerUser, &requesterID, &agentID, &requestID, &outcome, &createdAtRaw); err != nil {
			return storage.AuditEventPage{}, fmt.Errorf("scan audit event row: %w", err)
		}
		page.AuditEvents = append(page.AuditEvents, storage.AuditEventRecord{
			ID:              strconv.FormatInt(idValue, 10),
			EventName:       eventName,
			ActorUserID:     actorUserID,
			OwnerUserID:     ownerUser,
			RequesterUserID: requesterID,
			AgentID:         agentID,
			AccessRequestID: requestID,
			Outcome:         outcome,
			CreatedAt:       fromMillis(createdAtRaw),
		})
	}
	if err := rows.Err(); err != nil {
		return storage.AuditEventPage{}, fmt.Errorf("iterate audit event rows: %w", err)
	}
	if len(page.AuditEvents) > pageSize {
		page.NextPageToken = page.AuditEvents[pageSize-1].ID
		page.AuditEvents = page.AuditEvents[:pageSize]
	}
	return page, nil
}

// PutProviderGrant persists a provider grant record.
