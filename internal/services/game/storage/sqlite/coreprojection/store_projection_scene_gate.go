package coreprojection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

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

	_, err := s.projectionQueryable().ExecContext(ctx,
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

	row := s.projectionQueryable().QueryRowContext(ctx,
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

	row := s.projectionQueryable().QueryRowContext(ctx,
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
