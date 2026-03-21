package integrationoutbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	gameintegration "github.com/louisbranch/fracturing.space/internal/services/game/integration"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type integrationOutboxScanner func(dest ...any) error

type integrationOutboxExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Store binds integration-outbox persistence to a SQLite connection.
type Store struct {
	sqlDB *sql.DB
}

var _ storage.IntegrationOutboxStore = (*Store)(nil)

// Bind creates an integration-outbox backend bound to the provided SQLite DB.
func Bind(sqlDB *sql.DB) *Store {
	if sqlDB == nil {
		return nil
	}
	return &Store{sqlDB: sqlDB}
}

// EnqueueForEvent derives and inserts integration outbox rows for one event
// inside the caller's transaction.
func EnqueueForEvent(ctx context.Context, tx *sql.Tx, evt event.Event) error {
	outboxEvents, err := integrationOutboxEventsForEvent(evt)
	if err != nil {
		return err
	}
	for _, outboxEvent := range outboxEvents {
		if err := enqueue(ctx, tx, outboxEvent); err != nil {
			return err
		}
	}
	return nil
}

func enqueue(ctx context.Context, exec integrationOutboxExec, outboxEvent storage.IntegrationOutboxEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if exec == nil {
		return fmt.Errorf("integration outbox executor is required")
	}

	outboxEvent.ID = strings.TrimSpace(outboxEvent.ID)
	outboxEvent.EventType = strings.TrimSpace(outboxEvent.EventType)
	outboxEvent.DedupeKey = strings.TrimSpace(outboxEvent.DedupeKey)
	if outboxEvent.ID == "" {
		return fmt.Errorf("integration outbox event id is required")
	}
	if outboxEvent.EventType == "" {
		return fmt.Errorf("integration outbox event type is required")
	}
	if outboxEvent.Status == "" {
		outboxEvent.Status = storage.IntegrationOutboxStatusPending
	}
	if outboxEvent.NextAttemptAt.IsZero() {
		outboxEvent.NextAttemptAt = time.Now().UTC()
	}
	if outboxEvent.CreatedAt.IsZero() {
		outboxEvent.CreatedAt = outboxEvent.NextAttemptAt
	}
	if outboxEvent.UpdatedAt.IsZero() {
		outboxEvent.UpdatedAt = outboxEvent.CreatedAt
	}

	_, err := exec.ExecContext(ctx, `
INSERT INTO game_integration_outbox (
	id,
	event_type,
	payload_json,
	dedupe_key,
	status,
	attempt_count,
	next_attempt_at,
	lease_owner,
	lease_expires_at,
	last_error,
	processed_at,
	created_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, '', NULL, '', NULL, ?, ?)
ON CONFLICT(dedupe_key) DO NOTHING
`,
		outboxEvent.ID,
		outboxEvent.EventType,
		outboxEvent.PayloadJSON,
		outboxEvent.DedupeKey,
		outboxEvent.Status,
		outboxEvent.AttemptCount,
		sqliteutil.ToMillis(outboxEvent.NextAttemptAt.UTC()),
		sqliteutil.ToMillis(outboxEvent.CreatedAt.UTC()),
		sqliteutil.ToMillis(outboxEvent.UpdatedAt.UTC()),
	)
	if err != nil {
		return fmt.Errorf("enqueue integration outbox event: %w", err)
	}
	return nil
}

func integrationOutboxEventsForEvent(evt event.Event) ([]storage.IntegrationOutboxEvent, error) {
	switch evt.Type {
	case invite.EventTypeCreated:
		return buildInviteCreatedOutboxEvent(evt)
	case invite.EventTypeClaimed:
		return buildInviteClaimedOutboxEvent(evt)
	case invite.EventTypeDeclined:
		return buildInviteDeclinedOutboxEvent(evt)
	case session.EventTypeGMAuthoritySet:
		return buildAIGMTurnRequestedOutboxEvent(evt)
	case session.EventTypeOOCClosed:
		return buildAIGMTurnRequestedOutboxEvent(evt)
	case scene.EventTypePlayerPhaseReviewStarted:
		return buildAIGMTurnRequestedOutboxEvent(evt)
	default:
		return nil, nil
	}
}

func buildInviteCreatedOutboxEvent(evt event.Event) ([]storage.IntegrationOutboxEvent, error) {
	var payload invite.CreatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("decode invite.created integration payload: %w", err)
	}
	recipientUserID := strings.TrimSpace(string(payload.RecipientUserID))
	if recipientUserID == "" {
		return nil, nil
	}
	outboxEvent, err := newInviteNotificationOutboxEvent(
		evt,
		gameintegration.InviteNotificationCreatedOutboxEventType,
		gameintegration.InviteCreatedNotificationDedupeKey(string(payload.InviteID)),
		recipientUserID,
		"created",
	)
	if err != nil {
		return nil, err
	}
	return []storage.IntegrationOutboxEvent{outboxEvent}, nil
}

func buildInviteClaimedOutboxEvent(evt event.Event) ([]storage.IntegrationOutboxEvent, error) {
	var payload invite.ClaimPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("decode invite.claimed integration payload: %w", err)
	}
	outboxEvent, err := newInviteNotificationOutboxEvent(
		evt,
		gameintegration.InviteNotificationClaimedOutboxEventType,
		gameintegration.InviteAcceptedNotificationDedupeKey(string(payload.InviteID)),
		strings.TrimSpace(string(payload.UserID)),
		"accepted",
	)
	if err != nil {
		return nil, err
	}
	return []storage.IntegrationOutboxEvent{outboxEvent}, nil
}

func buildInviteDeclinedOutboxEvent(evt event.Event) ([]storage.IntegrationOutboxEvent, error) {
	var payload invite.DeclinePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("decode invite.declined integration payload: %w", err)
	}
	outboxEvent, err := newInviteNotificationOutboxEvent(
		evt,
		gameintegration.InviteNotificationDeclinedOutboxEventType,
		gameintegration.InviteDeclinedNotificationDedupeKey(string(payload.InviteID)),
		strings.TrimSpace(string(payload.UserID)),
		"declined",
	)
	if err != nil {
		return nil, err
	}
	return []storage.IntegrationOutboxEvent{outboxEvent}, nil
}

func newInviteNotificationOutboxEvent(
	evt event.Event,
	eventType string,
	dedupeKey string,
	recipientUserID string,
	notificationKind string,
) (storage.IntegrationOutboxEvent, error) {
	outboxEventID, err := id.NewID()
	if err != nil {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("generate integration outbox event id: %w", err)
	}
	payloadJSON, err := json.Marshal(gameintegration.InviteNotificationOutboxPayload{
		InviteID:         strings.TrimSpace(evt.EntityID),
		CampaignID:       strings.TrimSpace(string(evt.CampaignID)),
		RecipientUserID:  strings.TrimSpace(recipientUserID),
		NotificationKind: notificationKind,
	})
	if err != nil {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("marshal invite notification outbox payload: %w", err)
	}
	now := evt.Timestamp.UTC()
	return storage.IntegrationOutboxEvent{
		ID:            outboxEventID,
		EventType:     eventType,
		PayloadJSON:   string(payloadJSON),
		DedupeKey:     dedupeKey,
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func buildAIGMTurnRequestedOutboxEvent(evt event.Event) ([]storage.IntegrationOutboxEvent, error) {
	payload := gameintegration.AIGMTurnRequestedOutboxPayload{
		CampaignID:      strings.TrimSpace(string(evt.CampaignID)),
		SessionID:       strings.TrimSpace(evt.SessionID.String()),
		SourceEventType: strings.TrimSpace(string(evt.Type)),
	}
	switch evt.Type {
	case scene.EventTypePlayerPhaseReviewStarted:
		var source scene.PlayerPhaseReviewStartedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &source); err != nil {
			return nil, fmt.Errorf("decode scene.player_phase_review_started integration payload: %w", err)
		}
		payload.SourceSceneID = strings.TrimSpace(source.SceneID.String())
		payload.SourcePhaseID = strings.TrimSpace(source.PhaseID)
	}
	if strings.TrimSpace(payload.CampaignID) == "" || strings.TrimSpace(payload.SessionID) == "" {
		return nil, nil
	}
	outboxEventID, err := id.NewID()
	if err != nil {
		return nil, fmt.Errorf("generate integration outbox event id: %w", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal ai gm turn requested outbox payload: %w", err)
	}
	now := evt.Timestamp.UTC()
	return []storage.IntegrationOutboxEvent{{
		ID:            outboxEventID,
		EventType:     gameintegration.AIGMTurnRequestedOutboxEventType,
		PayloadJSON:   string(payloadJSON),
		DedupeKey:     gameintegration.AIGMTurnRequestedDedupeKey(outboxEventID),
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}}, nil
}

// EnqueueIntegrationOutboxEvent persists one outbox event directly.
func (s *Store) EnqueueIntegrationOutboxEvent(ctx context.Context, outboxEvent storage.IntegrationOutboxEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	return enqueue(ctx, s.sqlDB, outboxEvent)
}

// GetIntegrationOutboxEvent loads one outbox row by id.
func (s *Store) GetIntegrationOutboxEvent(ctx context.Context, id string) (storage.IntegrationOutboxEvent, error) {
	if err := ctx.Err(); err != nil {
		return storage.IntegrationOutboxEvent{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("event id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT
	id,
	event_type,
	payload_json,
	dedupe_key,
	status,
	attempt_count,
	next_attempt_at,
	lease_owner,
	lease_expires_at,
	last_error,
	processed_at,
	created_at,
	updated_at
FROM game_integration_outbox
WHERE id = ?
`, id)
	outboxEvent, err := scanIntegrationOutboxEvent(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.IntegrationOutboxEvent{}, storage.ErrNotFound
		}
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("get integration outbox event: %w", err)
	}
	return outboxEvent, nil
}

func scanIntegrationOutboxEvent(scan integrationOutboxScanner) (storage.IntegrationOutboxEvent, error) {
	var (
		outboxEvent      storage.IntegrationOutboxEvent
		nextAttemptAtMS  int64
		leaseExpiresAtMS sql.NullInt64
		processedAtMS    sql.NullInt64
		createdAtMS      int64
		updatedAtMS      int64
	)
	if err := scan(
		&outboxEvent.ID,
		&outboxEvent.EventType,
		&outboxEvent.PayloadJSON,
		&outboxEvent.DedupeKey,
		&outboxEvent.Status,
		&outboxEvent.AttemptCount,
		&nextAttemptAtMS,
		&outboxEvent.LeaseOwner,
		&leaseExpiresAtMS,
		&outboxEvent.LastError,
		&processedAtMS,
		&createdAtMS,
		&updatedAtMS,
	); err != nil {
		return storage.IntegrationOutboxEvent{}, err
	}
	outboxEvent.NextAttemptAt = sqliteutil.FromMillis(nextAttemptAtMS)
	outboxEvent.LeaseExpiresAt = sqliteutil.FromNullMillis(leaseExpiresAtMS)
	outboxEvent.ProcessedAt = sqliteutil.FromNullMillis(processedAtMS)
	outboxEvent.CreatedAt = sqliteutil.FromMillis(createdAtMS)
	outboxEvent.UpdatedAt = sqliteutil.FromMillis(updatedAtMS)
	return outboxEvent, nil
}

// LeaseIntegrationOutboxEvents claims due outbox rows for one consumer.
func (s *Store) LeaseIntegrationOutboxEvents(ctx context.Context, consumer string, limit int, now time.Time, leaseTTL time.Duration) ([]storage.IntegrationOutboxEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	consumer = strings.TrimSpace(consumer)
	if consumer == "" {
		return nil, fmt.Errorf("consumer is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}
	if leaseTTL <= 0 {
		return nil, fmt.Errorf("lease ttl must be greater than zero")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	leaseExpiresAt := now.Add(leaseTTL)

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("start lease transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	rows, err := tx.QueryContext(ctx, `
SELECT id
FROM game_integration_outbox
WHERE (
	(status = ? AND next_attempt_at <= ?)
	OR
	(status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
)
ORDER BY next_attempt_at ASC, created_at ASC, id ASC
LIMIT ?
`,
		storage.IntegrationOutboxStatusPending,
		sqliteutil.ToMillis(now),
		storage.IntegrationOutboxStatusLeased,
		sqliteutil.ToMillis(now),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("select lease candidates: %w", err)
	}
	candidateIDs := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan lease candidate: %w", scanErr)
		}
		candidateIDs = append(candidateIDs, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate lease candidates: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close lease candidates: %w", err)
	}
	if len(candidateIDs) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit empty lease transaction: %w", err)
		}
		return []storage.IntegrationOutboxEvent{}, nil
	}

	leased := make([]storage.IntegrationOutboxEvent, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		result, updateErr := tx.ExecContext(ctx, `
UPDATE game_integration_outbox
SET
	status = ?,
	lease_owner = ?,
	lease_expires_at = ?,
	updated_at = ?
WHERE id = ?
AND (
	(status = ? AND next_attempt_at <= ?)
	OR
	(status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
)
`,
			storage.IntegrationOutboxStatusLeased,
			consumer,
			sqliteutil.ToMillis(leaseExpiresAt),
			sqliteutil.ToMillis(now),
			id,
			storage.IntegrationOutboxStatusPending,
			sqliteutil.ToMillis(now),
			storage.IntegrationOutboxStatusLeased,
			sqliteutil.ToMillis(now),
		)
		if updateErr != nil {
			return nil, fmt.Errorf("lease integration outbox event %s: %w", id, updateErr)
		}
		rowsAffected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return nil, fmt.Errorf("lease rows affected for %s: %w", id, rowsErr)
		}
		if rowsAffected == 0 {
			continue
		}

		row := tx.QueryRowContext(ctx, `
SELECT
	id,
	event_type,
	payload_json,
	dedupe_key,
	status,
	attempt_count,
	next_attempt_at,
	lease_owner,
	lease_expires_at,
	last_error,
	processed_at,
	created_at,
	updated_at
FROM game_integration_outbox
WHERE id = ?
`, id)
		outboxEvent, scanErr := scanIntegrationOutboxEvent(row.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan leased integration outbox event %s: %w", id, scanErr)
		}
		leased = append(leased, outboxEvent)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit lease transaction: %w", err)
	}
	return leased, nil
}

// MarkIntegrationOutboxSucceeded marks one leased row as processed.
func (s *Store) MarkIntegrationOutboxSucceeded(ctx context.Context, id string, consumer string, processedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	consumer = strings.TrimSpace(consumer)
	if id == "" {
		return fmt.Errorf("event id is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if processedAt.IsZero() {
		processedAt = time.Now().UTC()
	}
	processedAt = processedAt.UTC()

	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE game_integration_outbox
SET
	status = ?,
	lease_owner = '',
	lease_expires_at = NULL,
	last_error = '',
	processed_at = ?,
	updated_at = ?
WHERE id = ?
AND status = ?
AND lease_owner = ?
`,
		storage.IntegrationOutboxStatusSucceeded,
		sqliteutil.ToMillis(processedAt),
		sqliteutil.ToMillis(processedAt),
		id,
		storage.IntegrationOutboxStatusLeased,
		consumer,
	)
	if err != nil {
		return fmt.Errorf("mark integration outbox succeeded: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark integration outbox succeeded rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// MarkIntegrationOutboxRetry releases a leased row for a future retry.
func (s *Store) MarkIntegrationOutboxRetry(ctx context.Context, id string, consumer string, nextAttemptAt time.Time, lastError string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	consumer = strings.TrimSpace(consumer)
	if id == "" {
		return fmt.Errorf("event id is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if nextAttemptAt.IsZero() {
		return fmt.Errorf("next attempt at is required")
	}
	now := time.Now().UTC()

	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE game_integration_outbox
SET
	status = ?,
	attempt_count = attempt_count + 1,
	next_attempt_at = ?,
	lease_owner = '',
	lease_expires_at = NULL,
	last_error = ?,
	processed_at = NULL,
	updated_at = ?
WHERE id = ?
AND status = ?
AND lease_owner = ?
`,
		storage.IntegrationOutboxStatusPending,
		sqliteutil.ToMillis(nextAttemptAt.UTC()),
		strings.TrimSpace(lastError),
		sqliteutil.ToMillis(now),
		id,
		storage.IntegrationOutboxStatusLeased,
		consumer,
	)
	if err != nil {
		return fmt.Errorf("mark integration outbox retry: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark integration outbox retry rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// MarkIntegrationOutboxDead marks a leased row permanently failed.
func (s *Store) MarkIntegrationOutboxDead(ctx context.Context, id string, consumer string, lastError string, processedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	id = strings.TrimSpace(id)
	consumer = strings.TrimSpace(consumer)
	if id == "" {
		return fmt.Errorf("event id is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if processedAt.IsZero() {
		processedAt = time.Now().UTC()
	}
	processedAt = processedAt.UTC()

	result, err := s.sqlDB.ExecContext(ctx, `
UPDATE game_integration_outbox
SET
	status = ?,
	attempt_count = attempt_count + 1,
	lease_owner = '',
	lease_expires_at = NULL,
	last_error = ?,
	processed_at = ?,
	updated_at = ?
WHERE id = ?
AND status = ?
AND lease_owner = ?
`,
		storage.IntegrationOutboxStatusDead,
		strings.TrimSpace(lastError),
		sqliteutil.ToMillis(processedAt),
		sqliteutil.ToMillis(processedAt),
		id,
		storage.IntegrationOutboxStatusLeased,
		consumer,
	)
	if err != nil {
		return fmt.Errorf("mark integration outbox dead: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark integration outbox dead rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}
