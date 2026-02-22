package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// EventStore methods (unified event journal)

// AppendEvent atomically appends an event and returns it with sequence and hash set.
func (s *Store) AppendEvent(ctx context.Context, evt event.Event) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}
	if s.eventRegistry == nil {
		return event.Event{}, fmt.Errorf("event registry is required")
	}

	validated, err := s.eventRegistry.ValidateForAppend(evt)
	if err != nil {
		return event.Event{}, err
	}
	evt = validated

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return event.Event{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	evt.Timestamp = evt.Timestamp.UTC().Truncate(time.Millisecond)

	if err := qtx.InitEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("init event seq: %w", err)
	}

	seq, err := qtx.GetEventSeq(ctx, evt.CampaignID)
	if err != nil {
		return event.Event{}, fmt.Errorf("get event seq: %w", err)
	}
	evt.Seq = uint64(seq)

	if err := qtx.IncrementEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("increment event seq: %w", err)
	}

	if s.keyring == nil {
		return event.Event{}, fmt.Errorf("event integrity keyring is required")
	}

	hash, err := integrity.EventHash(evt)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute event hash: %w", err)
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}
	evt.Hash = hash

	prevHash := ""
	if evt.Seq > 1 {
		prevRow, err := qtx.GetEventBySeq(ctx, db.GetEventBySeqParams{
			CampaignID: evt.CampaignID,
			Seq:        int64(evt.Seq - 1),
		})
		if err != nil {
			return event.Event{}, fmt.Errorf("load previous event: %w", err)
		}
		prevHash = prevRow.ChainHash
	}

	chainHash, err := integrity.ChainHash(evt, prevHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute chain hash: %w", err)
	}
	if strings.TrimSpace(chainHash) == "" {
		return event.Event{}, fmt.Errorf("chain hash is required")
	}

	signature, keyID, err := s.keyring.SignChainHash(evt.CampaignID, chainHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("sign chain hash: %w", err)
	}

	evt.PrevHash = prevHash
	evt.ChainHash = chainHash
	evt.Signature = signature
	evt.SignatureKeyID = keyID

	if err := qtx.AppendEvent(ctx, db.AppendEventParams{
		CampaignID:     evt.CampaignID,
		Seq:            int64(evt.Seq),
		EventHash:      evt.Hash,
		PrevEventHash:  prevHash,
		ChainHash:      chainHash,
		SignatureKeyID: keyID,
		EventSignature: signature,
		Timestamp:      toMillis(evt.Timestamp),
		EventType:      string(evt.Type),
		SessionID:      evt.SessionID,
		RequestID:      evt.RequestID,
		InvocationID:   evt.InvocationID,
		ActorType:      string(evt.ActorType),
		ActorID:        evt.ActorID,
		EntityType:     evt.EntityType,
		EntityID:       evt.EntityID,
		SystemID:       evt.SystemID,
		SystemVersion:  evt.SystemVersion,
		CorrelationID:  evt.CorrelationID,
		CausationID:    evt.CausationID,
		PayloadJson:    evt.PayloadJSON,
	}); err != nil {
		if isConstraintError(err) {
			stored, lookupErr := s.GetEventByHash(ctx, evt.Hash)
			if lookupErr == nil {
				return stored, nil
			}
		}
		return event.Event{}, fmt.Errorf("append event: %w", err)
	}
	if err := s.enqueueProjectionApplyOutbox(ctx, tx, evt); err != nil {
		return event.Event{}, err
	}

	if err := tx.Commit(); err != nil {
		return event.Event{}, fmt.Errorf("commit: %w", err)
	}

	return evt, nil
}

// BatchAppendEvents atomically appends multiple events in a single transaction.
//
// All events must belong to the same campaign. Sequence numbers are allocated
// contiguously, and chain hashes link each event to its predecessor — including
// the last previously stored event for the first item in the batch.
func (s *Store) BatchAppendEvents(ctx context.Context, events []event.Event) ([]event.Event, error) {
	if len(events) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if s.eventRegistry == nil {
		return nil, fmt.Errorf("event registry is required")
	}
	if s.keyring == nil {
		return nil, fmt.Errorf("event integrity keyring is required")
	}

	// Validate all events before opening a transaction.
	validated := make([]event.Event, len(events))
	for i, evt := range events {
		v, err := s.eventRegistry.ValidateForAppend(evt)
		if err != nil {
			return nil, fmt.Errorf("event %d: %w", i, err)
		}
		if v.Timestamp.IsZero() {
			v.Timestamp = time.Now().UTC()
		}
		v.Timestamp = v.Timestamp.UTC().Truncate(time.Millisecond)
		validated[i] = v
	}

	campaignID := validated[0].CampaignID

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if err := qtx.InitEventSeq(ctx, campaignID); err != nil {
		return nil, fmt.Errorf("init event seq: %w", err)
	}

	baseSeq, err := qtx.GetEventSeq(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get event seq: %w", err)
	}

	// Load previous chain hash for linking the first event in the batch.
	prevChainHash := ""
	if baseSeq > 1 {
		prevRow, err := qtx.GetEventBySeq(ctx, db.GetEventBySeqParams{
			CampaignID: campaignID,
			Seq:        baseSeq - 1,
		})
		if err != nil {
			return nil, fmt.Errorf("load previous event: %w", err)
		}
		prevChainHash = prevRow.ChainHash
	}

	stored := make([]event.Event, len(validated))
	for i, evt := range validated {
		evt.Seq = uint64(baseSeq) + uint64(i)

		hash, err := integrity.EventHash(evt)
		if err != nil {
			return nil, fmt.Errorf("event %d hash: %w", i, err)
		}
		if strings.TrimSpace(hash) == "" {
			return nil, fmt.Errorf("event %d: hash is empty", i)
		}
		evt.Hash = hash

		chainHash, err := integrity.ChainHash(evt, prevChainHash)
		if err != nil {
			return nil, fmt.Errorf("event %d chain hash: %w", i, err)
		}
		if strings.TrimSpace(chainHash) == "" {
			return nil, fmt.Errorf("event %d: chain hash is empty", i)
		}

		signature, keyID, err := s.keyring.SignChainHash(evt.CampaignID, chainHash)
		if err != nil {
			return nil, fmt.Errorf("event %d sign: %w", i, err)
		}

		evt.PrevHash = prevChainHash
		evt.ChainHash = chainHash
		evt.Signature = signature
		evt.SignatureKeyID = keyID

		if err := qtx.AppendEvent(ctx, db.AppendEventParams{
			CampaignID:     evt.CampaignID,
			Seq:            int64(evt.Seq),
			EventHash:      evt.Hash,
			PrevEventHash:  prevChainHash,
			ChainHash:      chainHash,
			SignatureKeyID: keyID,
			EventSignature: signature,
			Timestamp:      toMillis(evt.Timestamp),
			EventType:      string(evt.Type),
			SessionID:      evt.SessionID,
			RequestID:      evt.RequestID,
			InvocationID:   evt.InvocationID,
			ActorType:      string(evt.ActorType),
			ActorID:        evt.ActorID,
			EntityType:     evt.EntityType,
			EntityID:       evt.EntityID,
			SystemID:       evt.SystemID,
			SystemVersion:  evt.SystemVersion,
			CorrelationID:  evt.CorrelationID,
			CausationID:    evt.CausationID,
			PayloadJson:    evt.PayloadJSON,
		}); err != nil {
			return nil, fmt.Errorf("append event %d: %w", i, err)
		}

		if err := s.enqueueProjectionApplyOutbox(ctx, tx, evt); err != nil {
			return nil, err
		}

		prevChainHash = chainHash
		stored[i] = evt
	}

	// Advance the sequence counter to account for all appended events.
	nextSeq := int64(baseSeq) + int64(len(events))
	if _, err := tx.ExecContext(ctx,
		"UPDATE event_seq SET next_seq = ? WHERE campaign_id = ?",
		nextSeq, campaignID,
	); err != nil {
		return nil, fmt.Errorf("update event seq: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return stored, nil
}

func (s *Store) VerifyEventIntegrity(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if s.keyring == nil {
		return fmt.Errorf("event integrity keyring is required")
	}

	campaignIDs, err := s.listEventCampaignIDs(ctx)
	if err != nil {
		return err
	}
	for _, campaignID := range campaignIDs {
		if err := s.verifyCampaignEvents(ctx, campaignID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) listEventCampaignIDs(ctx context.Context) ([]string, error) {
	rows, err := s.sqlDB.QueryContext(ctx, "SELECT DISTINCT campaign_id FROM events ORDER BY campaign_id")
	if err != nil {
		return nil, fmt.Errorf("list campaign ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan campaign id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate campaign ids: %w", err)
	}
	return ids, nil
}

func (s *Store) verifyCampaignEvents(ctx context.Context, campaignID string) error {
	var lastSeq uint64
	prevChainHash := ""
	for {
		events, err := s.ListEvents(ctx, campaignID, lastSeq, 200)
		if err != nil {
			return fmt.Errorf("list events campaign_id=%s: %w", campaignID, err)
		}
		if len(events) == 0 {
			return nil
		}
		for _, evt := range events {
			if evt.Seq != lastSeq+1 {
				return fmt.Errorf("event sequence gap campaign_id=%s expected=%d got=%d", campaignID, lastSeq+1, evt.Seq)
			}
			if evt.Seq == 1 && evt.PrevHash != "" {
				return fmt.Errorf("first event prev hash must be empty campaign_id=%s", campaignID)
			}
			if evt.Seq > 1 && evt.PrevHash != prevChainHash {
				return fmt.Errorf("prev hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			hash, err := integrity.EventHash(evt)
			if err != nil {
				return fmt.Errorf("compute event hash campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}
			if hash != evt.Hash {
				return fmt.Errorf("event hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			chainHash, err := integrity.ChainHash(evt, prevChainHash)
			if err != nil {
				return fmt.Errorf("compute chain hash campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}
			if chainHash != evt.ChainHash {
				return fmt.Errorf("chain hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			if err := s.keyring.VerifyChainHash(campaignID, chainHash, evt.Signature, evt.SignatureKeyID); err != nil {
				return fmt.Errorf("signature mismatch campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}

			prevChainHash = evt.ChainHash
			lastSeq = evt.Seq
		}
	}
}

func isConstraintError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT || code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}

func isSQLiteBusyError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_BUSY || code == sqlite3.SQLITE_LOCKED
}

func isParticipantUserConflict(err error) bool {
	if !isConstraintError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_participants_campaign_user") ||
		(strings.Contains(message, "participant") && strings.Contains(message, "user_id"))
}

func isParticipantClaimConflict(err error) bool {
	if !isConstraintError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "participant_claims") ||
		strings.Contains(message, "idx_participant_claims")
}

// GetEventByHash retrieves an event by its content hash.
func (s *Store) GetEventByHash(ctx context.Context, hash string) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}

	row, err := s.q.GetEventByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return event.Event{}, storage.ErrNotFound
		}
		return event.Event{}, fmt.Errorf("get event by hash: %w", err)
	}

	return eventRowDataToDomain(eventRowDataFromGetEventByHashRow(row))
}

// GetEventBySeq retrieves a specific event by sequence number.
func (s *Store) GetEventBySeq(ctx context.Context, campaignID string, seq uint64) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return event.Event{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetEventBySeq(ctx, db.GetEventBySeqParams{
		CampaignID: campaignID,
		Seq:        int64(seq),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return event.Event{}, storage.ErrNotFound
		}
		return event.Event{}, fmt.Errorf("get event by seq: %w", err)
	}

	return eventRowDataToDomain(eventRowDataFromGetEventBySeqRow(row))
}

// ListEvents returns events ordered by sequence ascending.
func (s *Store) ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.q.ListEvents(ctx, db.ListEventsParams{
		CampaignID: campaignID,
		Seq:        int64(afterSeq),
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	return eventRowsToDomain(rows)
}

// ListEventsBySession returns events for a specific session.
func (s *Store) ListEventsBySession(ctx context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.q.ListEventsBySession(ctx, db.ListEventsBySessionParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
		Seq:        int64(afterSeq),
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list events by session: %w", err)
	}

	return eventRowsBySessionToDomain(rows)
}

// GetLatestEventSeq returns the latest event sequence number for a campaign.
func (s *Store) GetLatestEventSeq(ctx context.Context, campaignID string) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return 0, fmt.Errorf("campaign id is required")
	}

	seq, err := s.q.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return 0, fmt.Errorf("get latest event seq: %w", err)
	}

	return uint64(seq), nil
}

// ListEventsPage returns a paginated, filtered, and sorted list of events.
func (s *Store) ListEventsPage(ctx context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	if err := ctx.Err(); err != nil {
		return storage.ListEventsPageResult{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(req.CampaignID) == "" {
		return storage.ListEventsPageResult{}, fmt.Errorf("campaign id is required")
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	if req.PageSize > 200 {
		req.PageSize = 200
	}

	plan := buildListEventsPageSQLPlan(req)

	// Build and execute the query
	query := fmt.Sprintf(
		"SELECT campaign_id, seq, event_hash, prev_event_hash, chain_hash, signature_key_id, event_signature, timestamp, event_type, session_id, request_id, invocation_id, actor_type, actor_id, entity_type, entity_id, system_id, system_version, correlation_id, causation_id, payload_json FROM events WHERE %s %s %s",
		plan.whereClause,
		plan.orderClause,
		plan.limitClause,
	)

	rows, err := s.sqlDB.QueryContext(ctx, query, plan.params...)
	if err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	events := make([]event.Event, 0, req.PageSize)
	for rows.Next() {
		var row db.Event
		if err := rows.Scan(
			&row.CampaignID,
			&row.Seq,
			&row.EventHash,
			&row.PrevEventHash,
			&row.ChainHash,
			&row.SignatureKeyID,
			&row.EventSignature,
			&row.Timestamp,
			&row.EventType,
			&row.SessionID,
			&row.RequestID,
			&row.InvocationID,
			&row.ActorType,
			&row.ActorID,
			&row.EntityType,
			&row.EntityID,
			&row.SystemID,
			&row.SystemVersion,
			&row.CorrelationID,
			&row.CausationID,
			&row.PayloadJson,
		); err != nil {
			return storage.ListEventsPageResult{}, fmt.Errorf("scan event: %w", err)
		}

		evt, err := eventRowDataToDomain(eventRowDataFromEvent(row))
		if err != nil {
			return storage.ListEventsPageResult{}, err
		}
		events = append(events, evt)
	}
	if err := rows.Err(); err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("iterate events: %w", err)
	}

	// Determine if there are more pages
	hasMore := len(events) > req.PageSize
	if hasMore {
		events = events[:req.PageSize]
	}

	// For "previous page" navigation, reverse the results to maintain consistent order
	if req.CursorReverse {
		for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
			events[i], events[j] = events[j], events[i]
		}
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events WHERE %s", plan.countWhereClause)
	var totalCount int
	if err := s.sqlDB.QueryRowContext(ctx, countQuery, plan.countParams...).Scan(&totalCount); err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("count events: %w", err)
	}

	// Determine hasPrev/hasNext based on pagination direction
	result := storage.ListEventsPageResult{
		Events:     events,
		TotalCount: totalCount,
	}

	if req.CursorReverse {
		result.HasNextPage = true // We came from next, so there is a next
		result.HasPrevPage = hasMore
	} else {
		result.HasNextPage = hasMore
		result.HasPrevPage = req.CursorSeq > 0
	}

	return result, nil
}

// Domain conversion helpers for events

type eventRowData struct {
	CampaignID     string
	Seq            int64
	EventHash      string
	PrevEventHash  string
	ChainHash      string
	SignatureKeyID string
	EventSignature string
	Timestamp      int64
	EventType      string
	SessionID      string
	RequestID      string
	InvocationID   string
	ActorType      string
	ActorID        string
	EntityType     string
	EntityID       string
	SystemID       string
	SystemVersion  string
	CorrelationID  string
	CausationID    string
	PayloadJSON    []byte
}

func eventRowDataToDomain(row eventRowData) (event.Event, error) {
	return event.Event{
		CampaignID:     row.CampaignID,
		Seq:            uint64(row.Seq),
		Hash:           row.EventHash,
		PrevHash:       row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		Signature:      row.EventSignature,
		Timestamp:      fromMillis(row.Timestamp),
		Type:           event.Type(row.EventType),
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      event.ActorType(row.ActorType),
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJSON,
	}, nil
}

func eventRowDataFromEvent(row db.Event) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromGetEventByHashRow(row db.GetEventByHashRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromGetEventBySeqRow(row db.GetEventBySeqRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromListEventsRow(row db.ListEventsRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromListEventsBySessionRow(row db.ListEventsBySessionRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		CorrelationID:  row.CorrelationID,
		CausationID:    row.CausationID,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowsToDomain(rows []db.ListEventsRow) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := eventRowDataToDomain(eventRowDataFromListEventsRow(row))
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}

func eventRowsBySessionToDomain(rows []db.ListEventsBySessionRow) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := eventRowDataToDomain(eventRowDataFromListEventsBySessionRow(row))
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}
