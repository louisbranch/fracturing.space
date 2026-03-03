package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func (s *Store) PutCampaignTurn(ctx context.Context, record storage.CampaignTurnRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("turn id is required")
	}
	if strings.TrimSpace(record.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(record.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(record.AgentID) == "" {
		return fmt.Errorf("agent id is required")
	}
	if strings.TrimSpace(record.InputText) == "" {
		return fmt.Errorf("input text is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_campaign_turns (
	id, campaign_id, session_id, agent_id, requester_user_id,
	participant_id, participant_name, correlation_message_id,
	input_text, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	campaign_id = excluded.campaign_id,
	session_id = excluded.session_id,
	agent_id = excluded.agent_id,
	requester_user_id = excluded.requester_user_id,
	participant_id = excluded.participant_id,
	participant_name = excluded.participant_name,
	correlation_message_id = excluded.correlation_message_id,
	input_text = excluded.input_text,
	status = excluded.status,
	updated_at = excluded.updated_at
`,
		record.ID,
		record.CampaignID,
		record.SessionID,
		record.AgentID,
		record.RequesterUserID,
		record.ParticipantID,
		record.ParticipantName,
		record.CorrelationMessage,
		record.InputText,
		record.Status,
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("put campaign turn: %w", err)
	}
	return nil
}

func (s *Store) UpdateCampaignTurnStatus(ctx context.Context, turnID string, statusValue string, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	turnID = strings.TrimSpace(turnID)
	if turnID == "" {
		return fmt.Errorf("turn id is required")
	}
	statusValue = strings.TrimSpace(statusValue)
	if statusValue == "" {
		return fmt.Errorf("status is required")
	}

	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_campaign_turns
SET status = ?, updated_at = ?
WHERE id = ?
`, statusValue, toMillis(updatedAt), turnID)
	if err != nil {
		return fmt.Errorf("update campaign turn status: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update campaign turn rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) AppendCampaignTurnEvent(ctx context.Context, record storage.CampaignTurnEventRecord) (storage.CampaignTurnEventRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.CampaignTurnEventRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CampaignTurnEventRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.CampaignID) == "" {
		return storage.CampaignTurnEventRecord{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(record.TurnID) == "" {
		return storage.CampaignTurnEventRecord{}, fmt.Errorf("turn id is required")
	}
	if strings.TrimSpace(record.Kind) == "" {
		return storage.CampaignTurnEventRecord{}, fmt.Errorf("event kind is required")
	}

	result, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_campaign_turn_events (
	campaign_id, session_id, turn_id, kind, content,
	participant_visible, correlation_message_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
		record.CampaignID,
		record.SessionID,
		record.TurnID,
		record.Kind,
		record.Content,
		boolToInt(record.ParticipantVisible),
		record.CorrelationMessage,
		toMillis(record.CreatedAt),
	)
	if err != nil {
		return storage.CampaignTurnEventRecord{}, fmt.Errorf("append campaign turn event: %w", err)
	}
	sequenceID, err := result.LastInsertId()
	if err != nil {
		return storage.CampaignTurnEventRecord{}, fmt.Errorf("campaign turn event last insert id: %w", err)
	}
	if sequenceID <= 0 {
		return storage.CampaignTurnEventRecord{}, fmt.Errorf("campaign turn event sequence id is invalid")
	}
	record.SequenceID = uint64(sequenceID)
	return record, nil
}

func (s *Store) ListCampaignTurnEvents(ctx context.Context, campaignID string, afterSequenceID uint64, limit int) ([]storage.CampaignTurnEventRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.sqlDB.QueryContext(ctx, `
SELECT sequence_id, campaign_id, session_id, turn_id, kind, content,
	participant_visible, correlation_message_id, created_at
FROM ai_campaign_turn_events
WHERE campaign_id = ? AND sequence_id > ?
ORDER BY sequence_id ASC
LIMIT ?
`, campaignID, int64(afterSequenceID), limit)
	if err != nil {
		return nil, fmt.Errorf("list campaign turn events: %w", err)
	}
	defer rows.Close()

	records := make([]storage.CampaignTurnEventRecord, 0, limit)
	for rows.Next() {
		var (
			rec                storage.CampaignTurnEventRecord
			sequenceID         int64
			participantVisible int64
			createdAt          int64
		)
		if err := rows.Scan(
			&sequenceID,
			&rec.CampaignID,
			&rec.SessionID,
			&rec.TurnID,
			&rec.Kind,
			&rec.Content,
			&participantVisible,
			&rec.CorrelationMessage,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan campaign turn event: %w", err)
		}
		if sequenceID <= 0 {
			return nil, fmt.Errorf("campaign turn event sequence id is invalid")
		}
		rec.SequenceID = uint64(sequenceID)
		rec.ParticipantVisible = participantVisible != 0
		rec.CreatedAt = fromMillis(createdAt)
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate campaign turn events: %w", err)
	}
	return records, nil
}

func (s *Store) GetLatestCampaignTurnEventSequence(ctx context.Context, campaignID string) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return 0, fmt.Errorf("campaign id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT sequence_id
FROM ai_campaign_turn_events
WHERE campaign_id = ?
ORDER BY sequence_id DESC
LIMIT 1
`, campaignID)
	var sequenceID int64
	if err := row.Scan(&sequenceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get latest campaign turn event sequence: %w", err)
	}
	if sequenceID <= 0 {
		return 0, nil
	}
	return uint64(sequenceID), nil
}

func boolToInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
