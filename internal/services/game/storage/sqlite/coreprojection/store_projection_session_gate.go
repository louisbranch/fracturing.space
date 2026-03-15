package coreprojection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

const countOpenSessionGatesQuery = `
SELECT COUNT(*)
FROM session_gates
WHERE campaign_id = ? AND session_id = ? AND status = 'open';
`

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
	if s.tx != nil {
		return putSessionGateWithQueries(ctx, s.q, gate)
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)
	if err := putSessionGateWithQueries(ctx, qtx, gate); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
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

	gate, err := loadSessionGateWithQueries(ctx, s.q, row)
	if err != nil {
		return storage.SessionGate{}, fmt.Errorf("load session gate details: %w", err)
	}
	return gate, nil
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

	var openGateCount int64
	if err := s.sqlDB.QueryRowContext(ctx, countOpenSessionGatesQuery, campaignID, sessionID).Scan(&openGateCount); err != nil {
		return storage.SessionGate{}, fmt.Errorf("count open session gates: %w", err)
	}
	switch {
	case openGateCount == 0:
		return storage.SessionGate{}, storage.ErrNotFound
	case openGateCount > 1:
		return storage.SessionGate{}, fmt.Errorf("get open session gate: multiple open session gates for campaign %q session %q", campaignID, sessionID)
	}

	row, err := s.q.GetOpenSessionGate(ctx, db.GetOpenSessionGateParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		return storage.SessionGate{}, fmt.Errorf("get open session gate: %w", err)
	}

	gate, err := loadSessionGateWithQueries(ctx, s.q, row)
	if err != nil {
		return storage.SessionGate{}, fmt.Errorf("load open session gate details: %w", err)
	}
	return gate, nil
}

func putSessionGateWithQueries(ctx context.Context, queries *db.Queries, gate storage.SessionGate) error {
	metadata, err := session.BuildStoredGateMetadata(gate.GateType, gate.Metadata)
	if err != nil {
		return fmt.Errorf("build session gate metadata: %w", err)
	}
	metadataExtraJSON, err := marshalOptionalJSONBytes(metadata.Extra, "encode session gate metadata extras")
	if err != nil {
		return err
	}

	resolution, err := session.BuildStoredGateResolution("", gate.Resolution)
	if err != nil {
		return fmt.Errorf("build session gate resolution: %w", err)
	}
	resolutionExtraJSON, err := marshalOptionalJSONBytes(resolution.Extra, "encode session gate resolution extras")
	if err != nil {
		return err
	}

	if err := queries.PutSessionGate(ctx, db.PutSessionGateParams{
		CampaignID:          gate.CampaignID,
		SessionID:           gate.SessionID,
		GateID:              gate.GateID,
		GateType:            gate.GateType,
		Status:              strings.TrimSpace(string(gate.Status)),
		Reason:              gate.Reason,
		CreatedAt:           toMillis(gate.CreatedAt),
		CreatedByActorType:  gate.CreatedByActorType,
		CreatedByActorID:    gate.CreatedByActorID,
		ResolvedAt:          toNullMillis(gate.ResolvedAt),
		ResolvedByActorType: toNullString(gate.ResolvedByActorType),
		ResolvedByActorID:   toNullString(gate.ResolvedByActorID),
		ResponseAuthority:   metadata.ResponseAuthority,
		MetadataExtraJson:   metadataExtraJSON,
		ResolutionDecision:  resolution.Decision,
		ResolutionExtraJson: resolutionExtraJSON,
	}); err != nil {
		return fmt.Errorf("put session gate: %w", err)
	}

	key := db.DeleteSessionGateEligibleParticipantsParams{
		CampaignID: gate.CampaignID,
		SessionID:  gate.SessionID,
		GateID:     gate.GateID,
	}
	if err := queries.DeleteSessionGateEligibleParticipants(ctx, key); err != nil {
		return fmt.Errorf("delete session gate eligible participants: %w", err)
	}
	for idx, participantID := range metadata.EligibleParticipantIDs {
		if err := queries.PutSessionGateEligibleParticipant(ctx, db.PutSessionGateEligibleParticipantParams{
			CampaignID:    gate.CampaignID,
			SessionID:     gate.SessionID,
			GateID:        gate.GateID,
			Position:      int64(idx),
			ParticipantID: participantID,
		}); err != nil {
			return fmt.Errorf("put session gate eligible participant: %w", err)
		}
	}

	if err := queries.DeleteSessionGateOptions(ctx, db.DeleteSessionGateOptionsParams(key)); err != nil {
		return fmt.Errorf("delete session gate options: %w", err)
	}
	for idx, option := range metadata.Options {
		if err := queries.PutSessionGateOption(ctx, db.PutSessionGateOptionParams{
			CampaignID:  gate.CampaignID,
			SessionID:   gate.SessionID,
			GateID:      gate.GateID,
			Position:    int64(idx),
			OptionValue: option,
		}); err != nil {
			return fmt.Errorf("put session gate option: %w", err)
		}
	}

	if err := queries.DeleteSessionGateResponses(ctx, db.DeleteSessionGateResponsesParams(key)); err != nil {
		return fmt.Errorf("delete session gate responses: %w", err)
	}
	for _, response := range sessionGateResponsesFromProgress(gate.Progress) {
		recordedAt, err := sessionGateRecordedAtToNullMillis(response.RecordedAt)
		if err != nil {
			return fmt.Errorf("encode session gate response timestamp for participant %q: %w", response.ParticipantID, err)
		}
		responseJSON, err := marshalOptionalJSONBytes(response.Response, "encode session gate response payload")
		if err != nil {
			return err
		}
		if err := queries.PutSessionGateResponse(ctx, db.PutSessionGateResponseParams{
			CampaignID:    gate.CampaignID,
			SessionID:     gate.SessionID,
			GateID:        gate.GateID,
			ParticipantID: strings.TrimSpace(response.ParticipantID),
			Decision:      strings.TrimSpace(response.Decision),
			ResponseJson:  responseJSON,
			RecordedAt:    recordedAt,
			ActorType:     strings.TrimSpace(response.ActorType),
			ActorID:       strings.TrimSpace(response.ActorID),
		}); err != nil {
			return fmt.Errorf("put session gate response: %w", err)
		}
	}

	return nil
}

func loadSessionGateWithQueries(ctx context.Context, queries *db.Queries, row db.SessionGate) (storage.SessionGate, error) {
	key := db.ListSessionGateEligibleParticipantsParams{
		CampaignID: row.CampaignID,
		SessionID:  row.SessionID,
		GateID:     row.GateID,
	}
	eligibleParticipantIDs, err := queries.ListSessionGateEligibleParticipants(ctx, key)
	if err != nil {
		return storage.SessionGate{}, fmt.Errorf("list session gate eligible participants: %w", err)
	}
	options, err := queries.ListSessionGateOptions(ctx, db.ListSessionGateOptionsParams(key))
	if err != nil {
		return storage.SessionGate{}, fmt.Errorf("list session gate options: %w", err)
	}
	responseRows, err := queries.ListSessionGateResponses(ctx, db.ListSessionGateResponsesParams(key))
	if err != nil {
		return storage.SessionGate{}, fmt.Errorf("list session gate responses: %w", err)
	}

	metadataExtra, err := decodeOptionalJSONObjectBytes(row.MetadataExtraJson, "decode session gate metadata extras")
	if err != nil {
		return storage.SessionGate{}, err
	}
	metadata, err := session.BuildGateMetadataMapFromStored(row.GateType, session.StoredGateMetadata{
		ResponseAuthority:      row.ResponseAuthority,
		EligibleParticipantIDs: eligibleParticipantIDs,
		Options:                options,
		Extra:                  metadataExtra,
	})
	if err != nil {
		return storage.SessionGate{}, fmt.Errorf("build session gate metadata: %w", err)
	}

	responses, err := sessionGateResponsesToProgressRows(responseRows)
	if err != nil {
		return storage.SessionGate{}, err
	}
	progress, err := session.BuildGateProgressFromResponses(row.GateType, metadata, responses)
	if err != nil {
		return storage.SessionGate{}, fmt.Errorf("build session gate progress: %w", err)
	}

	resolutionExtra, err := decodeOptionalJSONObjectBytes(row.ResolutionExtraJson, "decode session gate resolution extras")
	if err != nil {
		return storage.SessionGate{}, err
	}
	resolution, err := session.BuildGateResolutionMapFromStored(row.ResolutionDecision, resolutionExtra)
	if err != nil {
		return storage.SessionGate{}, fmt.Errorf("build session gate resolution: %w", err)
	}

	gate := storage.SessionGate{
		CampaignID:         row.CampaignID,
		SessionID:          row.SessionID,
		GateID:             row.GateID,
		GateType:           row.GateType,
		Status:             session.GateStatus(strings.ToLower(strings.TrimSpace(row.Status))),
		Reason:             row.Reason,
		CreatedAt:          fromMillis(row.CreatedAt),
		CreatedByActorType: row.CreatedByActorType,
		CreatedByActorID:   row.CreatedByActorID,
		Metadata:           metadata,
		Progress:           progress,
		Resolution:         resolution,
		ResolvedAt:         fromNullMillis(row.ResolvedAt),
	}
	if row.ResolvedByActorType.Valid {
		gate.ResolvedByActorType = row.ResolvedByActorType.String
	}
	if row.ResolvedByActorID.Valid {
		gate.ResolvedByActorID = row.ResolvedByActorID.String
	}
	return gate, nil
}

func sessionGateResponsesFromProgress(progress *session.GateProgress) []session.GateProgressResponse {
	if progress == nil || len(progress.Responses) == 0 {
		return nil
	}
	return append([]session.GateProgressResponse(nil), progress.Responses...)
}

func sessionGateResponsesToProgressRows(rows []db.ListSessionGateResponsesRow) ([]session.GateProgressResponse, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	responses := make([]session.GateProgressResponse, 0, len(rows))
	for _, row := range rows {
		responsePayload, err := decodeOptionalJSONObjectBytes(row.ResponseJson, "decode session gate response payload")
		if err != nil {
			return nil, err
		}
		response := session.GateProgressResponse{
			ParticipantID: row.ParticipantID,
			Decision:      row.Decision,
			Response:      responsePayload,
			ActorType:     row.ActorType,
			ActorID:       row.ActorID,
		}
		if row.RecordedAt.Valid {
			response.RecordedAt = fromMillis(row.RecordedAt.Int64).UTC().Format(time.RFC3339Nano)
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func sessionGateRecordedAtToNullMillis(value string) (sql.NullInt64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullInt64{}, nil
	}
	recordedAt, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: toMillis(recordedAt.UTC()), Valid: true}, nil
}

func marshalOptionalJSONBytes(value map[string]any, message string) ([]byte, error) {
	if len(value) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", message, err)
	}
	return data, nil
}

func decodeOptionalJSONObjectBytes(data []byte, message string) (map[string]any, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("%s: %w", message, err)
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values, nil
}
