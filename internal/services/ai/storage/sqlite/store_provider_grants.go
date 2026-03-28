package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func (s *Store) PutProviderGrant(ctx context.Context, grant providergrant.ProviderGrant) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if err := putProviderGrant(ctx, s.sqlDB, grant); err != nil {
		return err
	}
	return nil
}

func putProviderGrant(ctx context.Context, execer sqlExecer, grant providergrant.ProviderGrant) error {
	if grant.ID == "" {
		return fmt.Errorf("provider grant id is required")
	}
	if grant.OwnerUserID == "" {
		return fmt.Errorf("owner user id is required")
	}
	if grant.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if grant.TokenCiphertext == "" {
		return fmt.Errorf("token ciphertext is required")
	}
	if grant.Status == "" {
		return fmt.Errorf("status is required")
	}
	scopesJSON, err := encodeScopes(grant.GrantedScopes)
	if err != nil {
		return err
	}

	var revokedAt sql.NullInt64
	if grant.RevokedAt != nil {
		revokedAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*grant.RevokedAt), Valid: true}
	}
	var expiresAt sql.NullInt64
	if grant.ExpiresAt != nil {
		expiresAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*grant.ExpiresAt), Valid: true}
	}
	var lastRefreshedAt sql.NullInt64
	if grant.RefreshedAt != nil {
		lastRefreshedAt = sql.NullInt64{Int64: sqliteutil.ToMillis(*grant.RefreshedAt), Valid: true}
	}

	_, err = execer.ExecContext(ctx, `
INSERT INTO ai_provider_grants (
	id, owner_user_id, provider, granted_scopes, token_ciphertext, refresh_supported, status, last_refresh_error, created_at, updated_at, revoked_at, expires_at, last_refreshed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	owner_user_id = excluded.owner_user_id,
	provider = excluded.provider,
	granted_scopes = excluded.granted_scopes,
	token_ciphertext = excluded.token_ciphertext,
	refresh_supported = excluded.refresh_supported,
	status = excluded.status,
	last_refresh_error = excluded.last_refresh_error,
	updated_at = excluded.updated_at,
	revoked_at = excluded.revoked_at,
	expires_at = excluded.expires_at,
	last_refreshed_at = excluded.last_refreshed_at
`,
		grant.ID,
		grant.OwnerUserID,
		string(grant.Provider),
		scopesJSON,
		grant.TokenCiphertext,
		grant.RefreshSupported,
		string(grant.Status),
		grant.LastRefreshError,
		sqliteutil.ToMillis(grant.CreatedAt),
		sqliteutil.ToMillis(grant.UpdatedAt),
		revokedAt,
		expiresAt,
		lastRefreshedAt,
	)
	if err != nil {
		return fmt.Errorf("put provider grant: %w", err)
	}
	return nil
}

// GetProviderGrant fetches a provider grant by ID.
func (s *Store) GetProviderGrant(ctx context.Context, providerGrantID string) (providergrant.ProviderGrant, error) {
	if err := ctx.Err(); err != nil {
		return providergrant.ProviderGrant{}, err
	}
	if s == nil || s.sqlDB == nil {
		return providergrant.ProviderGrant{}, fmt.Errorf("storage is not configured")
	}
	if providerGrantID == "" {
		return providergrant.ProviderGrant{}, fmt.Errorf("provider grant id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, provider, granted_scopes, token_ciphertext, refresh_supported, status, last_refresh_error, created_at, updated_at, revoked_at, expires_at, last_refreshed_at
FROM ai_provider_grants
WHERE id = ?
`, providerGrantID)

	grant, err := scanProviderGrant(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return providergrant.ProviderGrant{}, storage.ErrNotFound
		}
		return providergrant.ProviderGrant{}, fmt.Errorf("get provider grant: %w", err)
	}
	return grant, nil
}

// ListProviderGrantsByOwner returns a page of provider grants for one owner.
func (s *Store) ListProviderGrantsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter providergrant.Filter) (providergrant.Page, error) {
	if err := ctx.Err(); err != nil {
		return providergrant.Page{}, err
	}
	db, err := requireStoreDB(s)
	if err != nil {
		return providergrant.Page{}, err
	}
	if ownerUserID == "" {
		return providergrant.Page{}, fmt.Errorf("owner user id is required")
	}
	limit, err := keysetPageLimit(pageSize)
	if err != nil {
		return providergrant.Page{}, err
	}
	filterProvider := strings.ToLower(string(filter.Provider))
	filterStatus := strings.ToLower(string(filter.Status))

	whereParts := []string{"owner_user_id = ?"}
	args := []any{ownerUserID}
	if filterProvider != "" {
		whereParts = append(whereParts, "provider = ?")
		args = append(args, filterProvider)
	}
	if filterStatus != "" {
		whereParts = append(whereParts, "status = ?")
		args = append(args, filterStatus)
	}
	if pageToken != "" {
		whereParts = append(whereParts, "id > ?")
		args = append(args, pageToken)
	}
	args = append(args, limit)

	// Owner scope is always anchored in WHERE before optional filters so caller
	// input can only narrow visibility for that owner.
	query := fmt.Sprintf(`
SELECT id, owner_user_id, provider, granted_scopes, token_ciphertext, refresh_supported, status, last_refresh_error, created_at, updated_at, revoked_at, expires_at, last_refreshed_at
FROM ai_provider_grants
WHERE %s
ORDER BY id
LIMIT ?
`, strings.Join(whereParts, " AND "))
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return providergrant.Page{}, fmt.Errorf("list provider grants: %w", err)
	}
	defer rows.Close()

	providerGrants, nextPageToken, err := scanIDKeysetPage(rows, pageSize, scanProviderGrant, "provider grant", func(grant providergrant.ProviderGrant) string {
		return grant.ID
	})
	if err != nil {
		return providergrant.Page{}, err
	}
	return providergrant.Page{ProviderGrants: providerGrants, NextPageToken: nextPageToken}, nil
}

// PutProviderConnectSession persists a provider connect session record.
