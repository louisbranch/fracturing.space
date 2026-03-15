package coreprojection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

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

	_, err := s.projectionQueryable().ExecContext(ctx,
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
	result, err := s.projectionQueryable().ExecContext(ctx,
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

	row := s.projectionQueryable().QueryRowContext(ctx,
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
		rows, err = s.projectionQueryable().QueryContext(ctx,
			`SELECT campaign_id, scene_id, session_id, name, description, active, created_at, updated_at, ended_at
			 FROM scenes WHERE campaign_id = ? AND session_id = ?
			 ORDER BY scene_id ASC LIMIT ?`,
			campaignID, sessionID, limit,
		)
	} else {
		rows, err = s.projectionQueryable().QueryContext(ctx,
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

	rows, err := s.projectionQueryable().QueryContext(ctx,
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

// ListVisibleActiveScenesForCharacters returns active session scenes visible to one of the characters.
func (s *Store) ListVisibleActiveScenesForCharacters(ctx context.Context, campaignID, sessionID string, characterIDs []string) ([]storage.SceneRecord, error) {
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

	visibleCharacterIDs := make([]string, 0, len(characterIDs))
	for _, characterID := range characterIDs {
		characterID = strings.TrimSpace(characterID)
		if characterID == "" {
			continue
		}
		visibleCharacterIDs = append(visibleCharacterIDs, characterID)
	}
	if len(visibleCharacterIDs) == 0 {
		return []storage.SceneRecord{}, nil
	}

	args := make([]any, 0, len(visibleCharacterIDs)+2)
	args = append(args, campaignID, sessionID)
	placeholders := make([]string, 0, len(visibleCharacterIDs))
	for _, characterID := range visibleCharacterIDs {
		placeholders = append(placeholders, "?")
		args = append(args, characterID)
	}

	rows, err := s.projectionQueryable().QueryContext(ctx,
		fmt.Sprintf(
			`SELECT DISTINCT s.campaign_id, s.scene_id, s.session_id, s.name, s.description, s.active, s.created_at, s.updated_at, s.ended_at
			 FROM scenes s
			 JOIN scene_characters sc
			   ON sc.campaign_id = s.campaign_id
			  AND sc.scene_id = s.scene_id
			 WHERE s.campaign_id = ? AND s.session_id = ? AND s.active = 1
			   AND sc.character_id IN (%s)
			 ORDER BY s.scene_id ASC`,
			strings.Join(placeholders, ","),
		),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list visible active scenes: %w", err)
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
		return nil, fmt.Errorf("list visible active scenes rows: %w", err)
	}
	return result, nil
}

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
