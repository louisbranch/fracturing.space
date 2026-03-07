package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Scene methods

// PutScene stores a scene projection record.
func (s *Store) PutScene(ctx context.Context, rec storage.SceneRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(rec.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(rec.SceneID) == "" {
		return fmt.Errorf("scene id is required")
	}
	if strings.TrimSpace(rec.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	active := boolToInt(rec.Active)
	endedAt := toNullMillis(rec.EndedAt)

	_, err := s.sqlDB.ExecContext(ctx,
		`INSERT INTO scenes (campaign_id, scene_id, session_id, name, description, active, created_at, updated_at, ended_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (campaign_id, scene_id) DO UPDATE SET
		   session_id = excluded.session_id,
		   name = excluded.name,
		   description = excluded.description,
		   active = excluded.active,
		   updated_at = excluded.updated_at,
		   ended_at = excluded.ended_at`,
		rec.CampaignID, rec.SceneID, rec.SessionID, rec.Name, rec.Description,
		active, toMillis(rec.CreatedAt), toMillis(rec.UpdatedAt), endedAt,
	)
	if err != nil {
		return fmt.Errorf("put scene: %w", err)
	}
	return nil
}

// EndScene marks a scene as ended.
func (s *Store) EndScene(ctx context.Context, campaignID, sceneID string, endedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return fmt.Errorf("scene id is required")
	}

	endedAtMillis := toMillis(endedAt)
	result, err := s.sqlDB.ExecContext(ctx,
		`UPDATE scenes SET active = 0, updated_at = ?, ended_at = ? WHERE campaign_id = ? AND scene_id = ?`,
		endedAtMillis, endedAtMillis, campaignID, sceneID,
	)
	if err != nil {
		return fmt.Errorf("end scene: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// GetScene retrieves a scene by campaign ID and scene ID.
func (s *Store) GetScene(ctx context.Context, campaignID, sceneID string) (storage.SceneRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.SceneRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SceneRecord{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SceneRecord{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return storage.SceneRecord{}, fmt.Errorf("scene id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx,
		`SELECT campaign_id, scene_id, session_id, name, description, active, created_at, updated_at, ended_at
		 FROM scenes WHERE campaign_id = ? AND scene_id = ?`,
		campaignID, sceneID,
	)
	return scanSceneRow(row)
}

// ListScenes returns a page of scene records for a session.
func (s *Store) ListScenes(ctx context.Context, campaignID, sessionID string, pageSize int, pageToken string) (storage.ScenePage, error) {
	if err := ctx.Err(); err != nil {
		return storage.ScenePage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ScenePage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ScenePage{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.ScenePage{}, fmt.Errorf("session id is required")
	}
	if pageSize <= 0 {
		return storage.ScenePage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows *sql.Rows
	var err error
	limit := int64(pageSize + 1)

	if pageToken == "" {
		rows, err = s.sqlDB.QueryContext(ctx,
			`SELECT campaign_id, scene_id, session_id, name, description, active, created_at, updated_at, ended_at
			 FROM scenes WHERE campaign_id = ? AND session_id = ?
			 ORDER BY scene_id ASC LIMIT ?`,
			campaignID, sessionID, limit,
		)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx,
			`SELECT campaign_id, scene_id, session_id, name, description, active, created_at, updated_at, ended_at
			 FROM scenes WHERE campaign_id = ? AND session_id = ? AND scene_id > ?
			 ORDER BY scene_id ASC LIMIT ?`,
			campaignID, sessionID, pageToken, limit,
		)
	}
	if err != nil {
		return storage.ScenePage{}, fmt.Errorf("list scenes: %w", err)
	}
	defer rows.Close()

	page := storage.ScenePage{
		Scenes: make([]storage.SceneRecord, 0, pageSize),
	}
	i := 0
	for rows.Next() {
		if i >= pageSize {
			page.NextPageToken = page.Scenes[pageSize-1].SceneID
			break
		}
		rec, err := scanSceneRows(rows)
		if err != nil {
			return storage.ScenePage{}, err
		}
		page.Scenes = append(page.Scenes, rec)
		i++
	}
	if err := rows.Err(); err != nil {
		return storage.ScenePage{}, fmt.Errorf("list scenes rows: %w", err)
	}
	return page, nil
}

// ListActiveScenes returns all active scenes for a campaign.
func (s *Store) ListActiveScenes(ctx context.Context, campaignID string) ([]storage.SceneRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}

	rows, err := s.sqlDB.QueryContext(ctx,
		`SELECT campaign_id, scene_id, session_id, name, description, active, created_at, updated_at, ended_at
		 FROM scenes WHERE campaign_id = ? AND active = 1
		 ORDER BY scene_id ASC`,
		campaignID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active scenes: %w", err)
	}
	defer rows.Close()

	var result []storage.SceneRecord
	for rows.Next() {
		rec, err := scanSceneRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list active scenes rows: %w", err)
	}
	return result, nil
}

// Scene character methods

// PutSceneCharacter adds a character to a scene.
func (s *Store) PutSceneCharacter(ctx context.Context, rec storage.SceneCharacterRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(rec.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(rec.SceneID) == "" {
		return fmt.Errorf("scene id is required")
	}
	if strings.TrimSpace(rec.CharacterID) == "" {
		return fmt.Errorf("character id is required")
	}

	_, err := s.sqlDB.ExecContext(ctx,
		`INSERT INTO scene_characters (campaign_id, scene_id, character_id, added_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (campaign_id, scene_id, character_id) DO NOTHING`,
		rec.CampaignID, rec.SceneID, rec.CharacterID, toMillis(rec.AddedAt),
	)
	if err != nil {
		return fmt.Errorf("put scene character: %w", err)
	}
	return nil
}

// DeleteSceneCharacter removes a character from a scene.
func (s *Store) DeleteSceneCharacter(ctx context.Context, campaignID, sceneID, characterID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return fmt.Errorf("scene id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return fmt.Errorf("character id is required")
	}

	_, err := s.sqlDB.ExecContext(ctx,
		`DELETE FROM scene_characters WHERE campaign_id = ? AND scene_id = ? AND character_id = ?`,
		campaignID, sceneID, characterID,
	)
	if err != nil {
		return fmt.Errorf("delete scene character: %w", err)
	}
	return nil
}

// ListSceneCharacters returns all characters in a scene.
func (s *Store) ListSceneCharacters(ctx context.Context, campaignID, sceneID string) ([]storage.SceneCharacterRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return nil, fmt.Errorf("scene id is required")
	}

	rows, err := s.sqlDB.QueryContext(ctx,
		`SELECT campaign_id, scene_id, character_id, added_at
		 FROM scene_characters WHERE campaign_id = ? AND scene_id = ?
		 ORDER BY character_id ASC`,
		campaignID, sceneID,
	)
	if err != nil {
		return nil, fmt.Errorf("list scene characters: %w", err)
	}
	defer rows.Close()

	var result []storage.SceneCharacterRecord
	for rows.Next() {
		var rec storage.SceneCharacterRecord
		var addedAt int64
		if err := rows.Scan(&rec.CampaignID, &rec.SceneID, &rec.CharacterID, &addedAt); err != nil {
			return nil, fmt.Errorf("scan scene character: %w", err)
		}
		rec.AddedAt = fromMillis(addedAt)
		result = append(result, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list scene characters rows: %w", err)
	}
	return result, nil
}

// Scene gate methods

// PutSceneGate persists a scene gate projection.
func (s *Store) PutSceneGate(ctx context.Context, gate storage.SceneGate) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(gate.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(gate.SceneID) == "" {
		return fmt.Errorf("scene id is required")
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

	_, err := s.sqlDB.ExecContext(ctx,
		`INSERT INTO scene_gates (campaign_id, scene_id, gate_id, gate_type, status, reason,
		   created_at, created_by_actor_type, created_by_actor_id,
		   resolved_at, resolved_by_actor_type, resolved_by_actor_id,
		   metadata_json, resolution_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (campaign_id, scene_id, gate_id) DO UPDATE SET
		   status = excluded.status,
		   reason = excluded.reason,
		   resolved_at = excluded.resolved_at,
		   resolved_by_actor_type = excluded.resolved_by_actor_type,
		   resolved_by_actor_id = excluded.resolved_by_actor_id,
		   resolution_json = excluded.resolution_json`,
		gate.CampaignID, gate.SceneID, gate.GateID, gate.GateType, status, gate.Reason,
		toMillis(gate.CreatedAt), gate.CreatedByActorType, gate.CreatedByActorID,
		toNullMillis(gate.ResolvedAt), toNullString(gate.ResolvedByActorType), toNullString(gate.ResolvedByActorID),
		gate.MetadataJSON, gate.ResolutionJSON,
	)
	if err != nil {
		return fmt.Errorf("put scene gate: %w", err)
	}
	return nil
}

// GetSceneGate retrieves a scene gate by id.
func (s *Store) GetSceneGate(ctx context.Context, campaignID, sceneID, gateID string) (storage.SceneGate, error) {
	if err := ctx.Err(); err != nil {
		return storage.SceneGate{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SceneGate{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SceneGate{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return storage.SceneGate{}, fmt.Errorf("scene id is required")
	}
	if strings.TrimSpace(gateID) == "" {
		return storage.SceneGate{}, fmt.Errorf("gate id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx,
		`SELECT campaign_id, scene_id, gate_id, gate_type, status, reason,
		   created_at, created_by_actor_type, created_by_actor_id,
		   resolved_at, resolved_by_actor_type, resolved_by_actor_id,
		   metadata_json, resolution_json
		 FROM scene_gates WHERE campaign_id = ? AND scene_id = ? AND gate_id = ?`,
		campaignID, sceneID, gateID,
	)
	return scanSceneGateRow(row)
}

// GetOpenSceneGate retrieves the open gate for a scene.
func (s *Store) GetOpenSceneGate(ctx context.Context, campaignID, sceneID string) (storage.SceneGate, error) {
	if err := ctx.Err(); err != nil {
		return storage.SceneGate{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SceneGate{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SceneGate{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return storage.SceneGate{}, fmt.Errorf("scene id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx,
		`SELECT campaign_id, scene_id, gate_id, gate_type, status, reason,
		   created_at, created_by_actor_type, created_by_actor_id,
		   resolved_at, resolved_by_actor_type, resolved_by_actor_id,
		   metadata_json, resolution_json
		 FROM scene_gates WHERE campaign_id = ? AND scene_id = ? AND status = 'open'
		 LIMIT 1`,
		campaignID, sceneID,
	)
	return scanSceneGateRow(row)
}

// Scene spotlight methods

// PutSceneSpotlight persists a scene spotlight projection.
func (s *Store) PutSceneSpotlight(ctx context.Context, spotlight storage.SceneSpotlight) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(spotlight.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(spotlight.SceneID) == "" {
		return fmt.Errorf("scene id is required")
	}
	spotlightType := strings.TrimSpace(string(spotlight.SpotlightType))
	if spotlightType == "" {
		return fmt.Errorf("spotlight type is required")
	}

	_, err := s.sqlDB.ExecContext(ctx,
		`INSERT INTO scene_spotlight (campaign_id, scene_id, spotlight_type, character_id, updated_at, updated_by_actor_type, updated_by_actor_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (campaign_id, scene_id) DO UPDATE SET
		   spotlight_type = excluded.spotlight_type,
		   character_id = excluded.character_id,
		   updated_at = excluded.updated_at,
		   updated_by_actor_type = excluded.updated_by_actor_type,
		   updated_by_actor_id = excluded.updated_by_actor_id`,
		spotlight.CampaignID, spotlight.SceneID, spotlightType, spotlight.CharacterID,
		toMillis(spotlight.UpdatedAt), spotlight.UpdatedByActorType, spotlight.UpdatedByActorID,
	)
	if err != nil {
		return fmt.Errorf("put scene spotlight: %w", err)
	}
	return nil
}

// GetSceneSpotlight retrieves a scene spotlight by scene id.
func (s *Store) GetSceneSpotlight(ctx context.Context, campaignID, sceneID string) (storage.SceneSpotlight, error) {
	if err := ctx.Err(); err != nil {
		return storage.SceneSpotlight{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SceneSpotlight{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SceneSpotlight{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return storage.SceneSpotlight{}, fmt.Errorf("scene id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx,
		`SELECT campaign_id, scene_id, spotlight_type, character_id, updated_at, updated_by_actor_type, updated_by_actor_id
		 FROM scene_spotlight WHERE campaign_id = ? AND scene_id = ?`,
		campaignID, sceneID,
	)

	var spotlight storage.SceneSpotlight
	var updatedAt int64
	var spotlightType string
	err := row.Scan(&spotlight.CampaignID, &spotlight.SceneID, &spotlightType, &spotlight.CharacterID,
		&updatedAt, &spotlight.UpdatedByActorType, &spotlight.UpdatedByActorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SceneSpotlight{}, storage.ErrNotFound
		}
		return storage.SceneSpotlight{}, fmt.Errorf("get scene spotlight: %w", err)
	}
	spotlight.SpotlightType = scene.SpotlightType(strings.ToLower(strings.TrimSpace(spotlightType)))
	spotlight.UpdatedAt = fromMillis(updatedAt)
	return spotlight, nil
}

// ClearSceneSpotlight removes the current spotlight for a scene.
func (s *Store) ClearSceneSpotlight(ctx context.Context, campaignID, sceneID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return fmt.Errorf("scene id is required")
	}

	_, err := s.sqlDB.ExecContext(ctx,
		`DELETE FROM scene_spotlight WHERE campaign_id = ? AND scene_id = ?`,
		campaignID, sceneID,
	)
	if err != nil {
		return fmt.Errorf("clear scene spotlight: %w", err)
	}
	return nil
}

// Scan helpers

func scanSceneRow(row *sql.Row) (storage.SceneRecord, error) {
	var rec storage.SceneRecord
	var active int64
	var createdAt, updatedAt int64
	var endedAt sql.NullInt64
	err := row.Scan(&rec.CampaignID, &rec.SceneID, &rec.SessionID, &rec.Name, &rec.Description,
		&active, &createdAt, &updatedAt, &endedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SceneRecord{}, storage.ErrNotFound
		}
		return storage.SceneRecord{}, fmt.Errorf("scan scene: %w", err)
	}
	rec.Active = intToBool(active)
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	rec.EndedAt = fromNullMillis(endedAt)
	return rec, nil
}

func scanSceneRows(rows *sql.Rows) (storage.SceneRecord, error) {
	var rec storage.SceneRecord
	var active int64
	var createdAt, updatedAt int64
	var endedAt sql.NullInt64
	err := rows.Scan(&rec.CampaignID, &rec.SceneID, &rec.SessionID, &rec.Name, &rec.Description,
		&active, &createdAt, &updatedAt, &endedAt)
	if err != nil {
		return storage.SceneRecord{}, fmt.Errorf("scan scene: %w", err)
	}
	rec.Active = intToBool(active)
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	rec.EndedAt = fromNullMillis(endedAt)
	return rec, nil
}

func scanSceneGateRow(row *sql.Row) (storage.SceneGate, error) {
	var gate storage.SceneGate
	var status string
	var createdAt int64
	var resolvedAt sql.NullInt64
	var resolvedByActorType, resolvedByActorID sql.NullString
	err := row.Scan(&gate.CampaignID, &gate.SceneID, &gate.GateID, &gate.GateType,
		&status, &gate.Reason, &createdAt, &gate.CreatedByActorType, &gate.CreatedByActorID,
		&resolvedAt, &resolvedByActorType, &resolvedByActorID,
		&gate.MetadataJSON, &gate.ResolutionJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SceneGate{}, storage.ErrNotFound
		}
		return storage.SceneGate{}, fmt.Errorf("scan scene gate: %w", err)
	}
	gate.Status = session.GateStatus(strings.ToLower(strings.TrimSpace(status)))
	gate.CreatedAt = fromMillis(createdAt)
	gate.ResolvedAt = fromNullMillis(resolvedAt)
	if resolvedByActorType.Valid {
		gate.ResolvedByActorType = resolvedByActorType.String
	}
	if resolvedByActorID.Valid {
		gate.ResolvedByActorID = resolvedByActorID.String
	}
	return gate, nil
}
