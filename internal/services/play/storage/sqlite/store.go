package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/play/storage/sqlite/migrations"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

// Store owns transcript persistence for the play service.
type Store struct {
	sqlDB *sql.DB
	now   func() time.Time
}

// Open opens the play transcript store.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("storage path is required")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite store: %w", err)
	}
	if err := sqlitemigrate.ApplyMigrations(sqlDB, migrations.FS, "", time.Now); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("apply play sqlite migrations: %w", err)
	}
	return &Store{sqlDB: sqlDB, now: time.Now}, nil
}

// Close closes the underlying sqlite handle.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// LatestSequence returns the latest stored sequence for one campaign/session.
func (s *Store) LatestSequence(ctx context.Context, campaignID string, sessionID string) (int64, error) {
	if s == nil || s.sqlDB == nil {
		return 0, errors.New("store is required")
	}
	var latest sql.NullInt64
	err := s.sqlDB.QueryRowContext(
		ctx,
		`SELECT MAX(sequence_id) FROM transcript_messages WHERE campaign_id = ? AND session_id = ?`,
		strings.TrimSpace(campaignID),
		strings.TrimSpace(sessionID),
	).Scan(&latest)
	if err != nil {
		return 0, fmt.Errorf("query latest transcript sequence: %w", err)
	}
	if !latest.Valid {
		return 0, nil
	}
	return latest.Int64, nil
}

// AppendMessage stores one transcript message, preserving client-message idempotency.
func (s *Store) AppendMessage(
	ctx context.Context,
	campaignID string,
	sessionID string,
	actor transcript.MessageActor,
	body string,
	clientMessageID string,
) (transcript.Message, bool, error) {
	if s == nil || s.sqlDB == nil {
		return transcript.Message{}, false, errors.New("store is required")
	}
	campaignID = strings.TrimSpace(campaignID)
	sessionID = strings.TrimSpace(sessionID)
	actor.ParticipantID = strings.TrimSpace(actor.ParticipantID)
	actor.Name = strings.TrimSpace(actor.Name)
	body = strings.TrimSpace(body)
	clientMessageID = strings.TrimSpace(clientMessageID)
	sentAt := s.now().UTC()

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return transcript.Message{}, false, fmt.Errorf("begin transcript tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if clientMessageID != "" {
		existing, ok, err := lookupByClientMessageID(ctx, tx, campaignID, sessionID, clientMessageID)
		if err != nil {
			return transcript.Message{}, false, err
		}
		if ok {
			if err := tx.Commit(); err != nil {
				return transcript.Message{}, false, fmt.Errorf("commit transcript duplicate lookup: %w", err)
			}
			return existing, true, nil
		}
	}

	var nextSequence int64
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(sequence_id), 0) + 1 FROM transcript_messages WHERE campaign_id = ? AND session_id = ?`,
		campaignID,
		sessionID,
	).Scan(&nextSequence); err != nil {
		return transcript.Message{}, false, fmt.Errorf("query next transcript sequence: %w", err)
	}

	if actor.ParticipantID == "" {
		actor.ParticipantID = "participant"
	}
	if actor.Name == "" {
		actor.Name = actor.ParticipantID
	}
	message := transcript.Message{
		MessageID:       fmt.Sprintf("playmsg_%s_%s_%d", campaignID, sessionID, nextSequence),
		CampaignID:      campaignID,
		SessionID:       sessionID,
		SequenceID:      nextSequence,
		SentAt:          sentAt.Format(time.RFC3339Nano),
		Actor:           actor,
		Body:            body,
		ClientMessageID: clientMessageID,
	}
	var nullableClientID any
	if clientMessageID != "" {
		nullableClientID = clientMessageID
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO transcript_messages (
			campaign_id, session_id, sequence_id, message_id, sent_at_utc,
			participant_id, participant_name, body, client_message_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		message.CampaignID,
		message.SessionID,
		message.SequenceID,
		message.MessageID,
		message.SentAt,
		message.Actor.ParticipantID,
		message.Actor.Name,
		message.Body,
		nullableClientID,
	); err != nil {
		return transcript.Message{}, false, fmt.Errorf("insert transcript message: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return transcript.Message{}, false, fmt.Errorf("commit transcript insert: %w", err)
	}
	return message, false, nil
}

// HistoryAfter returns messages strictly after the given sequence in ascending order.
func (s *Store) HistoryAfter(ctx context.Context, campaignID string, sessionID string, afterSequenceID int64) ([]transcript.Message, error) {
	return s.queryHistory(
		ctx,
		`SELECT campaign_id, session_id, sequence_id, message_id, sent_at_utc, participant_id, participant_name, body, COALESCE(client_message_id, '')
		 FROM transcript_messages
		 WHERE campaign_id = ? AND session_id = ? AND sequence_id > ?
		 ORDER BY sequence_id ASC`,
		strings.TrimSpace(campaignID),
		strings.TrimSpace(sessionID),
		afterSequenceID,
	)
}

// HistoryBefore returns up to limit older messages, ordered ascending.
func (s *Store) HistoryBefore(ctx context.Context, campaignID string, sessionID string, beforeSequenceID int64, limit int) ([]transcript.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.sqlDB.QueryContext(
		ctx,
		`SELECT campaign_id, session_id, sequence_id, message_id, sent_at_utc, participant_id, participant_name, body, COALESCE(client_message_id, '')
		 FROM transcript_messages
		 WHERE campaign_id = ? AND session_id = ? AND sequence_id < ?
		 ORDER BY sequence_id DESC
		 LIMIT ?`,
		strings.TrimSpace(campaignID),
		strings.TrimSpace(sessionID),
		beforeSequenceID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query transcript history before: %w", err)
	}
	defer rows.Close()

	values, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
	return values, nil
}

func lookupByClientMessageID(ctx context.Context, tx *sql.Tx, campaignID string, sessionID string, clientMessageID string) (transcript.Message, bool, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT campaign_id, session_id, sequence_id, message_id, sent_at_utc, participant_id, participant_name, body, COALESCE(client_message_id, '')
		 FROM transcript_messages
		 WHERE campaign_id = ? AND session_id = ? AND client_message_id = ?`,
		campaignID,
		sessionID,
		clientMessageID,
	)
	value, err := scanMessage(row)
	if errors.Is(err, sql.ErrNoRows) {
		return transcript.Message{}, false, nil
	}
	if err != nil {
		return transcript.Message{}, false, fmt.Errorf("lookup transcript duplicate: %w", err)
	}
	return value, true, nil
}

func (s *Store) queryHistory(ctx context.Context, query string, args ...any) ([]transcript.Message, error) {
	if s == nil || s.sqlDB == nil {
		return nil, errors.New("store is required")
	}
	rows, err := s.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query transcript history: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMessages(rows *sql.Rows) ([]transcript.Message, error) {
	values := []transcript.Message{}
	for rows.Next() {
		value, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transcript rows: %w", err)
	}
	return values, nil
}

func scanMessage(scanner rowScanner) (transcript.Message, error) {
	var value transcript.Message
	if err := scanner.Scan(
		&value.CampaignID,
		&value.SessionID,
		&value.SequenceID,
		&value.MessageID,
		&value.SentAt,
		&value.Actor.ParticipantID,
		&value.Actor.Name,
		&value.Body,
		&value.ClientMessageID,
	); err != nil {
		return transcript.Message{}, err
	}
	return value, nil
}
