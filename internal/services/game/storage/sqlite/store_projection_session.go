package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// Session methods

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

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if sess.Status == session.StatusActive {
		hasActive, err := qtx.HasActiveSession(ctx, sess.CampaignID)
		if err != nil {
			return fmt.Errorf("check active session: %w", err)
		}
		if hasActive != 0 {
			return storage.ErrActiveSessionExists
		}
	}

	endedAt := toNullMillis(sess.EndedAt)

	if err := qtx.PutSession(ctx, db.PutSessionParams{
		CampaignID: sess.CampaignID,
		ID:         sess.ID,
		Name:       sess.Name,
		Status:     sessionStatusToString(sess.Status),
		StartedAt:  toMillis(sess.StartedAt),
		UpdatedAt:  toMillis(sess.UpdatedAt),
		EndedAt:    endedAt,
	}); err != nil {
		return fmt.Errorf("put session: %w", err)
	}

	if sess.Status == session.StatusActive {
		if err := qtx.SetActiveSession(ctx, db.SetActiveSessionParams{
			CampaignID: sess.CampaignID,
			SessionID:  sess.ID,
		}); err != nil {
			return fmt.Errorf("set active session: %w", err)
		}
	}

	return tx.Commit()
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

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return storage.SessionRecord{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	row, err := qtx.GetSession(ctx, db.GetSessionParams{
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

		if err := qtx.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
			Status:     sessionStatusToString(sess.Status),
			UpdatedAt:  toMillis(sess.UpdatedAt),
			EndedAt:    toNullMillis(sess.EndedAt),
			CampaignID: campaignID,
			ID:         sessionID,
		}); err != nil {
			return storage.SessionRecord{}, false, fmt.Errorf("update session status: %w", err)
		}
	}

	if err := qtx.ClearActiveSession(ctx, campaignID); err != nil {
		return storage.SessionRecord{}, false, fmt.Errorf("clear active session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return storage.SessionRecord{}, false, fmt.Errorf("commit: %w", err)
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

	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		sess, err := dbSessionToDomain(row)
		if err != nil {
			return storage.SessionPage{}, err
		}
		page.Sessions = append(page.Sessions, sess)
	}

	return page, nil
}

// Session gate methods

// PutSessionGate persists a session gate projection.
func (s *Store) PutSessionGate(ctx context.Context, gate storage.SessionGate) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(gate.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(gate.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(gate.GateID) == "" {
		return fmt.Errorf("gate id is required")
	}
	if strings.TrimSpace(gate.GateType) == "" {
		return fmt.Errorf("gate type is required")
	}
	status := strings.TrimSpace(string(gate.Status))
	if status == "" {
		return fmt.Errorf("gate status is required")
	}

	return s.q.PutSessionGate(ctx, db.PutSessionGateParams{
		CampaignID:          gate.CampaignID,
		SessionID:           gate.SessionID,
		GateID:              gate.GateID,
		GateType:            gate.GateType,
		Status:              status,
		Reason:              gate.Reason,
		CreatedAt:           toMillis(gate.CreatedAt),
		CreatedByActorType:  gate.CreatedByActorType,
		CreatedByActorID:    gate.CreatedByActorID,
		ResolvedAt:          toNullMillis(gate.ResolvedAt),
		ResolvedByActorType: toNullString(gate.ResolvedByActorType),
		ResolvedByActorID:   toNullString(gate.ResolvedByActorID),
		MetadataJson:        gate.MetadataJSON,
		ResolutionJson:      gate.ResolutionJSON,
	})
}

// GetSessionGate retrieves a session gate by id.
func (s *Store) GetSessionGate(ctx context.Context, campaignID, sessionID, gateID string) (storage.SessionGate, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionGate{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionGate{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionGate{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionGate{}, fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(gateID) == "" {
		return storage.SessionGate{}, fmt.Errorf("gate id is required")
	}

	row, err := s.q.GetSessionGate(ctx, db.GetSessionGateParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
		GateID:     gateID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionGate{}, storage.ErrNotFound
		}
		return storage.SessionGate{}, fmt.Errorf("get session gate: %w", err)
	}

	return dbSessionGateToStorage(row), nil
}

// GetOpenSessionGate retrieves the open gate for a session.
func (s *Store) GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionGate{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionGate{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionGate{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionGate{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetOpenSessionGate(ctx, db.GetOpenSessionGateParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionGate{}, storage.ErrNotFound
		}
		return storage.SessionGate{}, fmt.Errorf("get open session gate: %w", err)
	}

	return dbSessionGateToStorage(row), nil
}

// Session spotlight methods

// PutSessionSpotlight persists a session spotlight projection.
func (s *Store) PutSessionSpotlight(ctx context.Context, spotlight storage.SessionSpotlight) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(spotlight.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(spotlight.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	spotlightType := strings.TrimSpace(string(spotlight.SpotlightType))
	if spotlightType == "" {
		return fmt.Errorf("spotlight type is required")
	}

	return s.q.PutSessionSpotlight(ctx, db.PutSessionSpotlightParams{
		CampaignID:         spotlight.CampaignID,
		SessionID:          spotlight.SessionID,
		SpotlightType:      spotlightType,
		CharacterID:        spotlight.CharacterID,
		UpdatedAt:          toMillis(spotlight.UpdatedAt),
		UpdatedByActorType: spotlight.UpdatedByActorType,
		UpdatedByActorID:   spotlight.UpdatedByActorID,
	})
}

// GetSessionSpotlight retrieves a session spotlight by session id.
func (s *Store) GetSessionSpotlight(ctx context.Context, campaignID, sessionID string) (storage.SessionSpotlight, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionSpotlight{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionSpotlight{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionSpotlight{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionSpotlight{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetSessionSpotlight(ctx, db.GetSessionSpotlightParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionSpotlight{}, storage.ErrNotFound
		}
		return storage.SessionSpotlight{}, fmt.Errorf("get session spotlight: %w", err)
	}

	return dbSessionSpotlightToStorage(row), nil
}

// ClearSessionSpotlight removes the current spotlight for a session.
func (s *Store) ClearSessionSpotlight(ctx context.Context, campaignID, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	return s.q.ClearSessionSpotlight(ctx, db.ClearSessionSpotlightParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
}
