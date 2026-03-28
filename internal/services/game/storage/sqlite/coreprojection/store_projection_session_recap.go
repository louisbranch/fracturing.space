package coreprojection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// PutSessionRecap stores or replaces the recap markdown for one session.
func (s *Store) PutSessionRecap(ctx context.Context, recap storage.SessionRecap) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(recap.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(recap.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	_, err := s.projectionQueryable().ExecContext(ctx,
		`INSERT INTO session_recaps (campaign_id, session_id, markdown, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (campaign_id, session_id) DO UPDATE SET
		   markdown = excluded.markdown,
		   updated_at = excluded.updated_at`,
		recap.CampaignID,
		recap.SessionID,
		strings.TrimSpace(recap.Markdown),
		sqliteutil.ToMillis(recap.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("put session recap: %w", err)
	}
	return nil
}

// GetSessionRecap retrieves the stored recap markdown for one session.
func (s *Store) GetSessionRecap(ctx context.Context, campaignID, sessionID string) (storage.SessionRecap, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionRecap{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionRecap{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionRecap{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionRecap{}, fmt.Errorf("session id is required")
	}

	var (
		markdown  string
		updatedAt int64
	)
	err := s.projectionQueryable().QueryRowContext(ctx,
		`SELECT markdown, updated_at
		 FROM session_recaps
		 WHERE campaign_id = ? AND session_id = ?`,
		campaignID,
		sessionID,
	).Scan(&markdown, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionRecap{}, storage.ErrNotFound
		}
		return storage.SessionRecap{}, fmt.Errorf("get session recap: %w", err)
	}

	return storage.SessionRecap{
		CampaignID: campaignID,
		SessionID:  sessionID,
		Markdown:   markdown,
		UpdatedAt:  sqliteutil.FromMillis(updatedAt),
	}, nil
}
