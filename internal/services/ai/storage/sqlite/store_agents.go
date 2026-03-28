package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func (s *Store) PutAgent(ctx context.Context, a agent.Agent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if a.ID == "" {
		return fmt.Errorf("agent id is required")
	}
	if a.OwnerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if a.Label == "" {
		return fmt.Errorf("label is required")
	}
	if a.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if a.Model == "" {
		return fmt.Errorf("model is required")
	}
	if a.AuthReference.IsZero() {
		return fmt.Errorf("auth reference is required")
	}
	if a.Status == "" {
		return fmt.Errorf("status is required")
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_agents (
	id, owner_user_id, label, instructions, provider, model, auth_reference_type, auth_reference_id, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	owner_user_id = excluded.owner_user_id,
	label = excluded.label,
	instructions = excluded.instructions,
	provider = excluded.provider,
	model = excluded.model,
	auth_reference_type = excluded.auth_reference_type,
	auth_reference_id = excluded.auth_reference_id,
	status = excluded.status,
	updated_at = excluded.updated_at
`,
		a.ID,
		a.OwnerUserID,
		a.Label,
		a.Instructions,
		string(a.Provider),
		a.Model,
		a.AuthReference.Type(),
		a.AuthReference.ID,
		string(a.Status),
		sqliteutil.ToMillis(a.CreatedAt),
		sqliteutil.ToMillis(a.UpdatedAt),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return storage.ErrConflict
		}
		return fmt.Errorf("put agent: %w", err)
	}
	return nil
}

// GetAgent fetches an agent by ID.
func (s *Store) GetAgent(ctx context.Context, agentID string) (agent.Agent, error) {
	if err := ctx.Err(); err != nil {
		return agent.Agent{}, err
	}
	if s == nil || s.sqlDB == nil {
		return agent.Agent{}, fmt.Errorf("storage is not configured")
	}
	if agentID == "" {
		return agent.Agent{}, fmt.Errorf("agent id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, label, instructions, provider, model, auth_reference_type, auth_reference_id, status, created_at, updated_at
FROM ai_agents
WHERE id = ?
`, agentID)

	a, err := scanAgent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agent.Agent{}, storage.ErrNotFound
		}
		return agent.Agent{}, fmt.Errorf("get agent: %w", err)
	}
	return a, nil
}

// ListAgentsByOwner returns a page of agents for one owner.
func (s *Store) ListAgentsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (agent.Page, error) {
	if err := ctx.Err(); err != nil {
		return agent.Page{}, err
	}
	db, err := requireStoreDB(s)
	if err != nil {
		return agent.Page{}, err
	}
	if ownerUserID == "" {
		return agent.Page{}, fmt.Errorf("owner user id is required")
	}
	limit, err := keysetPageLimit(pageSize)
	if err != nil {
		return agent.Page{}, err
	}

	var rows *sql.Rows
	if pageToken == "" {
		rows, err = db.QueryContext(ctx, `
SELECT id, owner_user_id, label, instructions, provider, model, auth_reference_type, auth_reference_id, status, created_at, updated_at
FROM ai_agents
WHERE owner_user_id = ?
ORDER BY id
LIMIT ?
`, ownerUserID, limit)
	} else {
		rows, err = db.QueryContext(ctx, `
SELECT id, owner_user_id, label, instructions, provider, model, auth_reference_type, auth_reference_id, status, created_at, updated_at
FROM ai_agents
WHERE owner_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, ownerUserID, pageToken, limit)
	}
	if err != nil {
		return agent.Page{}, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	page, err := scanAgentPage(rows, pageSize)
	if err != nil {
		return agent.Page{}, err
	}
	return page, nil
}

// ListAccessibleAgents returns agents the user can invoke: owned agents plus
// agents with approved shared invoke access, in a single UNION query with
// keyset pagination by agent ID.
func (s *Store) ListAccessibleAgents(ctx context.Context, userID string, pageSize int, pageToken string) (agent.Page, error) {
	if err := ctx.Err(); err != nil {
		return agent.Page{}, err
	}
	db, err := requireStoreDB(s)
	if err != nil {
		return agent.Page{}, err
	}
	if userID == "" {
		return agent.Page{}, fmt.Errorf("user id is required")
	}
	limit, err := keysetPageLimit(pageSize)
	if err != nil {
		return agent.Page{}, err
	}

	// UNION deduplicates: an owned agent that also has an access request
	// appears once.
	const baseQuery = `
SELECT id, owner_user_id, label, instructions, provider, model, auth_reference_type, auth_reference_id, status, created_at, updated_at
FROM ai_agents
WHERE owner_user_id = ? AND id > ?
UNION
SELECT a.id, a.owner_user_id, a.label, a.instructions, a.provider, a.model, a.auth_reference_type, a.auth_reference_id, a.status, a.created_at, a.updated_at
FROM ai_agents a
INNER JOIN ai_access_requests ar ON a.id = ar.agent_id AND a.owner_user_id = ar.owner_user_id
WHERE ar.requester_user_id = ? AND ar.scope = 'invoke' AND ar.status = 'approved' AND a.id > ?
ORDER BY id
LIMIT ?`

	cursor := ""
	if pageToken != "" {
		cursor = pageToken
	}

	rows, err := db.QueryContext(ctx, baseQuery, userID, cursor, userID, cursor, limit)
	if err != nil {
		return agent.Page{}, fmt.Errorf("list accessible agents: %w", err)
	}
	defer rows.Close()

	page, err := scanAgentPage(rows, pageSize)
	if err != nil {
		return agent.Page{}, err
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
	if ownerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
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

// scanAgent reconstructs one agent.Agent from a database row.
func scanAgent(s scanner) (agent.Agent, error) {
	var (
		a           agent.Agent
		providerStr string
		authRefType string
		authRefID   string
		statusStr   string
		createdAt   int64
		updatedAt   int64
	)
	if err := s.Scan(
		&a.ID,
		&a.OwnerUserID,
		&a.Label,
		&a.Instructions,
		&providerStr,
		&a.Model,
		&authRefType,
		&authRefID,
		&statusStr,
		&createdAt,
		&updatedAt,
	); err != nil {
		return agent.Agent{}, err
	}
	a.Provider, _ = provider.Normalize(providerStr)
	a.Status = agent.ParseStatus(statusStr)
	a.AuthReference, _ = agent.NormalizeAuthReference(agent.AuthReference{
		Kind: agent.AuthReferenceKind(authRefType),
		ID:   authRefID,
	}, false)
	a.CreatedAt = sqliteutil.FromMillis(createdAt)
	a.UpdatedAt = sqliteutil.FromMillis(updatedAt)
	return a, nil
}

// scanAgentPage scans rows into an agent.Page with keyset pagination.
func scanAgentPage(rows *sql.Rows, pageSize int) (agent.Page, error) {
	agents, nextPageToken, err := scanIDKeysetPage(rows, pageSize, scanAgent, "agent", func(a agent.Agent) string {
		return a.ID
	})
	if err != nil {
		return agent.Page{}, err
	}
	return agent.Page{Agents: agents, NextPageToken: nextPageToken}, nil
}
