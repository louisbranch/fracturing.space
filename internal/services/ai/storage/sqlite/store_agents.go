package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func (s *Store) PutAgent(ctx context.Context, record storage.AgentRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("agent id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(record.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(record.Model) == "" {
		return fmt.Errorf("model is required")
	}
	record.CredentialID = strings.TrimSpace(record.CredentialID)
	record.ProviderGrantID = strings.TrimSpace(record.ProviderGrantID)
	hasCredentialID := record.CredentialID != ""
	hasProviderGrantID := record.ProviderGrantID != ""
	// Persist exactly one auth reference so invocation cannot resolve
	// ambiguous credential sources.
	if hasCredentialID == hasProviderGrantID {
		return fmt.Errorf("exactly one agent auth reference is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_agents (
	id, owner_user_id, name, provider, model, credential_id, provider_grant_id, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	owner_user_id = excluded.owner_user_id,
	name = excluded.name,
	provider = excluded.provider,
	model = excluded.model,
	credential_id = excluded.credential_id,
	provider_grant_id = excluded.provider_grant_id,
	status = excluded.status,
	updated_at = excluded.updated_at
`,
		record.ID,
		record.OwnerUserID,
		record.Name,
		record.Provider,
		record.Model,
		record.CredentialID,
		record.ProviderGrantID,
		record.Status,
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("put agent: %w", err)
	}
	return nil
}

// GetAgent fetches an agent record by ID.
func (s *Store) GetAgent(ctx context.Context, agentID string) (storage.AgentRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.AgentRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AgentRecord{}, fmt.Errorf("storage is not configured")
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return storage.AgentRecord{}, fmt.Errorf("agent id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, name, provider, model, credential_id, provider_grant_id, status, created_at, updated_at
FROM ai_agents
WHERE id = ?
`, agentID)

	var rec storage.AgentRecord
	var createdAt int64
	var updatedAt int64
	if err := row.Scan(
		&rec.ID,
		&rec.OwnerUserID,
		&rec.Name,
		&rec.Provider,
		&rec.Model,
		&rec.CredentialID,
		&rec.ProviderGrantID,
		&rec.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.AgentRecord{}, storage.ErrNotFound
		}
		return storage.AgentRecord{}, fmt.Errorf("get agent: %w", err)
	}
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	return rec, nil
}

// ListAgentsByOwner returns a page of agents for one owner.
func (s *Store) ListAgentsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (storage.AgentPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.AgentPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AgentPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.AgentPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.AgentPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, owner_user_id, name, provider, model, credential_id, provider_grant_id, status, created_at, updated_at
FROM ai_agents
WHERE owner_user_id = ?
ORDER BY id
LIMIT ?
`, ownerUserID, limit)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, owner_user_id, name, provider, model, credential_id, provider_grant_id, status, created_at, updated_at
FROM ai_agents
WHERE owner_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, ownerUserID, strings.TrimSpace(pageToken), limit)
	}
	if err != nil {
		return storage.AgentPage{}, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	page := storage.AgentPage{Agents: make([]storage.AgentRecord, 0, pageSize)}
	for rows.Next() {
		var rec storage.AgentRecord
		var createdAt int64
		var updatedAt int64
		if err := rows.Scan(
			&rec.ID,
			&rec.OwnerUserID,
			&rec.Name,
			&rec.Provider,
			&rec.Model,
			&rec.CredentialID,
			&rec.ProviderGrantID,
			&rec.Status,
			&createdAt,
			&updatedAt,
		); err != nil {
			return storage.AgentPage{}, fmt.Errorf("scan agent row: %w", err)
		}
		rec.CreatedAt = fromMillis(createdAt)
		rec.UpdatedAt = fromMillis(updatedAt)
		page.Agents = append(page.Agents, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.AgentPage{}, fmt.Errorf("iterate agent rows: %w", err)
	}

	if len(page.Agents) > pageSize {
		page.NextPageToken = page.Agents[pageSize-1].ID
		page.Agents = page.Agents[:pageSize]
	}
	return page, nil
}

// DeleteAgent deletes one agent owned by one user.
func (s *Store) DeleteAgent(ctx context.Context, ownerUserID string, agentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return fmt.Errorf("agent id is required")
	}

	res, err := s.sqlDB.ExecContext(ctx, `
DELETE FROM ai_agents
WHERE owner_user_id = ? AND id = ?
`, ownerUserID, agentID)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete agent rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// PutAccessRequest persists an access request record.
