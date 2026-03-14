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
		return fmt.Errorf("Storage is not configured.")
	}
	params, err := normalizePasskeyPutParams(credential)
	if err != nil {
		return err
	}
	return s.q.PutPasskey(ctx, params)
}

// GetPasskeyCredential fetches a stored WebAuthn credential.
func (s *Store) GetPasskeyCredential(ctx context.Context, credentialID string) (storage.PasskeyCredential, error) {
	if err := ctx.Err(); err != nil {
		return storage.PasskeyCredential{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.PasskeyCredential{}, fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(credentialID) == "" {
		return storage.PasskeyCredential{}, fmt.Errorf("Credential ID is required.")
	}

	row, err := s.q.GetPasskey(ctx, credentialID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.PasskeyCredential{}, storage.ErrNotFound
		}
		return storage.PasskeyCredential{}, fmt.Errorf("Get passkey: %w", err)
	}

	return dbPasskeyToDomain(row), nil
}

func normalizePasskeyPutParams(credential storage.PasskeyCredential) (db.PutPasskeyParams, error) {
	if strings.TrimSpace(credential.CredentialID) == "" {
		return db.PutPasskeyParams{}, fmt.Errorf("Credential ID is required.")
	}
	if strings.TrimSpace(credential.UserID) == "" {
		return db.PutPasskeyParams{}, fmt.Errorf("User ID is required.")
	}
	if strings.TrimSpace(credential.CredentialJSON) == "" {
		return db.PutPasskeyParams{}, fmt.Errorf("Credential JSON is required.")
	}
	if credential.CreatedAt.IsZero() {
		return db.PutPasskeyParams{}, fmt.Errorf("Created at is required.")
	}
	if credential.UpdatedAt.IsZero() {
		credential.UpdatedAt = credential.CreatedAt
	}

	lastUsed := sql.NullInt64{}
	if credential.LastUsedAt != nil {
		lastUsed = sql.NullInt64{Int64: toMillis(*credential.LastUsedAt), Valid: true}
	}

	return db.PutPasskeyParams{
		CredentialID:   credential.CredentialID,
		UserID:         credential.UserID,
		CredentialJson: credential.CredentialJSON,
		CreatedAt:      toMillis(credential.CreatedAt),
		UpdatedAt:      toMillis(credential.UpdatedAt),
		LastUsedAt:     lastUsed,
	}, nil
}

// ListPasskeyCredentials returns passkeys for a user.
func (s *Store) ListPasskeyCredentials(ctx context.Context, userID string) ([]storage.PasskeyCredential, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("User ID is required.")
	}

	rows, err := s.q.ListPasskeysByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("List passkeys: %w", err)
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
		return fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(credentialID) == "" {
		return fmt.Errorf("Credential ID is required.")
	}
	return s.q.DeletePasskey(ctx, credentialID)
}

// DeletePasskeyCredentialsByUser removes all passkey credentials for a user.
func (s *Store) DeletePasskeyCredentialsByUser(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("User ID is required.")
	}
	return s.q.DeletePasskeysByUser(ctx, userID)
}

// DeletePasskeyCredentialsByUserExcept removes all passkey credentials for a user except one.
func (s *Store) DeletePasskeyCredentialsByUserExcept(ctx context.Context, userID string, credentialID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(credentialID) == "" {
		return fmt.Errorf("User ID and credential ID are required.")
	}
	return s.q.DeletePasskeysByUserExcept(ctx, db.DeletePasskeysByUserExceptParams{
		UserID:       userID,
		CredentialID: credentialID,
	})
}

// PutPasskeySession stores a WebAuthn session.
func (s *Store) PutPasskeySession(ctx context.Context, session storage.PasskeySession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(session.ID) == "" {
		return fmt.Errorf("Session ID is required.")
	}
	if strings.TrimSpace(session.Kind) == "" {
		return fmt.Errorf("Session kind is required.")
	}
	if strings.TrimSpace(session.SessionJSON) == "" {
		return fmt.Errorf("Session JSON is required.")
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
		return storage.PasskeySession{}, fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(id) == "" {
		return storage.PasskeySession{}, fmt.Errorf("Session ID is required.")
	}

	row, err := s.q.GetPasskeySession(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.PasskeySession{}, storage.ErrNotFound
		}
		return storage.PasskeySession{}, fmt.Errorf("Get passkey session: %w", err)
	}

	return dbPasskeySessionToDomain(row), nil
}

// DeletePasskeySession removes a WebAuthn session.
func (s *Store) DeletePasskeySession(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("Session ID is required.")
	}
	return s.q.DeletePasskeySession(ctx, id)
}

// DeleteExpiredPasskeySessions removes expired WebAuthn sessions.
func (s *Store) DeleteExpiredPasskeySessions(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	return s.q.DeleteExpiredPasskeySessions(ctx, toMillis(now))
}

// PutRegistrationSession stores pending username signup state.
func (s *Store) PutRegistrationSession(ctx context.Context, session storage.RegistrationSession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(session.ID) == "" || strings.TrimSpace(session.UserID) == "" || strings.TrimSpace(session.Username) == "" {
		return fmt.Errorf("Registration session ID, user ID, and username are required.")
	}
	return s.q.PutRegistrationSession(ctx, db.PutRegistrationSessionParams{
		ID:               session.ID,
		UserID:           session.UserID,
		Username:         session.Username,
		Locale:           session.Locale,
		RecoveryCodeHash: session.RecoveryCodeHash,
		ExpiresAt:        toMillis(session.ExpiresAt),
		CreatedAt:        toMillis(session.CreatedAt),
		UpdatedAt:        toMillis(session.UpdatedAt),
	})
}

// GetRegistrationSession fetches pending username signup state.
func (s *Store) GetRegistrationSession(ctx context.Context, id string) (storage.RegistrationSession, error) {
	if err := ctx.Err(); err != nil {
		return storage.RegistrationSession{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.RegistrationSession{}, fmt.Errorf("Storage is not configured.")
	}
	row, err := s.q.GetRegistrationSession(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.RegistrationSession{}, storage.ErrNotFound
		}
		return storage.RegistrationSession{}, fmt.Errorf("Get registration session: %w", err)
	}
	return storage.RegistrationSession{
		ID:               row.ID,
		UserID:           row.UserID,
		Username:         row.Username,
		Locale:           row.Locale,
		RecoveryCodeHash: row.RecoveryCodeHash,
		ExpiresAt:        fromMillis(row.ExpiresAt),
		CreatedAt:        fromMillis(row.CreatedAt),
		UpdatedAt:        fromMillis(row.UpdatedAt),
	}, nil
}

// DeleteRegistrationSession removes pending username signup state.
func (s *Store) DeleteRegistrationSession(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	return s.q.DeleteRegistrationSession(ctx, id)
}

// DeleteExpiredRegistrationSessions removes expired signup state.
func (s *Store) DeleteExpiredRegistrationSessions(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	return s.q.DeleteExpiredRegistrationSessions(ctx, toMillis(now))
}

// PutRecoverySession stores narrow recovery-session state.
func (s *Store) PutRecoverySession(ctx context.Context, session storage.RecoverySession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	if strings.TrimSpace(session.ID) == "" || strings.TrimSpace(session.UserID) == "" {
		return fmt.Errorf("Recovery session ID and user ID are required.")
	}
	return s.q.PutRecoverySession(ctx, db.PutRecoverySessionParams{
		ID:        session.ID,
		UserID:    session.UserID,
		ExpiresAt: toMillis(session.ExpiresAt),
		CreatedAt: toMillis(session.CreatedAt),
	})
}

// GetRecoverySession fetches narrow recovery-session state.
func (s *Store) GetRecoverySession(ctx context.Context, id string) (storage.RecoverySession, error) {
	if err := ctx.Err(); err != nil {
		return storage.RecoverySession{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.RecoverySession{}, fmt.Errorf("Storage is not configured.")
	}
	row, err := s.q.GetRecoverySession(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.RecoverySession{}, storage.ErrNotFound
		}
		return storage.RecoverySession{}, fmt.Errorf("Get recovery session: %w", err)
	}
	return storage.RecoverySession{
		ID:        row.ID,
		UserID:    row.UserID,
		ExpiresAt: fromMillis(row.ExpiresAt),
		CreatedAt: fromMillis(row.CreatedAt),
	}, nil
}

// DeleteRecoverySession removes narrow recovery-session state.
func (s *Store) DeleteRecoverySession(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	return s.q.DeleteRecoverySession(ctx, id)
}

// DeleteExpiredRecoverySessions removes expired recovery-session state.
func (s *Store) DeleteExpiredRecoverySessions(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("Storage is not configured.")
	}
	return s.q.DeleteExpiredRecoverySessions(ctx, toMillis(now))
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
