// Package sqlite provides a SQLite-backed invite storage implementation.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
	"github.com/louisbranch/fracturing.space/internal/services/invite/storage/sqlite/migrations"
)

const timeLayout = time.RFC3339Nano

// Store persists invite state in SQLite.
type Store struct {
	sqlDB *sql.DB
}

// Open opens a SQLite invite store and applies embedded migrations.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}
	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}
	if err := sqlitemigrate.ApplyMigrations(sqlDB, migrations.FS, "", time.Now); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return &Store{sqlDB: sqlDB}, nil
}

// Close closes the SQLite handle.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

var _ storage.InviteStore = (*Store)(nil)
var _ storage.OutboxStore = (*Store)(nil)

func (s *Store) GetInvite(_ context.Context, inviteID string) (storage.InviteRecord, error) {
	row := s.sqlDB.QueryRow(
		`SELECT id, campaign_id, participant_id, recipient_user_id, status, created_by_participant_id, created_at, updated_at
		 FROM invites WHERE id = ?`, inviteID)
	return scanInvite(row)
}

func (s *Store) PutInvite(_ context.Context, inv storage.InviteRecord) error {
	_, err := s.sqlDB.Exec(
		`INSERT INTO invites (id, campaign_id, participant_id, recipient_user_id, status, created_by_participant_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   status = excluded.status,
		   updated_at = excluded.updated_at`,
		inv.ID, inv.CampaignID, inv.ParticipantID, inv.RecipientUserID,
		string(inv.Status), inv.CreatedByParticipantID,
		inv.CreatedAt.Format(timeLayout), inv.UpdatedAt.Format(timeLayout),
	)
	return err
}

func (s *Store) UpdateInviteStatus(_ context.Context, inviteID string, status storage.Status, updatedAt time.Time) error {
	res, err := s.sqlDB.Exec(
		`UPDATE invites SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), updatedAt.Format(timeLayout), inviteID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) ListInvites(_ context.Context, campaignID, recipientUserID string, status storage.Status, pageSize int, pageToken string) (storage.InvitePage, error) {
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 10
	}
	var args []any
	var conditions []string
	if campaignID != "" {
		conditions = append(conditions, "campaign_id = ?")
		args = append(args, campaignID)
	}
	if recipientUserID != "" {
		conditions = append(conditions, "recipient_user_id = ?")
		args = append(args, recipientUserID)
	}
	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, string(status))
	}
	if pageToken != "" {
		conditions = append(conditions, "id > ?")
		args = append(args, pageToken)
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	args = append(args, pageSize+1)

	rows, err := s.sqlDB.Query(
		fmt.Sprintf(`SELECT id, campaign_id, participant_id, recipient_user_id, status, created_by_participant_id, created_at, updated_at
		 FROM invites %s ORDER BY id ASC LIMIT ?`, where), args...)
	if err != nil {
		return storage.InvitePage{}, err
	}
	defer rows.Close()
	return collectInvitePage(rows, pageSize)
}

func (s *Store) ListPendingInvites(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 10
	}
	var args []any
	query := `SELECT id, campaign_id, participant_id, recipient_user_id, status, created_by_participant_id, created_at, updated_at
		 FROM invites WHERE campaign_id = ? AND status = 'pending'`
	args = append(args, campaignID)
	if pageToken != "" {
		query += " AND id > ?"
		args = append(args, pageToken)
	}
	query += " ORDER BY id ASC LIMIT ?"
	args = append(args, pageSize+1)

	rows, err := s.sqlDB.Query(query, args...)
	if err != nil {
		return storage.InvitePage{}, err
	}
	defer rows.Close()
	return collectInvitePage(rows, pageSize)
}

func (s *Store) ListPendingInvitesForRecipient(_ context.Context, userID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 10
	}
	var args []any
	query := `SELECT id, campaign_id, participant_id, recipient_user_id, status, created_by_participant_id, created_at, updated_at
		 FROM invites WHERE recipient_user_id = ? AND status = 'pending'`
	args = append(args, userID)
	if pageToken != "" {
		query += " AND id > ?"
		args = append(args, pageToken)
	}
	query += " ORDER BY id ASC LIMIT ?"
	args = append(args, pageSize+1)

	rows, err := s.sqlDB.Query(query, args...)
	if err != nil {
		return storage.InvitePage{}, err
	}
	defer rows.Close()
	return collectInvitePage(rows, pageSize)
}

func (s *Store) Enqueue(_ context.Context, evt storage.OutboxEvent) error {
	_, err := s.sqlDB.Exec(
		`INSERT INTO outbox (id, event_type, payload_json, dedupe_key, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT DO NOTHING`,
		evt.ID, evt.EventType, string(evt.PayloadJSON), evt.DedupeKey,
		evt.CreatedAt.Format(timeLayout),
	)
	return err
}

func scanInvite(row *sql.Row) (storage.InviteRecord, error) {
	var inv storage.InviteRecord
	var status, createdAt, updatedAt string
	err := row.Scan(&inv.ID, &inv.CampaignID, &inv.ParticipantID, &inv.RecipientUserID,
		&status, &inv.CreatedByParticipantID, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return storage.InviteRecord{}, storage.ErrNotFound
		}
		return storage.InviteRecord{}, err
	}
	inv.Status = storage.Status(status)
	inv.CreatedAt, _ = time.Parse(timeLayout, createdAt)
	inv.UpdatedAt, _ = time.Parse(timeLayout, updatedAt)
	return inv, nil
}

func scanInviteRow(rows *sql.Rows) (storage.InviteRecord, error) {
	var inv storage.InviteRecord
	var status, createdAt, updatedAt string
	err := rows.Scan(&inv.ID, &inv.CampaignID, &inv.ParticipantID, &inv.RecipientUserID,
		&status, &inv.CreatedByParticipantID, &createdAt, &updatedAt)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	inv.Status = storage.Status(status)
	inv.CreatedAt, _ = time.Parse(timeLayout, createdAt)
	inv.UpdatedAt, _ = time.Parse(timeLayout, updatedAt)
	return inv, nil
}

func collectInvitePage(rows *sql.Rows, pageSize int) (storage.InvitePage, error) {
	var page storage.InvitePage
	for rows.Next() {
		inv, err := scanInviteRow(rows)
		if err != nil {
			return storage.InvitePage{}, err
		}
		page.Invites = append(page.Invites, inv)
	}
	if err := rows.Err(); err != nil {
		return storage.InvitePage{}, err
	}
	if len(page.Invites) > pageSize {
		page.NextPageToken = page.Invites[pageSize-1].ID
		page.Invites = page.Invites[:pageSize]
	}
	return page, nil
}

// MarshalOutboxPayload marshals a payload to JSON for outbox events.
func MarshalOutboxPayload(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}
