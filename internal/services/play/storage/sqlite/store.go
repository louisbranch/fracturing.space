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

	msqlite "modernc.org/sqlite"
	sqlite3lib "modernc.org/sqlite/lib"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/play/storage/sqlite/migrations"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

const maxAppendRetries = 16

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
func (s *Store) LatestSequence(ctx context.Context, scope transcript.Scope) (int64, error) {
	if s == nil || s.sqlDB == nil {
		return 0, errors.New("store is required")
	}
	scope = scope.Normalize()
	if err := scope.Validate(); err != nil {
		return 0, err
	}
	var latest sql.NullInt64
	err := s.sqlDB.QueryRowContext(
		ctx,
		`SELECT MAX(sequence_id) FROM transcript_messages WHERE campaign_id = ? AND session_id = ?`,
		scope.CampaignID,
		scope.SessionID,
	).Scan(&latest)
	if err != nil {
		return 0, fmt.Errorf("query latest transcript sequence: %w", err)
	}
	if !latest.Valid {
		return 0, nil
	}
	return latest.Int64, nil
}

// AppendMessage stores one transcript message, preserving client-message
// idempotency and retrying sequence races caused by concurrent writers.
func (s *Store) AppendMessage(ctx context.Context, req transcript.AppendRequest) (transcript.AppendResult, error) {
	if s == nil || s.sqlDB == nil {
		return transcript.AppendResult{}, errors.New("store is required")
	}
	req = req.Normalize()
	if err := req.Validate(); err != nil {
		return transcript.AppendResult{}, err
	}

	var lastErr error
	for attempt := 1; attempt <= maxAppendRetries; attempt++ {
		result, retry, err := s.appendMessageAttempt(ctx, req)
		if err == nil {
			return result, nil
		}
		if !retry {
			return transcript.AppendResult{}, err
		}
		lastErr = err
	}
	return transcript.AppendResult{}, fmt.Errorf(
		"append transcript message: exhausted %d retries after concurrent write conflicts: %w",
		maxAppendRetries,
		lastErr,
	)
}

func (s *Store) appendMessageAttempt(ctx context.Context, req transcript.AppendRequest) (transcript.AppendResult, bool, error) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return transcript.AppendResult{}, false, fmt.Errorf("begin transcript tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if req.ClientMessageID != "" {
		existing, ok, err := lookupByClientMessageID(ctx, tx, req.Scope, req.ClientMessageID)
		if err != nil {
			return transcript.AppendResult{}, false, err
		}
		if ok {
			if err := tx.Commit(); err != nil {
				return transcript.AppendResult{}, false, fmt.Errorf("commit transcript duplicate lookup: %w", err)
			}
			return transcript.AppendResult{Message: existing, Duplicate: true}, false, nil
		}
	}

	message, err := s.buildMessage(ctx, tx, req)
	if err != nil {
		return transcript.AppendResult{}, false, err
	}
	if err := insertMessage(ctx, tx, message); err != nil {
		if isAppendRetryable(err) {
			return transcript.AppendResult{}, true, fmt.Errorf("retry transcript append after write conflict: %w", err)
		}
		return transcript.AppendResult{}, false, fmt.Errorf("insert transcript message: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return transcript.AppendResult{}, false, fmt.Errorf("commit transcript insert: %w", err)
	}
	return transcript.AppendResult{Message: message}, false, nil
}

func (s *Store) buildMessage(ctx context.Context, tx *sql.Tx, req transcript.AppendRequest) (transcript.Message, error) {
	nextSequence, err := nextSequenceID(ctx, tx, req.Scope)
	if err != nil {
		return transcript.Message{}, err
	}
	actor := req.Actor.Normalize()
	if actor.ParticipantID == "" {
		actor.ParticipantID = "participant"
	}
	if actor.Name == "" {
		actor.Name = actor.ParticipantID
	}
	sentAt := s.now().UTC()
	return transcript.Message{
		MessageID:       fmt.Sprintf("playmsg_%s_%s_%d", req.Scope.CampaignID, req.Scope.SessionID, nextSequence),
		CampaignID:      req.Scope.CampaignID,
		SessionID:       req.Scope.SessionID,
		SequenceID:      nextSequence,
		SentAt:          sentAt.Format(time.RFC3339Nano),
		Actor:           actor,
		Body:            req.Body,
		ClientMessageID: req.ClientMessageID,
	}, nil
}

// HistoryAfter returns messages strictly after the given sequence in ascending order.
func (s *Store) HistoryAfter(ctx context.Context, query transcript.HistoryAfterQuery) ([]transcript.Message, error) {
	if s == nil || s.sqlDB == nil {
		return nil, errors.New("store is required")
	}
	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return nil, err
	}
	return s.queryHistory(
		ctx,
		`SELECT campaign_id, session_id, sequence_id, message_id, sent_at_utc, participant_id, participant_name, body, COALESCE(client_message_id, '')
		 FROM transcript_messages
		 WHERE campaign_id = ? AND session_id = ? AND sequence_id > ?
		 ORDER BY sequence_id ASC`,
		query.Scope.CampaignID,
		query.Scope.SessionID,
		query.AfterSequenceID,
	)
}

// HistoryBefore returns up to limit older messages, ordered ascending.
func (s *Store) HistoryBefore(ctx context.Context, query transcript.HistoryBeforeQuery) ([]transcript.Message, error) {
	if s == nil || s.sqlDB == nil {
		return nil, errors.New("store is required")
	}
	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return nil, err
	}
	rows, err := s.sqlDB.QueryContext(
		ctx,
		`SELECT campaign_id, session_id, sequence_id, message_id, sent_at_utc, participant_id, participant_name, body, COALESCE(client_message_id, '')
		 FROM transcript_messages
		 WHERE campaign_id = ? AND session_id = ? AND sequence_id < ?
		 ORDER BY sequence_id DESC
		 LIMIT ?`,
		query.Scope.CampaignID,
		query.Scope.SessionID,
		query.BeforeSequenceID,
		query.Limit,
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

func lookupByClientMessageID(ctx context.Context, tx *sql.Tx, scope transcript.Scope, clientMessageID string) (transcript.Message, bool, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT campaign_id, session_id, sequence_id, message_id, sent_at_utc, participant_id, participant_name, body, COALESCE(client_message_id, '')
		 FROM transcript_messages
		 WHERE campaign_id = ? AND session_id = ? AND client_message_id = ?`,
		scope.CampaignID,
		scope.SessionID,
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

func nextSequenceID(ctx context.Context, tx *sql.Tx, scope transcript.Scope) (int64, error) {
	var nextSequence int64
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(sequence_id), 0) + 1 FROM transcript_messages WHERE campaign_id = ? AND session_id = ?`,
		scope.CampaignID,
		scope.SessionID,
	).Scan(&nextSequence); err != nil {
		return 0, fmt.Errorf("query next transcript sequence: %w", err)
	}
	return nextSequence, nil
}

func insertMessage(ctx context.Context, tx *sql.Tx, message transcript.Message) error {
	var nullableClientID any
	if message.ClientMessageID != "" {
		nullableClientID = message.ClientMessageID
	}
	_, err := tx.ExecContext(
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
	)
	return err
}

func isAppendRetryable(err error) bool {
	return isBusyOrLockedError(err) || isUniqueConstraintError(err)
}

func isBusyOrLockedError(err error) bool {
	if sqliteconn.IsBusyOrLockedError(err) {
		return true
	}
	var sqliteErr *msqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xff {
		case sqlite3lib.SQLITE_BUSY, sqlite3lib.SQLITE_LOCKED:
			return true
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), "database is locked")
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	var sqliteErr *msqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xff {
		case sqlite3lib.SQLITE_CONSTRAINT, sqlite3lib.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3lib.SQLITE_CONSTRAINT_UNIQUE:
			return true
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

func (s *Store) queryHistory(ctx context.Context, query string, args ...any) ([]transcript.Message, error) {
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
