package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/ai/debugtrace"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func scanCampaignDebugTurn(s scanner) (debugtrace.Turn, error) {
	var (
		turn                                     debugtrace.Turn
		providerRaw, statusRaw                   string
		inputTokens, outputTokens                int32
		reasoningTokens, totalTokens, entryCount int32
		startedAt, updatedAt                     int64
		completedAt                              sql.NullInt64
	)
	if err := s.Scan(
		&turn.ID,
		&turn.CampaignID,
		&turn.SessionID,
		&turn.TurnToken,
		&turn.ParticipantID,
		&providerRaw,
		&turn.Model,
		&statusRaw,
		&turn.LastError,
		&inputTokens,
		&outputTokens,
		&reasoningTokens,
		&totalTokens,
		&startedAt,
		&updatedAt,
		&completedAt,
		&entryCount,
	); err != nil {
		return debugtrace.Turn{}, err
	}
	normalizedProvider, _ := provider.Normalize(providerRaw)
	turn.Provider = normalizedProvider
	turn.Status = debugtrace.ParseStatus(statusRaw)
	turn.Usage = provider.Usage{
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		ReasoningTokens: reasoningTokens,
		TotalTokens:     totalTokens,
	}
	turn.StartedAt = sqliteutil.FromMillis(startedAt)
	turn.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	turn.EntryCount = int(entryCount)
	if completedAt.Valid {
		value := sqliteutil.FromMillis(completedAt.Int64)
		turn.CompletedAt = &value
	}
	return turn, nil
}

func scanCampaignDebugTurnEntry(s scanner) (debugtrace.Entry, error) {
	var (
		entry                        debugtrace.Entry
		kindRaw                      string
		payloadTruncated, isError    bool
		inputTokens, outputTokens    int32
		reasoningTokens, totalTokens int32
		createdAt                    int64
	)
	if err := s.Scan(
		&entry.TurnID,
		&entry.Sequence,
		&kindRaw,
		&entry.ToolName,
		&entry.Payload,
		&payloadTruncated,
		&entry.CallID,
		&entry.ResponseID,
		&isError,
		&inputTokens,
		&outputTokens,
		&reasoningTokens,
		&totalTokens,
		&createdAt,
	); err != nil {
		return debugtrace.Entry{}, err
	}
	entry.Kind = debugtrace.ParseEntryKind(kindRaw)
	entry.PayloadTruncated = payloadTruncated
	entry.IsError = isError
	entry.CreatedAt = sqliteutil.FromMillis(createdAt)
	entry.Usage = provider.Usage{
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		ReasoningTokens: reasoningTokens,
		TotalTokens:     totalTokens,
	}
	return entry, nil
}

func debugTurnPageToken(startedAtMillis int64, turnID string) string {
	return fmt.Sprintf("%d|%s", startedAtMillis, strings.TrimSpace(turnID))
}

func parseDebugTurnPageToken(pageToken string) (int64, string, error) {
	parts := strings.SplitN(strings.TrimSpace(pageToken), "|", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return 0, "", fmt.Errorf("invalid page token")
	}
	millis, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid page token: %w", err)
	}
	return millis, strings.TrimSpace(parts[1]), nil
}

// PutCampaignDebugTurn upserts one debug turn summary.
func (s *Store) PutCampaignDebugTurn(ctx context.Context, turn debugtrace.Turn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(turn.ID) == "" {
		return fmt.Errorf("turn id is required")
	}
	if strings.TrimSpace(turn.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(turn.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	if turn.StartedAt.IsZero() {
		return fmt.Errorf("started at is required")
	}
	var completedAt sql.NullInt64
	if turn.CompletedAt != nil {
		completedAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*turn.CompletedAt), Valid: true}
	}
	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_campaign_debug_turns (
	id, campaign_id, session_id, turn_token, participant_id, provider, model, status, last_error,
	input_tokens, output_tokens, reasoning_tokens, total_tokens,
	started_at, updated_at, completed_at, entry_count
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	campaign_id = excluded.campaign_id,
	session_id = excluded.session_id,
	turn_token = excluded.turn_token,
	participant_id = excluded.participant_id,
	provider = excluded.provider,
	model = excluded.model,
	status = excluded.status,
	last_error = excluded.last_error,
	input_tokens = excluded.input_tokens,
	output_tokens = excluded.output_tokens,
	reasoning_tokens = excluded.reasoning_tokens,
	total_tokens = excluded.total_tokens,
	started_at = excluded.started_at,
	updated_at = excluded.updated_at,
	completed_at = excluded.completed_at,
	entry_count = excluded.entry_count
`,
		turn.ID,
		turn.CampaignID,
		turn.SessionID,
		strings.TrimSpace(turn.TurnToken),
		strings.TrimSpace(turn.ParticipantID),
		string(turn.Provider),
		strings.TrimSpace(turn.Model),
		string(turn.Status),
		strings.TrimSpace(turn.LastError),
		turn.Usage.InputTokens,
		turn.Usage.OutputTokens,
		turn.Usage.ReasoningTokens,
		turn.Usage.TotalTokens,
		sqliteutil.ToMillis(turn.StartedAt),
		sqliteutil.ToMillis(turn.UpdatedAt),
		completedAt,
		turn.EntryCount,
	)
	if err != nil {
		return fmt.Errorf("put campaign debug turn: %w", err)
	}
	return nil
}

// PutCampaignDebugTurnEntry appends one ordered trace entry.
func (s *Store) PutCampaignDebugTurnEntry(ctx context.Context, entry debugtrace.Entry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(entry.TurnID) == "" {
		return fmt.Errorf("turn id is required")
	}
	if entry.Sequence <= 0 {
		return fmt.Errorf("sequence must be greater than zero")
	}
	if entry.CreatedAt.IsZero() {
		return fmt.Errorf("created at is required")
	}
	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_campaign_debug_turn_entries (
	turn_id, sequence, kind, tool_name, payload, payload_truncated, call_id, response_id, is_error,
	input_tokens, output_tokens, reasoning_tokens, total_tokens, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(turn_id, sequence) DO UPDATE SET
	kind = excluded.kind,
	tool_name = excluded.tool_name,
	payload = excluded.payload,
	payload_truncated = excluded.payload_truncated,
	call_id = excluded.call_id,
	response_id = excluded.response_id,
	is_error = excluded.is_error,
	input_tokens = excluded.input_tokens,
	output_tokens = excluded.output_tokens,
	reasoning_tokens = excluded.reasoning_tokens,
	total_tokens = excluded.total_tokens,
	created_at = excluded.created_at
`,
		entry.TurnID,
		entry.Sequence,
		string(entry.Kind),
		strings.TrimSpace(entry.ToolName),
		entry.Payload,
		entry.PayloadTruncated,
		strings.TrimSpace(entry.CallID),
		strings.TrimSpace(entry.ResponseID),
		entry.IsError,
		entry.Usage.InputTokens,
		entry.Usage.OutputTokens,
		entry.Usage.ReasoningTokens,
		entry.Usage.TotalTokens,
		sqliteutil.ToMillis(entry.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("put campaign debug turn entry: %w", err)
	}
	return nil
}

// ListCampaignDebugTurns returns newest-first paginated turn summaries for one session.
func (s *Store) ListCampaignDebugTurns(ctx context.Context, campaignID string, sessionID string, pageSize int, pageToken string) (debugtrace.Page, error) {
	if err := ctx.Err(); err != nil {
		return debugtrace.Page{}, err
	}
	if s == nil || s.sqlDB == nil {
		return debugtrace.Page{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return debugtrace.Page{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return debugtrace.Page{}, fmt.Errorf("session id is required")
	}
	if pageSize <= 0 {
		return debugtrace.Page{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, campaign_id, session_id, turn_token, participant_id, provider, model, status, last_error,
       input_tokens, output_tokens, reasoning_tokens, total_tokens,
       started_at, updated_at, completed_at, entry_count
FROM ai_campaign_debug_turns
WHERE campaign_id = ? AND session_id = ?
ORDER BY started_at DESC, id DESC
LIMIT ?
`, campaignID, sessionID, limit)
	} else {
		cursorStartedAt, cursorID, parseErr := parseDebugTurnPageToken(pageToken)
		if parseErr != nil {
			return debugtrace.Page{}, parseErr
		}
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, campaign_id, session_id, turn_token, participant_id, provider, model, status, last_error,
       input_tokens, output_tokens, reasoning_tokens, total_tokens,
       started_at, updated_at, completed_at, entry_count
FROM ai_campaign_debug_turns
WHERE campaign_id = ? AND session_id = ?
  AND (started_at < ? OR (started_at = ? AND id < ?))
ORDER BY started_at DESC, id DESC
LIMIT ?
`, campaignID, sessionID, cursorStartedAt, cursorStartedAt, cursorID, limit)
	}
	if err != nil {
		return debugtrace.Page{}, fmt.Errorf("list campaign debug turns: %w", err)
	}
	defer rows.Close()

	page := debugtrace.Page{Turns: make([]debugtrace.Turn, 0, pageSize)}
	for rows.Next() {
		turn, scanErr := scanCampaignDebugTurn(rows)
		if scanErr != nil {
			return debugtrace.Page{}, fmt.Errorf("scan campaign debug turn row: %w", scanErr)
		}
		page.Turns = append(page.Turns, turn)
	}
	if err := rows.Err(); err != nil {
		return debugtrace.Page{}, fmt.Errorf("iterate campaign debug turns: %w", err)
	}
	if len(page.Turns) > pageSize {
		last := page.Turns[pageSize-1]
		page.NextPageToken = debugTurnPageToken(sqliteutil.ToMillis(last.StartedAt), last.ID)
		page.Turns = page.Turns[:pageSize]
	}
	return page, nil
}

// GetCampaignDebugTurn fetches one turn summary by campaign and id.
func (s *Store) GetCampaignDebugTurn(ctx context.Context, campaignID string, turnID string) (debugtrace.Turn, error) {
	if err := ctx.Err(); err != nil {
		return debugtrace.Turn{}, err
	}
	if s == nil || s.sqlDB == nil {
		return debugtrace.Turn{}, fmt.Errorf("storage is not configured")
	}
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, campaign_id, session_id, turn_token, participant_id, provider, model, status, last_error,
       input_tokens, output_tokens, reasoning_tokens, total_tokens,
       started_at, updated_at, completed_at, entry_count
FROM ai_campaign_debug_turns
WHERE campaign_id = ? AND id = ?
`, strings.TrimSpace(campaignID), strings.TrimSpace(turnID))
	turn, err := scanCampaignDebugTurn(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return debugtrace.Turn{}, storage.ErrNotFound
		}
		return debugtrace.Turn{}, fmt.Errorf("get campaign debug turn: %w", err)
	}
	return turn, nil
}

// ListCampaignDebugTurnEntries returns all ordered entries for one turn.
func (s *Store) ListCampaignDebugTurnEntries(ctx context.Context, turnID string) ([]debugtrace.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	rows, err := s.sqlDB.QueryContext(ctx, `
SELECT turn_id, sequence, kind, tool_name, payload, payload_truncated, call_id, response_id, is_error,
       input_tokens, output_tokens, reasoning_tokens, total_tokens, created_at
FROM ai_campaign_debug_turn_entries
WHERE turn_id = ?
ORDER BY sequence
`, strings.TrimSpace(turnID))
	if err != nil {
		return nil, fmt.Errorf("list campaign debug turn entries: %w", err)
	}
	defer rows.Close()
	entries := make([]debugtrace.Entry, 0, 8)
	for rows.Next() {
		entry, scanErr := scanCampaignDebugTurnEntry(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan campaign debug turn entry row: %w", scanErr)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate campaign debug turn entries: %w", err)
	}
	return entries, nil
}
