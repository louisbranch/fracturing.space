package coreprojection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// PutSession atomically stores a session and sets it as the active session for the campaign.
func (s *Store) PutSession(ctx context.Context, sess storage.SessionRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(sess.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sess.ID) == "" {
		return fmt.Errorf("session id is required")
	}

	if s.tx != nil {
		return putSessionWithQueries(ctx, s.q, sess)
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)
	if err := putSessionWithQueries(ctx, qtx, sess); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// EndSession marks a session as ended and clears it as active for the campaign.
func (s *Store) EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (storage.SessionRecord, bool, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionRecord{}, false, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionRecord{}, false, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionRecord{}, false, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionRecord{}, false, fmt.Errorf("session id is required")
	}

	if s.tx != nil {
		return endSessionWithQueries(ctx, s.q, campaignID, sessionID, endedAt)
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return storage.SessionRecord{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)
	sess, transitioned, err := endSessionWithQueries(ctx, qtx, campaignID, sessionID, endedAt)
	if err != nil {
		return storage.SessionRecord{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return storage.SessionRecord{}, false, fmt.Errorf("commit: %w", err)
	}

	return sess, transitioned, nil
}

func putSessionWithQueries(ctx context.Context, queries *db.Queries, sess storage.SessionRecord) error {
	if sess.Status == session.StatusActive {
		hasActive, err := queries.HasActiveSession(ctx, sess.CampaignID)
		if err != nil {
			return fmt.Errorf("check active session: %w", err)
		}
		if hasActive != 0 {
			return storage.ErrActiveSessionExists
		}
	}

	endedAt := sqliteutil.ToNullMillis(sess.EndedAt)

	if err := queries.PutSession(ctx, db.PutSessionParams{
		CampaignID: sess.CampaignID,
		ID:         sess.ID,
		Name:       sess.Name,
		Status:     enumToStorage(sess.Status),
		StartedAt:  sqliteutil.ToMillis(sess.StartedAt),
		UpdatedAt:  sqliteutil.ToMillis(sess.UpdatedAt),
		EndedAt:    endedAt,
	}); err != nil {
		return fmt.Errorf("put session: %w", err)
	}

	if sess.Status == session.StatusActive {
		if err := queries.SetActiveSession(ctx, db.SetActiveSessionParams{
			CampaignID: sess.CampaignID,
			SessionID:  sess.ID,
		}); err != nil {
			return fmt.Errorf("set active session: %w", err)
		}
	}

	return nil
}

func endSessionWithQueries(ctx context.Context, queries *db.Queries, campaignID, sessionID string, endedAt time.Time) (storage.SessionRecord, bool, error) {
	row, err := queries.GetSession(ctx, db.GetSessionParams{
		CampaignID: campaignID,
		ID:         sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionRecord{}, false, storage.ErrNotFound
		}
		return storage.SessionRecord{}, false, fmt.Errorf("get session: %w", err)
	}

	sess, err := dbSessionToDomain(row)
	if err != nil {
		return storage.SessionRecord{}, false, err
	}

	transitioned := false
	if sess.Status == session.StatusActive {
		transitioned = true
		sess.Status = session.StatusEnded
		sess.UpdatedAt = endedAt.UTC()
		sess.EndedAt = &sess.UpdatedAt

		if err := queries.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
			Status:     enumToStorage(sess.Status),
			UpdatedAt:  sqliteutil.ToMillis(sess.UpdatedAt),
			EndedAt:    sqliteutil.ToNullMillis(sess.EndedAt),
			CampaignID: campaignID,
			ID:         sessionID,
		}); err != nil {
			return storage.SessionRecord{}, false, fmt.Errorf("update session status: %w", err)
		}
	}

	if err := queries.ClearActiveSession(ctx, campaignID); err != nil {
		return storage.SessionRecord{}, false, fmt.Errorf("clear active session: %w", err)
	}

	return sess, transitioned, nil
}

// GetSession retrieves a session by campaign ID and session ID.
func (s *Store) GetSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionRecord{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionRecord{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetSession(ctx, db.GetSessionParams{
		CampaignID: campaignID,
		ID:         sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionRecord{}, storage.ErrNotFound
		}
		return storage.SessionRecord{}, fmt.Errorf("get session: %w", err)
	}

	return dbSessionToDomain(row)
}

// GetActiveSession retrieves the active session for a campaign.
func (s *Store) GetActiveSession(ctx context.Context, campaignID string) (storage.SessionRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionRecord{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetActiveSession(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionRecord{}, storage.ErrNotFound
		}
		return storage.SessionRecord{}, fmt.Errorf("get active session: %w", err)
	}

	return dbSessionToDomain(row)
}

// CountSessions returns the number of sessions for a campaign.
func (s *Store) CountSessions(ctx context.Context, campaignID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return 0, fmt.Errorf("campaign id is required")
	}
	var count int64
	err := s.projectionQueryable().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sessions WHERE campaign_id = ?", campaignID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count sessions: %w", err)
	}
	return int(count), nil
}

// ListSessions returns a page of session records.
func (s *Store) ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.SessionPage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.Session
	var err error

	if pageToken == "" {
		rows, err = s.q.ListSessionsByCampaignPagedFirst(ctx, db.ListSessionsByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListSessionsByCampaignPaged(ctx, db.ListSessionsByCampaignPagedParams{
			CampaignID: campaignID,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.SessionPage{}, fmt.Errorf("list sessions: %w", err)
	}

	page := storage.SessionPage{
		Sessions: make([]storage.SessionRecord, 0, pageSize),
	}

	sessions, nextPageToken, err := sqliteutil.MapPageRows(rows, pageSize, func(row db.Session) string {
		return row.ID
	}, dbSessionToDomain)
	if err != nil {
		return storage.SessionPage{}, err
	}
	page.Sessions = sessions
	page.NextPageToken = nextPageToken

	return page, nil
}
