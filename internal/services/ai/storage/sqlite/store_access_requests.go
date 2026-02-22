package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func (s *Store) PutAccessRequest(ctx context.Context, record storage.AccessRequestRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("access request id is required")
	}
	if strings.TrimSpace(record.RequesterUserID) == "" {
		return fmt.Errorf("requester user id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.RequesterUserID) == strings.TrimSpace(record.OwnerUserID) {
		return fmt.Errorf("requester user id must differ from owner user id")
	}
	if strings.TrimSpace(record.AgentID) == "" {
		return fmt.Errorf("agent id is required")
	}
	if strings.TrimSpace(record.Scope) == "" {
		return fmt.Errorf("scope is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}

	var reviewedAt sql.NullInt64
	if record.ReviewedAt != nil {
		reviewedAt = sql.NullInt64{Int64: toMillis(*record.ReviewedAt), Valid: true}
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_access_requests (
	id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	requester_user_id = excluded.requester_user_id,
	owner_user_id = excluded.owner_user_id,
	agent_id = excluded.agent_id,
	scope = excluded.scope,
	request_note = excluded.request_note,
	status = excluded.status,
	reviewer_user_id = excluded.reviewer_user_id,
	review_note = excluded.review_note,
	updated_at = excluded.updated_at,
	reviewed_at = excluded.reviewed_at
`,
		record.ID,
		record.RequesterUserID,
		record.OwnerUserID,
		record.AgentID,
		record.Scope,
		strings.TrimSpace(record.RequestNote),
		record.Status,
		strings.TrimSpace(record.ReviewerUserID),
		strings.TrimSpace(record.ReviewNote),
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
		reviewedAt,
	)
	if err != nil {
		return fmt.Errorf("put access request: %w", err)
	}
	return nil
}

// GetAccessRequest fetches an access request record by ID.
func (s *Store) GetAccessRequest(ctx context.Context, accessRequestID string) (storage.AccessRequestRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestRecord{}, fmt.Errorf("storage is not configured")
	}
	accessRequestID = strings.TrimSpace(accessRequestID)
	if accessRequestID == "" {
		return storage.AccessRequestRecord{}, fmt.Errorf("access request id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE id = ?
`, accessRequestID)

	rec, err := scanAccessRequestRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.AccessRequestRecord{}, storage.ErrNotFound
		}
		return storage.AccessRequestRecord{}, fmt.Errorf("get access request: %w", err)
	}
	return rec, nil
}

// ListAccessRequestsByRequester returns a page of access requests by requester.
func (s *Store) ListAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (storage.AccessRequestPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestPage{}, fmt.Errorf("storage is not configured")
	}
	requesterUserID = strings.TrimSpace(requesterUserID)
	if requesterUserID == "" {
		return storage.AccessRequestPage{}, fmt.Errorf("requester user id is required")
	}
	if pageSize <= 0 {
		return storage.AccessRequestPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ?
ORDER BY id
LIMIT ?
`, requesterUserID, limit)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, requesterUserID, strings.TrimSpace(pageToken), limit)
	}
	if err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("list access requests by requester: %w", err)
	}
	defer rows.Close()

	page := storage.AccessRequestPage{AccessRequests: make([]storage.AccessRequestRecord, 0, pageSize)}
	for rows.Next() {
		rec, err := scanAccessRequestRows(rows)
		if err != nil {
			return storage.AccessRequestPage{}, fmt.Errorf("scan access request row: %w", err)
		}
		page.AccessRequests = append(page.AccessRequests, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("iterate access request rows: %w", err)
	}
	if len(page.AccessRequests) > pageSize {
		page.NextPageToken = page.AccessRequests[pageSize-1].ID
		page.AccessRequests = page.AccessRequests[:pageSize]
	}
	return page, nil
}

// ListAccessRequestsByOwner returns a page of access requests by owner.
func (s *Store) ListAccessRequestsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (storage.AccessRequestPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.AccessRequestPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.AccessRequestPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE owner_user_id = ?
ORDER BY id
LIMIT ?
`, ownerUserID, limit)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE owner_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, ownerUserID, strings.TrimSpace(pageToken), limit)
	}
	if err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("list access requests by owner: %w", err)
	}
	defer rows.Close()

	page := storage.AccessRequestPage{AccessRequests: make([]storage.AccessRequestRecord, 0, pageSize)}
	for rows.Next() {
		rec, err := scanAccessRequestRows(rows)
		if err != nil {
			return storage.AccessRequestPage{}, fmt.Errorf("scan access request row: %w", err)
		}
		page.AccessRequests = append(page.AccessRequests, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("iterate access request rows: %w", err)
	}
	if len(page.AccessRequests) > pageSize {
		page.NextPageToken = page.AccessRequests[pageSize-1].ID
		page.AccessRequests = page.AccessRequests[:pageSize]
	}
	return page, nil
}

// GetApprovedInvokeAccessByRequesterForAgent returns one approved invoke access
// request for a requester/owner/agent tuple.
func (s *Store) GetApprovedInvokeAccessByRequesterForAgent(ctx context.Context, requesterUserID string, ownerUserID string, agentID string) (storage.AccessRequestRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestRecord{}, fmt.Errorf("storage is not configured")
	}
	requesterUserID = strings.TrimSpace(requesterUserID)
	if requesterUserID == "" {
		return storage.AccessRequestRecord{}, fmt.Errorf("requester user id is required")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.AccessRequestRecord{}, fmt.Errorf("owner user id is required")
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return storage.AccessRequestRecord{}, fmt.Errorf("agent id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ? AND owner_user_id = ? AND agent_id = ? AND scope = 'invoke' AND status = 'approved'
ORDER BY id
LIMIT 1
`, requesterUserID, ownerUserID, agentID)

	rec, err := scanAccessRequestRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.AccessRequestRecord{}, storage.ErrNotFound
		}
		return storage.AccessRequestRecord{}, fmt.Errorf("get approved invoke access request: %w", err)
	}
	return rec, nil
}

// ListApprovedInvokeAccessRequestsByRequester returns approved invoke access
// requests for one requester.
func (s *Store) ListApprovedInvokeAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (storage.AccessRequestPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestPage{}, fmt.Errorf("storage is not configured")
	}
	requesterUserID = strings.TrimSpace(requesterUserID)
	if requesterUserID == "" {
		return storage.AccessRequestPage{}, fmt.Errorf("requester user id is required")
	}
	if pageSize <= 0 {
		return storage.AccessRequestPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ? AND scope = 'invoke' AND status = 'approved'
ORDER BY id
LIMIT ?
`, requesterUserID, limit)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ? AND scope = 'invoke' AND status = 'approved' AND id > ?
ORDER BY id
LIMIT ?
`, requesterUserID, strings.TrimSpace(pageToken), limit)
	}
	if err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("list approved invoke access requests by requester: %w", err)
	}
	defer rows.Close()

	page := storage.AccessRequestPage{AccessRequests: make([]storage.AccessRequestRecord, 0, pageSize)}
	for rows.Next() {
		rec, err := scanAccessRequestRows(rows)
		if err != nil {
			return storage.AccessRequestPage{}, fmt.Errorf("scan access request row: %w", err)
		}
		page.AccessRequests = append(page.AccessRequests, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("iterate access request rows: %w", err)
	}
	if len(page.AccessRequests) > pageSize {
		page.NextPageToken = page.AccessRequests[pageSize-1].ID
		page.AccessRequests = page.AccessRequests[:pageSize]
	}
	return page, nil
}

// ReviewAccessRequest applies an owner review decision for one pending request.
func (s *Store) ReviewAccessRequest(ctx context.Context, ownerUserID string, accessRequestID string, status string, reviewerUserID string, reviewNote string, reviewedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	accessRequestID = strings.TrimSpace(accessRequestID)
	if accessRequestID == "" {
		return fmt.Errorf("access request id is required")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return fmt.Errorf("status is required")
	}
	reviewerUserID = strings.TrimSpace(reviewerUserID)
	if reviewerUserID == "" {
		return fmt.Errorf("reviewer user id is required")
	}
	// Owner-scoped reviews must attribute the decision to the same owner.
	if reviewerUserID != ownerUserID {
		return fmt.Errorf("reviewer user id must match owner user id")
	}

	var existingStatus string
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT status
FROM ai_access_requests
WHERE owner_user_id = ? AND id = ?
`, ownerUserID, accessRequestID)
	if err := row.Scan(&existingStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("check access request status: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(existingStatus), "pending") {
		return storage.ErrConflict
	}

	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_access_requests
SET status = ?, reviewer_user_id = ?, review_note = ?, reviewed_at = ?, updated_at = ?
WHERE owner_user_id = ? AND id = ? AND status = 'pending'
`, status, reviewerUserID, strings.TrimSpace(reviewNote), toMillis(reviewedAt.UTC()), toMillis(reviewedAt.UTC()), ownerUserID, accessRequestID)
	if err != nil {
		return fmt.Errorf("review access request: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("review access request rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrConflict
	}
	return nil
}

// RevokeAccessRequest applies an owner revocation for one approved request.
func (s *Store) RevokeAccessRequest(ctx context.Context, ownerUserID string, accessRequestID string, status string, reviewerUserID string, reviewNote string, revokedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	accessRequestID = strings.TrimSpace(accessRequestID)
	if accessRequestID == "" {
		return fmt.Errorf("access request id is required")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return fmt.Errorf("status is required")
	}
	reviewerUserID = strings.TrimSpace(reviewerUserID)
	if reviewerUserID == "" {
		return fmt.Errorf("reviewer user id is required")
	}
	// Owner-scoped revocations must attribute the action to the same owner.
	if reviewerUserID != ownerUserID {
		return fmt.Errorf("reviewer user id must match owner user id")
	}

	var existingStatus string
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT status
FROM ai_access_requests
WHERE owner_user_id = ? AND id = ?
`, ownerUserID, accessRequestID)
	if err := row.Scan(&existingStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("check access request status: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(existingStatus), "approved") {
		return storage.ErrConflict
	}

	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_access_requests
SET status = ?, reviewer_user_id = ?, review_note = ?, updated_at = ?
WHERE owner_user_id = ? AND id = ? AND status = 'approved'
`, status, reviewerUserID, strings.TrimSpace(reviewNote), toMillis(revokedAt.UTC()), ownerUserID, accessRequestID)
	if err != nil {
		return fmt.Errorf("revoke access request: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke access request rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrConflict
	}
	return nil
}

// PutAuditEvent appends one AI audit event row.
