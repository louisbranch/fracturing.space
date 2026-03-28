package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func (s *Store) PutAccessRequest(ctx context.Context, request accessrequest.AccessRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if request.ID == "" {
		return fmt.Errorf("access request id is required")
	}
	if request.RequesterUserID == "" {
		return fmt.Errorf("requester user id is required")
	}
	if request.OwnerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if request.RequesterUserID == request.OwnerUserID {
		return fmt.Errorf("requester user id must differ from owner user id")
	}
	if request.AgentID == "" {
		return fmt.Errorf("agent id is required")
	}
	if request.Scope == "" {
		return fmt.Errorf("scope is required")
	}
	if request.Status == "" {
		return fmt.Errorf("status is required")
	}

	var reviewedAt sql.NullInt64
	if request.ReviewedAt != nil {
		reviewedAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*request.ReviewedAt), Valid: true}
	}
	var revokedAt sql.NullInt64
	if request.RevokedAt != nil {
		revokedAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*request.RevokedAt), Valid: true}
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_access_requests (
	id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, revoker_user_id, revoke_note, created_at, updated_at, reviewed_at, revoked_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	requester_user_id = excluded.requester_user_id,
	owner_user_id = excluded.owner_user_id,
	agent_id = excluded.agent_id,
	scope = excluded.scope,
	request_note = excluded.request_note,
	status = excluded.status,
	reviewer_user_id = excluded.reviewer_user_id,
	review_note = excluded.review_note,
	revoker_user_id = excluded.revoker_user_id,
	revoke_note = excluded.revoke_note,
	updated_at = excluded.updated_at,
	reviewed_at = excluded.reviewed_at,
	revoked_at = excluded.revoked_at
`,
		request.ID,
		request.RequesterUserID,
		request.OwnerUserID,
		request.AgentID,
		string(request.Scope),
		request.RequestNote,
		string(request.Status),
		request.ReviewerUserID,
		request.ReviewNote,
		request.RevokerUserID,
		request.RevokeNote,
		sqliteutil.ToMillis(request.CreatedAt),
		sqliteutil.ToMillis(request.UpdatedAt),
		reviewedAt,
		revokedAt,
	)
	if err != nil {
		return fmt.Errorf("put access request: %w", err)
	}
	return nil
}

// GetAccessRequest fetches an access request by ID.
func (s *Store) GetAccessRequest(ctx context.Context, accessRequestID string) (accessrequest.AccessRequest, error) {
	if err := ctx.Err(); err != nil {
		return accessrequest.AccessRequest{}, err
	}
	if s == nil || s.sqlDB == nil {
		return accessrequest.AccessRequest{}, fmt.Errorf("storage is not configured")
	}
	if accessRequestID == "" {
		return accessrequest.AccessRequest{}, fmt.Errorf("access request id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, revoker_user_id, revoke_note, created_at, updated_at, reviewed_at, revoked_at
FROM ai_access_requests
WHERE id = ?
`, accessRequestID)

	rec, err := scanAccessRequest(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return accessrequest.AccessRequest{}, storage.ErrNotFound
		}
		return accessrequest.AccessRequest{}, fmt.Errorf("get access request: %w", err)
	}
	return rec, nil
}

// ListAccessRequestsByRequester returns a page of access requests by requester.
func (s *Store) ListAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (accessrequest.Page, error) {
	if err := ctx.Err(); err != nil {
		return accessrequest.Page{}, err
	}
	db, err := requireStoreDB(s)
	if err != nil {
		return accessrequest.Page{}, err
	}
	if requesterUserID == "" {
		return accessrequest.Page{}, fmt.Errorf("requester user id is required")
	}
	limit, err := keysetPageLimit(pageSize)
	if err != nil {
		return accessrequest.Page{}, err
	}

	var rows *sql.Rows
	if pageToken == "" {
		rows, err = db.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, revoker_user_id, revoke_note, created_at, updated_at, reviewed_at, revoked_at
FROM ai_access_requests
WHERE requester_user_id = ?
ORDER BY id
LIMIT ?
`, requesterUserID, limit)
	} else {
		rows, err = db.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, revoker_user_id, revoke_note, created_at, updated_at, reviewed_at, revoked_at
FROM ai_access_requests
WHERE requester_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, requesterUserID, pageToken, limit)
	}
	if err != nil {
		return accessrequest.Page{}, fmt.Errorf("list access requests by requester: %w", err)
	}
	defer rows.Close()

	accessRequests, nextPageToken, err := scanIDKeysetPage(rows, pageSize, scanAccessRequest, "access request", func(request accessrequest.AccessRequest) string {
		return request.ID
	})
	if err != nil {
		return accessrequest.Page{}, err
	}
	return accessrequest.Page{AccessRequests: accessRequests, NextPageToken: nextPageToken}, nil
}

// ListAccessRequestsByOwner returns a page of access requests by owner.
func (s *Store) ListAccessRequestsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (accessrequest.Page, error) {
	if err := ctx.Err(); err != nil {
		return accessrequest.Page{}, err
	}
	db, err := requireStoreDB(s)
	if err != nil {
		return accessrequest.Page{}, err
	}
	if ownerUserID == "" {
		return accessrequest.Page{}, fmt.Errorf("owner user id is required")
	}
	limit, err := keysetPageLimit(pageSize)
	if err != nil {
		return accessrequest.Page{}, err
	}

	var rows *sql.Rows
	if pageToken == "" {
		rows, err = db.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, revoker_user_id, revoke_note, created_at, updated_at, reviewed_at, revoked_at
FROM ai_access_requests
WHERE owner_user_id = ?
ORDER BY id
LIMIT ?
`, ownerUserID, limit)
	} else {
		rows, err = db.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, revoker_user_id, revoke_note, created_at, updated_at, reviewed_at, revoked_at
FROM ai_access_requests
WHERE owner_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, ownerUserID, pageToken, limit)
	}
	if err != nil {
		return accessrequest.Page{}, fmt.Errorf("list access requests by owner: %w", err)
	}
	defer rows.Close()

	accessRequests, nextPageToken, err := scanIDKeysetPage(rows, pageSize, scanAccessRequest, "access request", func(request accessrequest.AccessRequest) string {
		return request.ID
	})
	if err != nil {
		return accessrequest.Page{}, err
	}
	return accessrequest.Page{AccessRequests: accessRequests, NextPageToken: nextPageToken}, nil
}

// GetApprovedInvokeAccessByRequesterForAgent returns one approved invoke access
// request for a requester/owner/agent tuple.
func (s *Store) GetApprovedInvokeAccessByRequesterForAgent(ctx context.Context, requesterUserID string, ownerUserID string, agentID string) (accessrequest.AccessRequest, error) {
	if err := ctx.Err(); err != nil {
		return accessrequest.AccessRequest{}, err
	}
	if s == nil || s.sqlDB == nil {
		return accessrequest.AccessRequest{}, fmt.Errorf("storage is not configured")
	}
	if requesterUserID == "" {
		return accessrequest.AccessRequest{}, fmt.Errorf("requester user id is required")
	}
	if ownerUserID == "" {
		return accessrequest.AccessRequest{}, fmt.Errorf("owner user id is required")
	}
	if agentID == "" {
		return accessrequest.AccessRequest{}, fmt.Errorf("agent id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, revoker_user_id, revoke_note, created_at, updated_at, reviewed_at, revoked_at
FROM ai_access_requests
WHERE requester_user_id = ? AND owner_user_id = ? AND agent_id = ? AND scope = 'invoke' AND status = 'approved'
ORDER BY id
LIMIT 1
`, requesterUserID, ownerUserID, agentID)

	rec, err := scanAccessRequest(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return accessrequest.AccessRequest{}, storage.ErrNotFound
		}
		return accessrequest.AccessRequest{}, fmt.Errorf("get approved invoke access request: %w", err)
	}
	return rec, nil
}

// ListApprovedInvokeAccessRequestsByRequester returns approved invoke access
// requests for one requester.
func (s *Store) ListApprovedInvokeAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (accessrequest.Page, error) {
	if err := ctx.Err(); err != nil {
		return accessrequest.Page{}, err
	}
	db, err := requireStoreDB(s)
	if err != nil {
		return accessrequest.Page{}, err
	}
	if requesterUserID == "" {
		return accessrequest.Page{}, fmt.Errorf("requester user id is required")
	}
	limit, err := keysetPageLimit(pageSize)
	if err != nil {
		return accessrequest.Page{}, err
	}

	var rows *sql.Rows
	if pageToken == "" {
		rows, err = db.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, revoker_user_id, revoke_note, created_at, updated_at, reviewed_at, revoked_at
FROM ai_access_requests
WHERE requester_user_id = ? AND scope = 'invoke' AND status = 'approved'
ORDER BY id
LIMIT ?
`, requesterUserID, limit)
	} else {
		rows, err = db.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, revoker_user_id, revoke_note, created_at, updated_at, reviewed_at, revoked_at
FROM ai_access_requests
WHERE requester_user_id = ? AND scope = 'invoke' AND status = 'approved' AND id > ?
ORDER BY id
LIMIT ?
`, requesterUserID, pageToken, limit)
	}
	if err != nil {
		return accessrequest.Page{}, fmt.Errorf("list approved invoke access requests by requester: %w", err)
	}
	defer rows.Close()

	accessRequests, nextPageToken, err := scanIDKeysetPage(rows, pageSize, scanAccessRequest, "access request", func(request accessrequest.AccessRequest) string {
		return request.ID
	})
	if err != nil {
		return accessrequest.Page{}, err
	}
	return accessrequest.Page{AccessRequests: accessRequests, NextPageToken: nextPageToken}, nil
}

// ReviewAccessRequest applies an owner review decision for one pending request.
// It extracts fields from the reviewed domain object and performs a CAS update
// against the current pending status.
func (s *Store) ReviewAccessRequest(ctx context.Context, reviewed accessrequest.AccessRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if reviewed.OwnerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if reviewed.ID == "" {
		return fmt.Errorf("access request id is required")
	}
	if reviewed.Status == "" {
		return fmt.Errorf("status is required")
	}
	if reviewed.ReviewerUserID == "" {
		return fmt.Errorf("reviewer user id is required")
	}
	if reviewed.ReviewerUserID != reviewed.OwnerUserID {
		return fmt.Errorf("reviewer user id must match owner user id")
	}
	if reviewed.ReviewedAt == nil {
		return fmt.Errorf("reviewed_at is required")
	}
	reviewedAt := reviewed.ReviewedAt.UTC()

	var existingStatus string
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT status
FROM ai_access_requests
WHERE owner_user_id = ? AND id = ?
`, reviewed.OwnerUserID, reviewed.ID)
	if err := row.Scan(&existingStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("check access request status: %w", err)
	}
	if existingStatus != "pending" {
		return storage.ErrConflict
	}

	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_access_requests
SET status = ?, reviewer_user_id = ?, review_note = ?, revoker_user_id = '', revoke_note = '', reviewed_at = ?, revoked_at = NULL, updated_at = ?
WHERE owner_user_id = ? AND id = ? AND status = 'pending'
`, string(reviewed.Status), reviewed.ReviewerUserID, reviewed.ReviewNote, sqliteutil.ToMillis(reviewedAt), sqliteutil.ToMillis(reviewedAt), reviewed.OwnerUserID, reviewed.ID)
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
func (s *Store) RevokeAccessRequest(ctx context.Context, revoked accessrequest.AccessRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if revoked.OwnerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if revoked.ID == "" {
		return fmt.Errorf("access request id is required")
	}
	if revoked.Status == "" {
		return fmt.Errorf("status is required")
	}
	if revoked.RevokerUserID == "" {
		return fmt.Errorf("revoker user id is required")
	}
	if revoked.RevokerUserID != revoked.OwnerUserID {
		return fmt.Errorf("revoker user id must match owner user id")
	}
	if revoked.RevokedAt == nil {
		return fmt.Errorf("revoked_at is required")
	}
	revokedAt := revoked.RevokedAt.UTC()

	var existingStatus string
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT status
FROM ai_access_requests
WHERE owner_user_id = ? AND id = ?
`, revoked.OwnerUserID, revoked.ID)
	if err := row.Scan(&existingStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("check access request status: %w", err)
	}
	if existingStatus != "approved" {
		return storage.ErrConflict
	}

	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_access_requests
SET status = ?, revoker_user_id = ?, revoke_note = ?, revoked_at = ?, updated_at = ?
WHERE owner_user_id = ? AND id = ? AND status = 'approved'
`, string(revoked.Status), revoked.RevokerUserID, revoked.RevokeNote, sqliteutil.ToMillis(revokedAt), sqliteutil.ToMillis(revokedAt), revoked.OwnerUserID, revoked.ID)
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
