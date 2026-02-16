package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage/sqlite/migrations"
	_ "modernc.org/sqlite"
)

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

func encodeScopes(scopes []string) (string, error) {
	if len(scopes) == 0 {
		return "[]", nil
	}
	encoded, err := json.Marshal(scopes)
	if err != nil {
		return "", fmt.Errorf("marshal scopes: %w", err)
	}
	return string(encoded), nil
}

func decodeScopes(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var scopes []string
	if err := json.Unmarshal([]byte(value), &scopes); err != nil {
		return nil, fmt.Errorf("unmarshal scopes: %w", err)
	}
	return scopes, nil
}

// Store provides SQLite-backed persistence for AI records.
type Store struct {
	sqlDB *sql.DB
}

// DB returns the underlying sql.DB instance.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.sqlDB
}

// Open opens a SQLite store at the provided path.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	cleanPath := filepath.Clean(path)
	dsn := cleanPath + "?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000&_synchronous=NORMAL"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	store := &Store{sqlDB: sqlDB}
	if err := store.runMigrations(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return store, nil
}

// Close closes the underlying SQLite database.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

func (s *Store) runMigrations() error {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	sqlFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		sqlFiles = append(sqlFiles, entry.Name())
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		content, err := fs.ReadFile(migrations.FS, file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}
		upSQL := extractUpMigration(string(content))
		if strings.TrimSpace(upSQL) == "" {
			continue
		}
		if _, err := s.sqlDB.Exec(upSQL); err != nil {
			if !isAlreadyExistsError(err) {
				return fmt.Errorf("exec migration %s: %w", file, err)
			}
		}
	}

	return nil
}

func extractUpMigration(content string) string {
	upIdx := strings.Index(content, "-- +migrate Up")
	if upIdx == -1 {
		return content
	}
	downIdx := strings.Index(content, "-- +migrate Down")
	if downIdx == -1 {
		return content[upIdx+len("-- +migrate Up"):]
	}
	return content[upIdx+len("-- +migrate Up") : downIdx]
}

func isAlreadyExistsError(err error) bool {
	value := strings.ToLower(err.Error())
	return strings.Contains(value, "already exists") || strings.Contains(value, "duplicate column name")
}

// PutCredential persists a credential record.
func (s *Store) PutCredential(ctx context.Context, record storage.CredentialRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("credential id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(record.Label) == "" {
		return fmt.Errorf("label is required")
	}
	if strings.TrimSpace(record.SecretCiphertext) == "" {
		return fmt.Errorf("secret ciphertext is required")
	}
	// SecretCiphertext is expected to already be sealed by the service layer.
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}

	var revokedAt sql.NullInt64
	if record.RevokedAt != nil {
		revokedAt = sql.NullInt64{Int64: toMillis(*record.RevokedAt), Valid: true}
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_credentials (
	id, owner_user_id, provider, label, secret_ciphertext, status, created_at, updated_at, revoked_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	owner_user_id = excluded.owner_user_id,
	provider = excluded.provider,
	label = excluded.label,
	secret_ciphertext = excluded.secret_ciphertext,
	status = excluded.status,
	updated_at = excluded.updated_at,
	revoked_at = excluded.revoked_at
`,
		record.ID,
		record.OwnerUserID,
		record.Provider,
		record.Label,
		record.SecretCiphertext,
		record.Status,
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
		revokedAt,
	)
	if err != nil {
		return fmt.Errorf("put credential: %w", err)
	}
	return nil
}

// GetCredential fetches a credential record by ID.
func (s *Store) GetCredential(ctx context.Context, credentialID string) (storage.CredentialRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.CredentialRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CredentialRecord{}, fmt.Errorf("storage is not configured")
	}
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		return storage.CredentialRecord{}, fmt.Errorf("credential id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, provider, label, secret_ciphertext, status, created_at, updated_at, revoked_at
FROM ai_credentials
WHERE id = ?
`, credentialID)

	var rec storage.CredentialRecord
	var createdAt int64
	var updatedAt int64
	var revokedAt sql.NullInt64
	if err := row.Scan(
		&rec.ID,
		&rec.OwnerUserID,
		&rec.Provider,
		&rec.Label,
		&rec.SecretCiphertext,
		&rec.Status,
		&createdAt,
		&updatedAt,
		&revokedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.CredentialRecord{}, storage.ErrNotFound
		}
		return storage.CredentialRecord{}, fmt.Errorf("get credential: %w", err)
	}

	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if revokedAt.Valid {
		value := fromMillis(revokedAt.Int64)
		rec.RevokedAt = &value
	}
	return rec, nil
}

// ListCredentialsByOwner returns a page of credential records for one owner.
func (s *Store) ListCredentialsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (storage.CredentialPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CredentialPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CredentialPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.CredentialPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.CredentialPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, owner_user_id, provider, label, secret_ciphertext, status, created_at, updated_at, revoked_at
FROM ai_credentials
WHERE owner_user_id = ?
ORDER BY id
LIMIT ?
`, ownerUserID, limit)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, owner_user_id, provider, label, secret_ciphertext, status, created_at, updated_at, revoked_at
FROM ai_credentials
WHERE owner_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, ownerUserID, strings.TrimSpace(pageToken), limit)
	}
	if err != nil {
		return storage.CredentialPage{}, fmt.Errorf("list credentials: %w", err)
	}
	defer rows.Close()

	page := storage.CredentialPage{Credentials: make([]storage.CredentialRecord, 0, pageSize)}
	for rows.Next() {
		var rec storage.CredentialRecord
		var createdAt int64
		var updatedAt int64
		var revokedAt sql.NullInt64
		if err := rows.Scan(
			&rec.ID,
			&rec.OwnerUserID,
			&rec.Provider,
			&rec.Label,
			&rec.SecretCiphertext,
			&rec.Status,
			&createdAt,
			&updatedAt,
			&revokedAt,
		); err != nil {
			return storage.CredentialPage{}, fmt.Errorf("scan credential row: %w", err)
		}
		rec.CreatedAt = fromMillis(createdAt)
		rec.UpdatedAt = fromMillis(updatedAt)
		if revokedAt.Valid {
			value := fromMillis(revokedAt.Int64)
			rec.RevokedAt = &value
		}
		page.Credentials = append(page.Credentials, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.CredentialPage{}, fmt.Errorf("iterate credential rows: %w", err)
	}

	if len(page.Credentials) > pageSize {
		page.NextPageToken = page.Credentials[pageSize-1].ID
		page.Credentials = page.Credentials[:pageSize]
	}
	return page, nil
}

// RevokeCredential marks a credential as revoked.
func (s *Store) RevokeCredential(ctx context.Context, ownerUserID string, credentialID string, revokedAt time.Time) error {
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
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		return fmt.Errorf("credential id is required")
	}

	// Revocation is a lifecycle state change; ciphertext is retained for audit
	// history and is no longer considered usable by service-level checks.
	updatedAt := revokedAt.UTC()
	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_credentials
SET status = 'revoked', updated_at = ?, revoked_at = ?
WHERE owner_user_id = ? AND id = ?
`, toMillis(updatedAt), toMillis(revokedAt.UTC()), ownerUserID, credentialID)
	if err != nil {
		return fmt.Errorf("revoke credential: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke credential rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// PutAgent persists an agent record.
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
func (s *Store) PutAccessRequest(ctx context.Context, record storage.AccessRequestRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("access request id is required")
	}
	if strings.TrimSpace(record.RequesterUserID) == "" {
		return fmt.Errorf("requester user id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.RequesterUserID) == strings.TrimSpace(record.OwnerUserID) {
		return fmt.Errorf("requester user id must differ from owner user id")
	}
	if strings.TrimSpace(record.AgentID) == "" {
		return fmt.Errorf("agent id is required")
	}
	if strings.TrimSpace(record.Scope) == "" {
		return fmt.Errorf("scope is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}

	var reviewedAt sql.NullInt64
	if record.ReviewedAt != nil {
		reviewedAt = sql.NullInt64{Int64: toMillis(*record.ReviewedAt), Valid: true}
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_access_requests (
	id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	requester_user_id = excluded.requester_user_id,
	owner_user_id = excluded.owner_user_id,
	agent_id = excluded.agent_id,
	scope = excluded.scope,
	request_note = excluded.request_note,
	status = excluded.status,
	reviewer_user_id = excluded.reviewer_user_id,
	review_note = excluded.review_note,
	updated_at = excluded.updated_at,
	reviewed_at = excluded.reviewed_at
`,
		record.ID,
		record.RequesterUserID,
		record.OwnerUserID,
		record.AgentID,
		record.Scope,
		strings.TrimSpace(record.RequestNote),
		record.Status,
		strings.TrimSpace(record.ReviewerUserID),
		strings.TrimSpace(record.ReviewNote),
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
		reviewedAt,
	)
	if err != nil {
		return fmt.Errorf("put access request: %w", err)
	}
	return nil
}

// GetAccessRequest fetches an access request record by ID.
func (s *Store) GetAccessRequest(ctx context.Context, accessRequestID string) (storage.AccessRequestRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestRecord{}, fmt.Errorf("storage is not configured")
	}
	accessRequestID = strings.TrimSpace(accessRequestID)
	if accessRequestID == "" {
		return storage.AccessRequestRecord{}, fmt.Errorf("access request id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE id = ?
`, accessRequestID)

	rec, err := scanAccessRequestRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.AccessRequestRecord{}, storage.ErrNotFound
		}
		return storage.AccessRequestRecord{}, fmt.Errorf("get access request: %w", err)
	}
	return rec, nil
}

// ListAccessRequestsByRequester returns a page of access requests by requester.
func (s *Store) ListAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (storage.AccessRequestPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestPage{}, fmt.Errorf("storage is not configured")
	}
	requesterUserID = strings.TrimSpace(requesterUserID)
	if requesterUserID == "" {
		return storage.AccessRequestPage{}, fmt.Errorf("requester user id is required")
	}
	if pageSize <= 0 {
		return storage.AccessRequestPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ?
ORDER BY id
LIMIT ?
`, requesterUserID, limit)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, requesterUserID, strings.TrimSpace(pageToken), limit)
	}
	if err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("list access requests by requester: %w", err)
	}
	defer rows.Close()

	page := storage.AccessRequestPage{AccessRequests: make([]storage.AccessRequestRecord, 0, pageSize)}
	for rows.Next() {
		rec, err := scanAccessRequestRows(rows)
		if err != nil {
			return storage.AccessRequestPage{}, fmt.Errorf("scan access request row: %w", err)
		}
		page.AccessRequests = append(page.AccessRequests, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("iterate access request rows: %w", err)
	}
	if len(page.AccessRequests) > pageSize {
		page.NextPageToken = page.AccessRequests[pageSize-1].ID
		page.AccessRequests = page.AccessRequests[:pageSize]
	}
	return page, nil
}

// ListAccessRequestsByOwner returns a page of access requests by owner.
func (s *Store) ListAccessRequestsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (storage.AccessRequestPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.AccessRequestPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.AccessRequestPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE owner_user_id = ?
ORDER BY id
LIMIT ?
`, ownerUserID, limit)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE owner_user_id = ? AND id > ?
ORDER BY id
LIMIT ?
`, ownerUserID, strings.TrimSpace(pageToken), limit)
	}
	if err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("list access requests by owner: %w", err)
	}
	defer rows.Close()

	page := storage.AccessRequestPage{AccessRequests: make([]storage.AccessRequestRecord, 0, pageSize)}
	for rows.Next() {
		rec, err := scanAccessRequestRows(rows)
		if err != nil {
			return storage.AccessRequestPage{}, fmt.Errorf("scan access request row: %w", err)
		}
		page.AccessRequests = append(page.AccessRequests, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("iterate access request rows: %w", err)
	}
	if len(page.AccessRequests) > pageSize {
		page.NextPageToken = page.AccessRequests[pageSize-1].ID
		page.AccessRequests = page.AccessRequests[:pageSize]
	}
	return page, nil
}

// GetApprovedInvokeAccessByRequesterForAgent returns one approved invoke access
// request for a requester/owner/agent tuple.
func (s *Store) GetApprovedInvokeAccessByRequesterForAgent(ctx context.Context, requesterUserID string, ownerUserID string, agentID string) (storage.AccessRequestRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestRecord{}, fmt.Errorf("storage is not configured")
	}
	requesterUserID = strings.TrimSpace(requesterUserID)
	if requesterUserID == "" {
		return storage.AccessRequestRecord{}, fmt.Errorf("requester user id is required")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.AccessRequestRecord{}, fmt.Errorf("owner user id is required")
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return storage.AccessRequestRecord{}, fmt.Errorf("agent id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ? AND owner_user_id = ? AND agent_id = ? AND scope = 'invoke' AND status = 'approved'
ORDER BY id
LIMIT 1
`, requesterUserID, ownerUserID, agentID)

	rec, err := scanAccessRequestRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.AccessRequestRecord{}, storage.ErrNotFound
		}
		return storage.AccessRequestRecord{}, fmt.Errorf("get approved invoke access request: %w", err)
	}
	return rec, nil
}

// ListApprovedInvokeAccessRequestsByRequester returns approved invoke access
// requests for one requester.
func (s *Store) ListApprovedInvokeAccessRequestsByRequester(ctx context.Context, requesterUserID string, pageSize int, pageToken string) (storage.AccessRequestPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.AccessRequestPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AccessRequestPage{}, fmt.Errorf("storage is not configured")
	}
	requesterUserID = strings.TrimSpace(requesterUserID)
	if requesterUserID == "" {
		return storage.AccessRequestPage{}, fmt.Errorf("requester user id is required")
	}
	if pageSize <= 0 {
		return storage.AccessRequestPage{}, fmt.Errorf("page size must be greater than zero")
	}

	limit := pageSize + 1
	var (
		rows *sql.Rows
		err  error
	)
	if strings.TrimSpace(pageToken) == "" {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ? AND scope = 'invoke' AND status = 'approved'
ORDER BY id
LIMIT ?
`, requesterUserID, limit)
	} else {
		rows, err = s.sqlDB.QueryContext(ctx, `
SELECT id, requester_user_id, owner_user_id, agent_id, scope, request_note, status, reviewer_user_id, review_note, created_at, updated_at, reviewed_at
FROM ai_access_requests
WHERE requester_user_id = ? AND scope = 'invoke' AND status = 'approved' AND id > ?
ORDER BY id
LIMIT ?
`, requesterUserID, strings.TrimSpace(pageToken), limit)
	}
	if err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("list approved invoke access requests by requester: %w", err)
	}
	defer rows.Close()

	page := storage.AccessRequestPage{AccessRequests: make([]storage.AccessRequestRecord, 0, pageSize)}
	for rows.Next() {
		rec, err := scanAccessRequestRows(rows)
		if err != nil {
			return storage.AccessRequestPage{}, fmt.Errorf("scan access request row: %w", err)
		}
		page.AccessRequests = append(page.AccessRequests, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.AccessRequestPage{}, fmt.Errorf("iterate access request rows: %w", err)
	}
	if len(page.AccessRequests) > pageSize {
		page.NextPageToken = page.AccessRequests[pageSize-1].ID
		page.AccessRequests = page.AccessRequests[:pageSize]
	}
	return page, nil
}

// ReviewAccessRequest applies an owner review decision for one pending request.
func (s *Store) ReviewAccessRequest(ctx context.Context, ownerUserID string, accessRequestID string, status string, reviewerUserID string, reviewNote string, reviewedAt time.Time) error {
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
	accessRequestID = strings.TrimSpace(accessRequestID)
	if accessRequestID == "" {
		return fmt.Errorf("access request id is required")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return fmt.Errorf("status is required")
	}
	reviewerUserID = strings.TrimSpace(reviewerUserID)
	if reviewerUserID == "" {
		return fmt.Errorf("reviewer user id is required")
	}
	// Owner-scoped reviews must attribute the decision to the same owner.
	if reviewerUserID != ownerUserID {
		return fmt.Errorf("reviewer user id must match owner user id")
	}

	var existingStatus string
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT status
FROM ai_access_requests
WHERE owner_user_id = ? AND id = ?
`, ownerUserID, accessRequestID)
	if err := row.Scan(&existingStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("check access request status: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(existingStatus), "pending") {
		return storage.ErrConflict
	}

	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_access_requests
SET status = ?, reviewer_user_id = ?, review_note = ?, reviewed_at = ?, updated_at = ?
WHERE owner_user_id = ? AND id = ? AND status = 'pending'
`, status, reviewerUserID, strings.TrimSpace(reviewNote), toMillis(reviewedAt.UTC()), toMillis(reviewedAt.UTC()), ownerUserID, accessRequestID)
	if err != nil {
		return fmt.Errorf("review access request: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("review access request rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrConflict
	}
	return nil
}

// RevokeAccessRequest applies an owner revocation for one approved request.
func (s *Store) RevokeAccessRequest(ctx context.Context, ownerUserID string, accessRequestID string, status string, reviewerUserID string, reviewNote string, revokedAt time.Time) error {
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
	accessRequestID = strings.TrimSpace(accessRequestID)
	if accessRequestID == "" {
		return fmt.Errorf("access request id is required")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return fmt.Errorf("status is required")
	}
	reviewerUserID = strings.TrimSpace(reviewerUserID)
	if reviewerUserID == "" {
		return fmt.Errorf("reviewer user id is required")
	}
	// Owner-scoped revocations must attribute the action to the same owner.
	if reviewerUserID != ownerUserID {
		return fmt.Errorf("reviewer user id must match owner user id")
	}

	var existingStatus string
	row := s.sqlDB.QueryRowContext(ctx, `
SELECT status
FROM ai_access_requests
WHERE owner_user_id = ? AND id = ?
`, ownerUserID, accessRequestID)
	if err := row.Scan(&existingStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("check access request status: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(existingStatus), "approved") {
		return storage.ErrConflict
	}

	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_access_requests
SET status = ?, reviewer_user_id = ?, review_note = ?, updated_at = ?
WHERE owner_user_id = ? AND id = ? AND status = 'approved'
`, status, reviewerUserID, strings.TrimSpace(reviewNote), toMillis(revokedAt.UTC()), ownerUserID, accessRequestID)
	if err != nil {
		return fmt.Errorf("revoke access request: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke access request rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrConflict
	}
	return nil
}

// PutAuditEvent appends one AI audit event row.
func (s *Store) PutAuditEvent(ctx context.Context, record storage.AuditEventRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.EventName) == "" {
		return fmt.Errorf("event name is required")
	}
	if strings.TrimSpace(record.ActorUserID) == "" {
		return fmt.Errorf("actor user id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.RequesterUserID) == "" {
		return fmt.Errorf("requester user id is required")
	}
	if strings.TrimSpace(record.AgentID) == "" {
		return fmt.Errorf("agent id is required")
	}
	if strings.TrimSpace(record.AccessRequestID) == "" {
		return fmt.Errorf("access request id is required")
	}
	if strings.TrimSpace(record.Outcome) == "" {
		return fmt.Errorf("outcome is required")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("created at is required")
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_audit_events (
	event_name, actor_user_id, owner_user_id, requester_user_id, agent_id, access_request_id, outcome, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
		strings.TrimSpace(record.EventName),
		strings.TrimSpace(record.ActorUserID),
		strings.TrimSpace(record.OwnerUserID),
		strings.TrimSpace(record.RequesterUserID),
		strings.TrimSpace(record.AgentID),
		strings.TrimSpace(record.AccessRequestID),
		strings.TrimSpace(record.Outcome),
		toMillis(record.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("put audit event: %w", err)
	}
	return nil
}

// ListAuditEventsByOwner returns a page of audit events scoped to one owner.
func (s *Store) ListAuditEventsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter storage.AuditEventFilter) (storage.AuditEventPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.AuditEventPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.AuditEventPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.AuditEventPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.AuditEventPage{}, fmt.Errorf("page size must be greater than zero")
	}
	eventName := strings.TrimSpace(filter.EventName)
	agentID := strings.TrimSpace(filter.AgentID)

	var (
		createdAfterMillis  *int64
		createdBeforeMillis *int64
	)
	if filter.CreatedAfter != nil {
		value := toMillis(filter.CreatedAfter.UTC())
		createdAfterMillis = &value
	}
	if filter.CreatedBefore != nil {
		value := toMillis(filter.CreatedBefore.UTC())
		createdBeforeMillis = &value
	}
	if createdAfterMillis != nil && createdBeforeMillis != nil && *createdAfterMillis > *createdBeforeMillis {
		return storage.AuditEventPage{}, fmt.Errorf("created_after must be before or equal to created_before")
	}

	limit := pageSize + 1
	pageToken = strings.TrimSpace(pageToken)
	whereParts := []string{"owner_user_id = ?"}
	args := []any{ownerUserID}
	if eventName != "" {
		whereParts = append(whereParts, "event_name = ?")
		args = append(args, eventName)
	}
	if agentID != "" {
		whereParts = append(whereParts, "agent_id = ?")
		args = append(args, agentID)
	}
	if createdAfterMillis != nil {
		whereParts = append(whereParts, "created_at >= ?")
		args = append(args, *createdAfterMillis)
	}
	if createdBeforeMillis != nil {
		whereParts = append(whereParts, "created_at <= ?")
		args = append(args, *createdBeforeMillis)
	}
	if pageToken != "" {
		tokenValue, parseErr := strconv.ParseInt(pageToken, 10, 64)
		if parseErr != nil || tokenValue < 0 {
			return storage.AuditEventPage{}, fmt.Errorf("invalid page token")
		}
		whereParts = append(whereParts, "id > ?")
		args = append(args, tokenValue)
	}
	args = append(args, limit)

	// Owner scope is always included in WHERE before optional filters so callers
	// can only narrow their own visibility, never expand to another tenant.
	query := fmt.Sprintf(`
SELECT id, event_name, actor_user_id, owner_user_id, requester_user_id, agent_id, access_request_id, outcome, created_at
FROM ai_audit_events
WHERE %s
ORDER BY id
LIMIT ?
`, strings.Join(whereParts, " AND "))
	rows, err := s.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return storage.AuditEventPage{}, fmt.Errorf("list audit events by owner: %w", err)
	}
	defer rows.Close()

	page := storage.AuditEventPage{AuditEvents: make([]storage.AuditEventRecord, 0, pageSize)}
	for rows.Next() {
		var (
			idValue      int64
			eventName    string
			actorUserID  string
			ownerUser    string
			requesterID  string
			agentID      string
			requestID    string
			outcome      string
			createdAtRaw int64
		)
		if err := rows.Scan(&idValue, &eventName, &actorUserID, &ownerUser, &requesterID, &agentID, &requestID, &outcome, &createdAtRaw); err != nil {
			return storage.AuditEventPage{}, fmt.Errorf("scan audit event row: %w", err)
		}
		page.AuditEvents = append(page.AuditEvents, storage.AuditEventRecord{
			ID:              strconv.FormatInt(idValue, 10),
			EventName:       eventName,
			ActorUserID:     actorUserID,
			OwnerUserID:     ownerUser,
			RequesterUserID: requesterID,
			AgentID:         agentID,
			AccessRequestID: requestID,
			Outcome:         outcome,
			CreatedAt:       fromMillis(createdAtRaw),
		})
	}
	if err := rows.Err(); err != nil {
		return storage.AuditEventPage{}, fmt.Errorf("iterate audit event rows: %w", err)
	}
	if len(page.AuditEvents) > pageSize {
		page.NextPageToken = page.AuditEvents[pageSize-1].ID
		page.AuditEvents = page.AuditEvents[:pageSize]
	}
	return page, nil
}

// PutProviderGrant persists a provider grant record.
func (s *Store) PutProviderGrant(ctx context.Context, record storage.ProviderGrantRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("provider grant id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(record.TokenCiphertext) == "" {
		return fmt.Errorf("token ciphertext is required")
	}
	// TokenCiphertext must be pre-sealed by the service layer.
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}
	scopesJSON, err := encodeScopes(record.GrantedScopes)
	if err != nil {
		return err
	}

	var revokedAt sql.NullInt64
	if record.RevokedAt != nil {
		revokedAt = sql.NullInt64{Int64: toMillis(*record.RevokedAt), Valid: true}
	}
	var expiresAt sql.NullInt64
	if record.ExpiresAt != nil {
		expiresAt = sql.NullInt64{Int64: toMillis(*record.ExpiresAt), Valid: true}
	}
	var lastRefreshedAt sql.NullInt64
	if record.LastRefreshedAt != nil {
		lastRefreshedAt = sql.NullInt64{Int64: toMillis(*record.LastRefreshedAt), Valid: true}
	}

	_, err = s.sqlDB.ExecContext(ctx, `
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
		record.ID,
		record.OwnerUserID,
		record.Provider,
		scopesJSON,
		record.TokenCiphertext,
		record.RefreshSupported,
		record.Status,
		record.LastRefreshError,
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
		revokedAt,
		expiresAt,
		lastRefreshedAt,
	)
	if err != nil {
		return fmt.Errorf("put provider grant: %w", err)
	}
	return nil
}

// GetProviderGrant fetches a provider grant record by ID.
func (s *Store) GetProviderGrant(ctx context.Context, providerGrantID string) (storage.ProviderGrantRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("storage is not configured")
	}
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider grant id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, provider, granted_scopes, token_ciphertext, refresh_supported, status, last_refresh_error, created_at, updated_at, revoked_at, expires_at, last_refreshed_at
FROM ai_provider_grants
WHERE id = ?
`, providerGrantID)

	rec, err := scanProviderGrantRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ProviderGrantRecord{}, storage.ErrNotFound
		}
		return storage.ProviderGrantRecord{}, fmt.Errorf("get provider grant: %w", err)
	}
	return rec, nil
}

// ListProviderGrantsByOwner returns a page of provider grants for one owner.
func (s *Store) ListProviderGrantsByOwner(ctx context.Context, ownerUserID string, pageSize int, pageToken string, filter storage.ProviderGrantFilter) (storage.ProviderGrantPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.ProviderGrantPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ProviderGrantPage{}, fmt.Errorf("storage is not configured")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.ProviderGrantPage{}, fmt.Errorf("owner user id is required")
	}
	if pageSize <= 0 {
		return storage.ProviderGrantPage{}, fmt.Errorf("page size must be greater than zero")
	}
	provider := strings.ToLower(strings.TrimSpace(filter.Provider))
	status := strings.ToLower(strings.TrimSpace(filter.Status))

	limit := pageSize + 1
	pageToken = strings.TrimSpace(pageToken)
	whereParts := []string{"owner_user_id = ?"}
	args := []any{ownerUserID}
	if provider != "" {
		whereParts = append(whereParts, "provider = ?")
		args = append(args, provider)
	}
	if status != "" {
		whereParts = append(whereParts, "status = ?")
		args = append(args, status)
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
	rows, err := s.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return storage.ProviderGrantPage{}, fmt.Errorf("list provider grants: %w", err)
	}
	defer rows.Close()

	page := storage.ProviderGrantPage{ProviderGrants: make([]storage.ProviderGrantRecord, 0, pageSize)}
	for rows.Next() {
		rec, err := scanProviderGrantRows(rows)
		if err != nil {
			return storage.ProviderGrantPage{}, fmt.Errorf("scan provider grant row: %w", err)
		}
		page.ProviderGrants = append(page.ProviderGrants, rec)
	}
	if err := rows.Err(); err != nil {
		return storage.ProviderGrantPage{}, fmt.Errorf("iterate provider grant rows: %w", err)
	}

	if len(page.ProviderGrants) > pageSize {
		page.NextPageToken = page.ProviderGrants[pageSize-1].ID
		page.ProviderGrants = page.ProviderGrants[:pageSize]
	}
	return page, nil
}

// RevokeProviderGrant marks a provider grant as revoked.
func (s *Store) RevokeProviderGrant(ctx context.Context, ownerUserID string, providerGrantID string, revokedAt time.Time) error {
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
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return fmt.Errorf("provider grant id is required")
	}

	updatedAt := revokedAt.UTC()
	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_provider_grants
SET status = 'revoked', updated_at = ?, revoked_at = ?
WHERE owner_user_id = ? AND id = ?
`, toMillis(updatedAt), toMillis(revokedAt.UTC()), ownerUserID, providerGrantID)
	if err != nil {
		return fmt.Errorf("revoke provider grant: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke provider grant rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// UpdateProviderGrantToken updates token ciphertext and refresh metadata.
func (s *Store) UpdateProviderGrantToken(ctx context.Context, ownerUserID string, providerGrantID string, tokenCiphertext string, refreshedAt time.Time, expiresAt *time.Time, status string, lastRefreshError string) error {
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
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return fmt.Errorf("provider grant id is required")
	}
	tokenCiphertext = strings.TrimSpace(tokenCiphertext)
	if tokenCiphertext == "" {
		return fmt.Errorf("token ciphertext is required")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return fmt.Errorf("status is required")
	}

	var expiresAtValue sql.NullInt64
	if expiresAt != nil {
		expiresAtValue = sql.NullInt64{Int64: toMillis(expiresAt.UTC()), Valid: true}
	}
	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_provider_grants
SET token_ciphertext = ?, status = ?, last_refresh_error = ?, updated_at = ?, expires_at = ?, last_refreshed_at = ?
WHERE owner_user_id = ? AND id = ?
`, tokenCiphertext, status, strings.TrimSpace(lastRefreshError), toMillis(refreshedAt.UTC()), expiresAtValue, toMillis(refreshedAt.UTC()), ownerUserID, providerGrantID)
	if err != nil {
		return fmt.Errorf("update provider grant token: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update provider grant token rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// PutProviderConnectSession persists a provider connect session record.
func (s *Store) PutProviderConnectSession(ctx context.Context, record storage.ProviderConnectSessionRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("connect session id is required")
	}
	if strings.TrimSpace(record.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(record.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("status is required")
	}
	if strings.TrimSpace(record.StateHash) == "" {
		return fmt.Errorf("state hash is required")
	}
	if strings.TrimSpace(record.CodeVerifierCiphertext) == "" {
		return fmt.Errorf("code verifier ciphertext is required")
	}
	if record.ExpiresAt.IsZero() {
		return fmt.Errorf("expires at is required")
	}
	scopesJSON, err := encodeScopes(record.RequestedScopes)
	if err != nil {
		return err
	}
	var completedAt sql.NullInt64
	if record.CompletedAt != nil {
		completedAt = sql.NullInt64{Int64: toMillis(*record.CompletedAt), Valid: true}
	}

	_, err = s.sqlDB.ExecContext(ctx, `
INSERT INTO ai_provider_connect_sessions (
	id, owner_user_id, provider, status, requested_scopes, state_hash, code_verifier_ciphertext, created_at, updated_at, expires_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	owner_user_id = excluded.owner_user_id,
	provider = excluded.provider,
	status = excluded.status,
	requested_scopes = excluded.requested_scopes,
	state_hash = excluded.state_hash,
	code_verifier_ciphertext = excluded.code_verifier_ciphertext,
	updated_at = excluded.updated_at,
	expires_at = excluded.expires_at,
	completed_at = excluded.completed_at
`,
		record.ID,
		record.OwnerUserID,
		record.Provider,
		record.Status,
		scopesJSON,
		record.StateHash,
		record.CodeVerifierCiphertext,
		toMillis(record.CreatedAt),
		toMillis(record.UpdatedAt),
		toMillis(record.ExpiresAt),
		completedAt,
	)
	if err != nil {
		return fmt.Errorf("put provider connect session: %w", err)
	}
	return nil
}

// GetProviderConnectSession fetches one provider connect session by ID.
func (s *Store) GetProviderConnectSession(ctx context.Context, connectSessionID string) (storage.ProviderConnectSessionRecord, error) {
	if err := ctx.Err(); err != nil {
		return storage.ProviderConnectSessionRecord{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ProviderConnectSessionRecord{}, fmt.Errorf("storage is not configured")
	}
	connectSessionID = strings.TrimSpace(connectSessionID)
	if connectSessionID == "" {
		return storage.ProviderConnectSessionRecord{}, fmt.Errorf("connect session id is required")
	}

	row := s.sqlDB.QueryRowContext(ctx, `
SELECT id, owner_user_id, provider, status, requested_scopes, state_hash, code_verifier_ciphertext, created_at, updated_at, expires_at, completed_at
FROM ai_provider_connect_sessions
WHERE id = ?
`, connectSessionID)

	rec, err := scanProviderConnectSessionRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ProviderConnectSessionRecord{}, storage.ErrNotFound
		}
		return storage.ProviderConnectSessionRecord{}, fmt.Errorf("get provider connect session: %w", err)
	}
	return rec, nil
}

// CompleteProviderConnectSession marks one connect session as completed.
func (s *Store) CompleteProviderConnectSession(ctx context.Context, ownerUserID string, connectSessionID string, completedAt time.Time) error {
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
	connectSessionID = strings.TrimSpace(connectSessionID)
	if connectSessionID == "" {
		return fmt.Errorf("connect session id is required")
	}

	updatedAt := completedAt.UTC()
	res, err := s.sqlDB.ExecContext(ctx, `
UPDATE ai_provider_connect_sessions
SET status = 'completed', updated_at = ?, completed_at = ?
WHERE owner_user_id = ? AND id = ? AND status = 'pending'
`, toMillis(updatedAt), toMillis(completedAt.UTC()), ownerUserID, connectSessionID)
	if err != nil {
		return fmt.Errorf("complete provider connect session: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("complete provider connect session rows affected: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func scanProviderGrantRow(row *sql.Row) (storage.ProviderGrantRecord, error) {
	var (
		rec              storage.ProviderGrantRecord
		grantedScopesRaw string
		createdAt        int64
		updatedAt        int64
		revokedAt        sql.NullInt64
		expiresAt        sql.NullInt64
		lastRefreshedAt  sql.NullInt64
	)
	if err := row.Scan(
		&rec.ID,
		&rec.OwnerUserID,
		&rec.Provider,
		&grantedScopesRaw,
		&rec.TokenCiphertext,
		&rec.RefreshSupported,
		&rec.Status,
		&rec.LastRefreshError,
		&createdAt,
		&updatedAt,
		&revokedAt,
		&expiresAt,
		&lastRefreshedAt,
	); err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	scopes, err := decodeScopes(grantedScopesRaw)
	if err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	rec.GrantedScopes = scopes
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if revokedAt.Valid {
		value := fromMillis(revokedAt.Int64)
		rec.RevokedAt = &value
	}
	if expiresAt.Valid {
		value := fromMillis(expiresAt.Int64)
		rec.ExpiresAt = &value
	}
	if lastRefreshedAt.Valid {
		value := fromMillis(lastRefreshedAt.Int64)
		rec.LastRefreshedAt = &value
	}
	return rec, nil
}

func scanProviderConnectSessionRow(row *sql.Row) (storage.ProviderConnectSessionRecord, error) {
	var (
		rec                storage.ProviderConnectSessionRecord
		requestedScopesRaw string
		createdAt          int64
		updatedAt          int64
		expiresAt          int64
		completedAt        sql.NullInt64
	)
	if err := row.Scan(
		&rec.ID,
		&rec.OwnerUserID,
		&rec.Provider,
		&rec.Status,
		&requestedScopesRaw,
		&rec.StateHash,
		&rec.CodeVerifierCiphertext,
		&createdAt,
		&updatedAt,
		&expiresAt,
		&completedAt,
	); err != nil {
		return storage.ProviderConnectSessionRecord{}, err
	}
	scopes, err := decodeScopes(requestedScopesRaw)
	if err != nil {
		return storage.ProviderConnectSessionRecord{}, err
	}
	rec.RequestedScopes = scopes
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	rec.ExpiresAt = fromMillis(expiresAt)
	if completedAt.Valid {
		value := fromMillis(completedAt.Int64)
		rec.CompletedAt = &value
	}
	return rec, nil
}

func scanProviderGrantRows(rows *sql.Rows) (storage.ProviderGrantRecord, error) {
	var (
		rec              storage.ProviderGrantRecord
		grantedScopesRaw string
		createdAt        int64
		updatedAt        int64
		revokedAt        sql.NullInt64
		expiresAt        sql.NullInt64
		lastRefreshedAt  sql.NullInt64
	)
	if err := rows.Scan(
		&rec.ID,
		&rec.OwnerUserID,
		&rec.Provider,
		&grantedScopesRaw,
		&rec.TokenCiphertext,
		&rec.RefreshSupported,
		&rec.Status,
		&rec.LastRefreshError,
		&createdAt,
		&updatedAt,
		&revokedAt,
		&expiresAt,
		&lastRefreshedAt,
	); err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	scopes, err := decodeScopes(grantedScopesRaw)
	if err != nil {
		return storage.ProviderGrantRecord{}, err
	}
	rec.GrantedScopes = scopes
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if revokedAt.Valid {
		value := fromMillis(revokedAt.Int64)
		rec.RevokedAt = &value
	}
	if expiresAt.Valid {
		value := fromMillis(expiresAt.Int64)
		rec.ExpiresAt = &value
	}
	if lastRefreshedAt.Valid {
		value := fromMillis(lastRefreshedAt.Int64)
		rec.LastRefreshedAt = &value
	}
	return rec, nil
}

func scanAccessRequestRow(row *sql.Row) (storage.AccessRequestRecord, error) {
	var (
		rec        storage.AccessRequestRecord
		createdAt  int64
		updatedAt  int64
		reviewedAt sql.NullInt64
	)
	if err := row.Scan(
		&rec.ID,
		&rec.RequesterUserID,
		&rec.OwnerUserID,
		&rec.AgentID,
		&rec.Scope,
		&rec.RequestNote,
		&rec.Status,
		&rec.ReviewerUserID,
		&rec.ReviewNote,
		&createdAt,
		&updatedAt,
		&reviewedAt,
	); err != nil {
		return storage.AccessRequestRecord{}, err
	}
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if reviewedAt.Valid {
		value := fromMillis(reviewedAt.Int64)
		rec.ReviewedAt = &value
	}
	return rec, nil
}

func scanAccessRequestRows(rows *sql.Rows) (storage.AccessRequestRecord, error) {
	var (
		rec        storage.AccessRequestRecord
		createdAt  int64
		updatedAt  int64
		reviewedAt sql.NullInt64
	)
	if err := rows.Scan(
		&rec.ID,
		&rec.RequesterUserID,
		&rec.OwnerUserID,
		&rec.AgentID,
		&rec.Scope,
		&rec.RequestNote,
		&rec.Status,
		&rec.ReviewerUserID,
		&rec.ReviewNote,
		&createdAt,
		&updatedAt,
		&reviewedAt,
	); err != nil {
		return storage.AccessRequestRecord{}, err
	}
	rec.CreatedAt = fromMillis(createdAt)
	rec.UpdatedAt = fromMillis(updatedAt)
	if reviewedAt.Valid {
		value := fromMillis(reviewedAt.Int64)
		rec.ReviewedAt = &value
	}
	return rec, nil
}
