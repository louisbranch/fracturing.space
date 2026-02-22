package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite/db"
)

func (s *Store) PutPasskeyCredential(ctx context.Context, credential storage.PasskeyCredential) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(credential.CredentialID) == "" {
		return fmt.Errorf("credential id is required")
	}
	if strings.TrimSpace(credential.UserID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(credential.CredentialJSON) == "" {
		return fmt.Errorf("credential json is required")
	}

	lastUsed := sql.NullInt64{}
	if credential.LastUsedAt != nil {
		lastUsed = sql.NullInt64{Int64: toMillis(*credential.LastUsedAt), Valid: true}
	}

	return s.q.PutPasskey(ctx, db.PutPasskeyParams{
		CredentialID:   credential.CredentialID,
		UserID:         credential.UserID,
		CredentialJson: credential.CredentialJSON,
		CreatedAt:      toMillis(credential.CreatedAt),
		UpdatedAt:      toMillis(credential.UpdatedAt),
		LastUsedAt:     lastUsed,
	})
}

// GetPasskeyCredential fetches a stored WebAuthn credential.
func (s *Store) GetPasskeyCredential(ctx context.Context, credentialID string) (storage.PasskeyCredential, error) {
	if err := ctx.Err(); err != nil {
		return storage.PasskeyCredential{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.PasskeyCredential{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(credentialID) == "" {
		return storage.PasskeyCredential{}, fmt.Errorf("credential id is required")
	}

	row, err := s.q.GetPasskey(ctx, credentialID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.PasskeyCredential{}, storage.ErrNotFound
		}
		return storage.PasskeyCredential{}, fmt.Errorf("get passkey: %w", err)
	}

	return dbPasskeyToDomain(row), nil
}

// ListPasskeyCredentials returns passkeys for a user.
func (s *Store) ListPasskeyCredentials(ctx context.Context, userID string) ([]storage.PasskeyCredential, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("user id is required")
	}

	rows, err := s.q.ListPasskeysByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}

	credentials := make([]storage.PasskeyCredential, 0, len(rows))
	for _, row := range rows {
		credentials = append(credentials, dbPasskeyToDomain(row))
	}
	return credentials, nil
}

// DeletePasskeyCredential removes a passkey credential.
func (s *Store) DeletePasskeyCredential(ctx context.Context, credentialID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(credentialID) == "" {
		return fmt.Errorf("credential id is required")
	}
	return s.q.DeletePasskey(ctx, credentialID)
}

// PutPasskeySession stores a WebAuthn session.
func (s *Store) PutPasskeySession(ctx context.Context, session storage.PasskeySession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(session.ID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(session.Kind) == "" {
		return fmt.Errorf("session kind is required")
	}
	if strings.TrimSpace(session.SessionJSON) == "" {
		return fmt.Errorf("session json is required")
	}

	userID := sql.NullString{}
	if strings.TrimSpace(session.UserID) != "" {
		userID = sql.NullString{String: session.UserID, Valid: true}
	}

	return s.q.PutPasskeySession(ctx, db.PutPasskeySessionParams{
		ID:          session.ID,
		Kind:        session.Kind,
		UserID:      userID,
		SessionJson: session.SessionJSON,
		ExpiresAt:   toMillis(session.ExpiresAt),
	})
}

// GetPasskeySession fetches a stored WebAuthn session.
func (s *Store) GetPasskeySession(ctx context.Context, id string) (storage.PasskeySession, error) {
	if err := ctx.Err(); err != nil {
		return storage.PasskeySession{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.PasskeySession{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.PasskeySession{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetPasskeySession(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.PasskeySession{}, storage.ErrNotFound
		}
		return storage.PasskeySession{}, fmt.Errorf("get passkey session: %w", err)
	}

	return dbPasskeySessionToDomain(row), nil
}

// DeletePasskeySession removes a WebAuthn session.
func (s *Store) DeletePasskeySession(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("session id is required")
	}
	return s.q.DeletePasskeySession(ctx, id)
}

// DeleteExpiredPasskeySessions removes expired WebAuthn sessions.
func (s *Store) DeleteExpiredPasskeySessions(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	return s.q.DeleteExpiredPasskeySessions(ctx, toMillis(now))
}

func dbPasskeyToDomain(row db.Passkey) storage.PasskeyCredential {
	var lastUsed *time.Time
	if row.LastUsedAt.Valid {
		value := fromMillis(row.LastUsedAt.Int64)
		lastUsed = &value
	}
	return storage.PasskeyCredential{
		CredentialID:   row.CredentialID,
		UserID:         row.UserID,
		CredentialJSON: row.CredentialJson,
		CreatedAt:      fromMillis(row.CreatedAt),
		UpdatedAt:      fromMillis(row.UpdatedAt),
		LastUsedAt:     lastUsed,
	}
}

func dbPasskeySessionToDomain(row db.PasskeySession) storage.PasskeySession {
	userID := ""
	if row.UserID.Valid {
		userID = row.UserID.String
	}
	return storage.PasskeySession{
		ID:          row.ID,
		Kind:        row.Kind,
		UserID:      userID,
		SessionJSON: row.SessionJson,
		ExpiresAt:   fromMillis(row.ExpiresAt),
	}
}
