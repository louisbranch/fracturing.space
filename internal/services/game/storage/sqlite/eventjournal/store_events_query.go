package eventjournal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

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
//
// PageSize is capped at 200 rows to bound memory and query latency. Callers
// requesting more than 200 are silently clamped; a default of 50 applies when
// the caller supplies zero or a negative value.
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

	plan, err := buildListEventsPageSQLPlan(req)
	if err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("build list events query plan: %w", err)
	}

	// Build and execute the query
	query := fmt.Sprintf(
		"SELECT campaign_id, seq, event_hash, prev_event_hash, chain_hash, signature_key_id, event_signature, timestamp, event_type, session_id, scene_id, request_id, invocation_id, actor_type, actor_id, entity_type, entity_id, system_id, system_version, correlation_id, causation_id, payload_json FROM events WHERE %s %s %s",
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
			&row.SceneID,
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
