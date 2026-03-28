package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

func TestOpenRequiresPath(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestStoreDBNilSafe(t *testing.T) {
	var store *Store
	if store.DB() != nil {
		t.Fatal("expected nil DB for nil store")
	}
}

func TestStoreCloseNilSafe(t *testing.T) {
	var store *Store
	if err := store.Close(); err != nil {
		t.Fatalf("nil store close: %v", err)
	}
	if err := (&Store{}).Close(); err != nil {
		t.Fatalf("zero store close: %v", err)
	}
}

func TestOpenRemovesLegacySocialTables(t *testing.T) {
	store := openTempStore(t)

	for _, table := range []string{"account_profiles", "user_contacts"} {
		var found string
		err := store.DB().QueryRow(
			`SELECT name
			 FROM sqlite_master
			 WHERE type = 'table' AND name = ?`,
			table,
		).Scan(&found)
		if err == nil {
			t.Fatalf("legacy table %q should not exist", table)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("query legacy table %q: %v", table, err)
		}
	}
}

func TestPutGetUserRoundTrip(t *testing.T) {
	store := openTempStore(t)

	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	reservedUntil := updated.Add(time.Hour)
	input := user.User{
		ID:                        "user-1",
		Username:                  "testuser",
		Locale:                    commonv1.Locale_LOCALE_PT_BR,
		RecoveryCodeHash:          "hash-1",
		RecoveryReservedSessionID: "recovery-1",
		RecoveryReservedUntil:     &reservedUntil,
		RecoveryCodeUpdatedAt:     updated,
		CreatedAt:                 created,
		UpdatedAt:                 updated,
	}

	if err := store.PutUser(context.Background(), input); err != nil {
		t.Fatalf("put user: %v", err)
	}

	got, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.ID != input.ID || got.Username != input.Username {
		t.Fatalf("unexpected user: %+v", got)
	}
	if got.Locale != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("locale = %v, want %v", got.Locale, commonv1.Locale_LOCALE_PT_BR)
	}
	if got.RecoveryCodeHash != input.RecoveryCodeHash {
		t.Fatalf("recovery code hash = %q, want %q", got.RecoveryCodeHash, input.RecoveryCodeHash)
	}

	byUsername, err := store.GetUserByUsername(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("get user by username: %v", err)
	}
	if byUsername.ID != input.ID {
		t.Fatalf("user by username = %+v, want id %q", byUsername, input.ID)
	}
}

func TestPutUserDefaultsLocaleWhenUnset(t *testing.T) {
	store := openTempStore(t)

	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	input := user.User{
		ID:                    "user-1",
		Username:              "testuser",
		Locale:                commonv1.Locale_LOCALE_UNSPECIFIED,
		RecoveryCodeUpdatedAt: now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := store.PutUser(context.Background(), input); err != nil {
		t.Fatalf("put user: %v", err)
	}

	got, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.Locale != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("locale = %v, want %v", got.Locale, commonv1.Locale_LOCALE_EN_US)
	}
}

func TestPutUserValidationAndUniqueness(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)

	if err := store.PutUser(context.Background(), user.User{ID: "  "}); err == nil {
		t.Fatal("expected error for empty user id")
	}

	err := store.PutUser(context.Background(), user.User{
		ID:                    "user-1",
		Username:              " ",
		RecoveryCodeUpdatedAt: now,
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err == nil {
		t.Fatal("expected error for empty username")
	}

	if err := store.PutUser(context.Background(), user.User{
		ID:                    "user-1",
		Username:              "shared-user",
		RecoveryCodeUpdatedAt: now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	err = store.PutUser(context.Background(), user.User{
		ID:                    "user-2",
		Username:              "shared-user",
		RecoveryCodeUpdatedAt: now,
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err == nil {
		t.Fatal("expected duplicate username error")
	}
	if _, err := store.GetUser(context.Background(), "user-2"); err != storage.ErrNotFound {
		t.Fatalf("expected user-2 not found, got %v", err)
	}
}

func TestPutUserCanonicalizesUsername(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)

	if err := store.PutUser(context.Background(), user.User{
		ID:                    "user-1",
		Username:              "  Test.User  ",
		RecoveryCodeUpdatedAt: now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	got, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.Username != "test.user" {
		t.Fatalf("username = %q, want %q", got.Username, "test.user")
	}
}

func TestGetUserByUsernameValidationAndNotFound(t *testing.T) {
	store := openTempStore(t)

	if _, err := store.GetUserByUsername(context.Background(), "   "); err == nil || err.Error() != "Username is required." {
		t.Fatalf("expected username required error, got %v", err)
	}
	if _, err := store.GetUserByUsername(context.Background(), "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestListUsersPagination(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)

	for i, id := range []string{"user-1", "user-2", "user-3"} {
		if err := store.PutUser(context.Background(), user.User{
			ID:                    id,
			Username:              fmt.Sprintf("user%d", i+1),
			RecoveryCodeUpdatedAt: now,
			CreatedAt:             now,
			UpdatedAt:             now,
		}); err != nil {
			t.Fatalf("put user: %v", err)
		}
	}

	page, err := store.ListUsers(context.Background(), 2, "")
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(page.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(page.Users))
	}
	if page.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	second, err := store.ListUsers(context.Background(), 2, page.NextPageToken)
	if err != nil {
		t.Fatalf("list users page 2: %v", err)
	}
	if len(second.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(second.Users))
	}
	if second.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %s", second.NextPageToken)
	}
}

func TestGetAuthStatistics(t *testing.T) {
	store := openTempStore(t)

	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	if err := store.PutUser(context.Background(), user.User{
		ID:                    "user-1",
		Username:              "testuser",
		RecoveryCodeUpdatedAt: created,
		CreatedAt:             created,
		UpdatedAt:             created,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	since := created.Add(-time.Hour)
	stats, err := store.GetAuthStatistics(context.Background(), &since)
	if err != nil {
		t.Fatalf("get auth statistics: %v", err)
	}
	if stats.UserCount != 1 {
		t.Fatalf("expected 1 user, got %d", stats.UserCount)
	}

	stats, err = store.GetAuthStatistics(context.Background(), nil)
	if err != nil {
		t.Fatalf("get auth statistics all time: %v", err)
	}
	if stats.UserCount != 1 {
		t.Fatalf("expected 1 user, got %d", stats.UserCount)
	}
}

func TestPutUserWithIntegrationOutboxEventCanonicalizesUsername(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)

	event := storage.IntegrationOutboxEvent{
		ID:            "event-1",
		EventType:     "auth.user.created",
		PayloadJSON:   `{"user_id":"user-1"}`,
		DedupeKey:     "user-1",
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.PutUserWithIntegrationOutboxEvent(context.Background(), user.User{
		ID:                    "user-1",
		Username:              "  Mixed-Case  ",
		RecoveryCodeUpdatedAt: now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, event); err != nil {
		t.Fatalf("put user with outbox event: %v", err)
	}

	got, err := store.GetUserByUsername(context.Background(), "mixed-case")
	if err != nil {
		t.Fatalf("get user by canonical username: %v", err)
	}
	if got.ID != "user-1" {
		t.Fatalf("user id = %q, want %q", got.ID, "user-1")
	}

	outboxEvent, err := store.GetIntegrationOutboxEvent(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("get integration outbox event: %v", err)
	}
	if outboxEvent.EventType != event.EventType {
		t.Fatalf("event type = %q, want %q", outboxEvent.EventType, event.EventType)
	}
}

func TestPutUserPasskeyWithIntegrationOutboxEventPersistsSignupAtomically(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	credential := storage.PasskeyCredential{
		CredentialID:   "cred-1",
		UserID:         "user-1",
		CredentialJSON: "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	session := storage.WebSession{
		ID:        "ws-1",
		UserID:    "user-1",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	event := storage.IntegrationOutboxEvent{
		ID:            "event-1",
		EventType:     "auth.signup_completed",
		PayloadJSON:   `{"user_id":"user-1","username":"alpha"}`,
		DedupeKey:     "signup_completed:user:user-1:v1",
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := store.PutUserPasskeyWithIntegrationOutboxEvent(context.Background(), user.User{
		ID:                    "user-1",
		Username:              "Alpha",
		RecoveryCodeUpdatedAt: now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, credential, session, event); err != nil {
		t.Fatalf("put signup payload: %v", err)
	}

	storedUser, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if storedUser.Username != "alpha" {
		t.Fatalf("username = %q, want %q", storedUser.Username, "alpha")
	}

	storedCredential, err := store.GetPasskeyCredential(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("get passkey: %v", err)
	}
	if storedCredential.UserID != "user-1" {
		t.Fatalf("credential user_id = %q, want %q", storedCredential.UserID, "user-1")
	}

	outboxEvent, err := store.GetIntegrationOutboxEvent(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("get integration outbox event: %v", err)
	}
	if outboxEvent.EventType != "auth.signup_completed" {
		t.Fatalf("event type = %q, want %q", outboxEvent.EventType, "auth.signup_completed")
	}

	storedSession, err := store.GetWebSession(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("get web session: %v", err)
	}
	if storedSession.UserID != "user-1" {
		t.Fatalf("session user_id = %q, want %q", storedSession.UserID, "user-1")
	}
}

func TestPutUserPasskeyWithIntegrationOutboxEventRejectsInvalidWebSessionAtomically(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	err := store.PutUserPasskeyWithIntegrationOutboxEvent(context.Background(), user.User{
		ID:                    "user-1",
		Username:              "Alpha",
		RecoveryCodeUpdatedAt: now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, storage.PasskeyCredential{
		CredentialID:   "cred-1",
		UserID:         "user-1",
		CredentialJSON: "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
	}, storage.WebSession{
		UserID:    "user-1",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}, storage.IntegrationOutboxEvent{
		ID:            "event-1",
		EventType:     "auth.signup_completed",
		PayloadJSON:   `{"user_id":"user-1","username":"alpha"}`,
		DedupeKey:     "signup_completed:user:user-1:v1",
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err == nil {
		t.Fatal("expected invalid web session error")
	}

	if _, getErr := store.GetUser(context.Background(), "user-1"); !errors.Is(getErr, storage.ErrNotFound) {
		t.Fatalf("expected user write rollback, got %v", getErr)
	}
	if _, getErr := store.GetPasskeyCredential(context.Background(), "cred-1"); !errors.Is(getErr, storage.ErrNotFound) {
		t.Fatalf("expected passkey write rollback, got %v", getErr)
	}
	if _, getErr := store.GetIntegrationOutboxEvent(context.Background(), "event-1"); !errors.Is(getErr, storage.ErrNotFound) {
		t.Fatalf("expected outbox write rollback, got %v", getErr)
	}
}

func TestPasskeyCredentialRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	putTestUser(t, store, "user-1", "testuser", now)

	lastUsed := now.Add(time.Minute)
	input := storage.PasskeyCredential{
		CredentialID:   "cred-1",
		UserID:         "user-1",
		CredentialJSON: "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
		LastUsedAt:     &lastUsed,
	}
	if err := store.PutPasskeyCredential(context.Background(), input); err != nil {
		t.Fatalf("put passkey: %v", err)
	}

	got, err := store.GetPasskeyCredential(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("get passkey: %v", err)
	}
	if got.CredentialID != input.CredentialID || got.UserID != input.UserID {
		t.Fatalf("unexpected credential: %+v", got)
	}
	if got.LastUsedAt == nil {
		t.Fatalf("expected last used at")
	}

	second := input
	second.CredentialID = "cred-2"
	if err := store.PutPasskeyCredential(context.Background(), second); err != nil {
		t.Fatalf("put second passkey: %v", err)
	}
	if err := store.DeletePasskeyCredentialsByUserExcept(context.Background(), "user-1", "cred-2"); err != nil {
		t.Fatalf("delete passkeys except: %v", err)
	}
	if _, err := store.GetPasskeyCredential(context.Background(), "cred-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected cred-1 deleted, got %v", err)
	}
	if _, err := store.GetPasskeyCredential(context.Background(), "cred-2"); err != nil {
		t.Fatalf("expected cred-2 retained: %v", err)
	}

	if err := store.DeletePasskeyCredentialsByUser(context.Background(), "user-1"); err != nil {
		t.Fatalf("delete passkeys by user: %v", err)
	}
	if _, err := store.GetPasskeyCredential(context.Background(), "cred-2"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected cred-2 deleted, got %v", err)
	}
}

func TestListPasskeyCredentialsAndDeleteSingle(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	putTestUser(t, store, "user-1", "alpha", now)
	putTestUser(t, store, "user-2", "beta", now)

	if err := store.PutPasskeyCredential(context.Background(), storage.PasskeyCredential{
		CredentialID:   "cred-1",
		UserID:         "user-1",
		CredentialJSON: "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("put passkey 1: %v", err)
	}
	if err := store.PutPasskeyCredential(context.Background(), storage.PasskeyCredential{
		CredentialID:   "cred-2",
		UserID:         "user-2",
		CredentialJSON: "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("put passkey 2: %v", err)
	}

	rows, err := store.ListPasskeyCredentials(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list passkeys: %v", err)
	}
	if len(rows) != 1 || rows[0].CredentialID != "cred-1" {
		t.Fatalf("rows = %#v", rows)
	}

	if err := store.DeletePasskeyCredential(context.Background(), "cred-1"); err != nil {
		t.Fatalf("delete passkey: %v", err)
	}
	if _, err := store.GetPasskeyCredential(context.Background(), "cred-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected deleted credential, got %v", err)
	}
	if _, err := store.GetPasskeyCredential(context.Background(), "cred-2"); err != nil {
		t.Fatalf("expected unrelated credential retained, got %v", err)
	}
}

func TestPasskeySessionRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	input := storage.PasskeySession{
		ID:          "session-1",
		Kind:        "login",
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}
	if err := store.PutPasskeySession(context.Background(), input); err != nil {
		t.Fatalf("put session: %v", err)
	}

	got, err := store.GetPasskeySession(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.ID != input.ID || got.Kind != input.Kind {
		t.Fatalf("unexpected session: %+v", got)
	}

	if err := store.DeletePasskeySession(context.Background(), "session-1"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, err := store.GetPasskeySession(context.Background(), "session-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestPasskeyDeleteValidation(t *testing.T) {
	store := openTempStore(t)

	if err := store.DeletePasskeyCredential(context.Background(), " "); err == nil || err.Error() != "Credential ID is required." {
		t.Fatalf("expected credential id required error, got %v", err)
	}
	if err := store.DeletePasskeySession(context.Background(), " "); err == nil || err.Error() != "Session ID is required." {
		t.Fatalf("expected session id required error, got %v", err)
	}
}

func TestDeleteExpiredPasskeySessions(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	if err := store.PutPasskeySession(context.Background(), storage.PasskeySession{
		ID:          "expired",
		Kind:        "login",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("put session: %v", err)
	}
	if err := store.PutPasskeySession(context.Background(), storage.PasskeySession{
		ID:          "active",
		Kind:        "login",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("put session: %v", err)
	}

	if err := store.DeleteExpiredPasskeySessions(context.Background(), now); err != nil {
		t.Fatalf("delete expired sessions: %v", err)
	}
	if _, err := store.GetPasskeySession(context.Background(), "expired"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected expired session deleted")
	}
	if _, err := store.GetPasskeySession(context.Background(), "active"); err != nil {
		t.Fatalf("expected active session retained: %v", err)
	}
}

func TestRegistrationSessionRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	input := storage.RegistrationSession{
		ID:               "reg-1",
		UserID:           "user-1",
		Username:         "testuser",
		Locale:           "en-US",
		RecoveryCodeHash: "hash",
		ExpiresAt:        now.Add(10 * time.Minute),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.PutRegistrationSession(context.Background(), input); err != nil {
		t.Fatalf("put registration session: %v", err)
	}

	got, err := store.GetRegistrationSession(context.Background(), "reg-1")
	if err != nil {
		t.Fatalf("get registration session: %v", err)
	}
	if got.Username != input.Username || got.UserID != input.UserID {
		t.Fatalf("unexpected registration session: %+v", got)
	}

	if err := store.DeleteRegistrationSession(context.Background(), "reg-1"); err != nil {
		t.Fatalf("delete registration session: %v", err)
	}
	if _, err := store.GetRegistrationSession(context.Background(), "reg-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGetRegistrationSessionByUsername(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	input := storage.RegistrationSession{
		ID:        "reg-1",
		UserID:    "user-1",
		Username:  "testuser",
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.PutRegistrationSession(context.Background(), input); err != nil {
		t.Fatalf("put registration session: %v", err)
	}

	got, err := store.GetRegistrationSessionByUsername(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("get registration session by username: %v", err)
	}
	if got.ID != input.ID || got.UserID != input.UserID {
		t.Fatalf("unexpected registration session: %+v", got)
	}
	if _, err := store.GetRegistrationSessionByUsername(context.Background(), "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found for missing username, got %v", err)
	}
}

func TestRegistrationAndRecoverySessionValidation(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	if err := store.PutRegistrationSession(context.Background(), storage.RegistrationSession{
		ID:        "reg-1",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}); err == nil || err.Error() != "Registration session ID, user ID, and username are required." {
		t.Fatalf("expected registration validation error, got %v", err)
	}

	if err := store.PutRecoverySession(context.Background(), storage.RecoverySession{
		ID:        "recovery-1",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}); err == nil || err.Error() != "Recovery session ID and user ID are required." {
		t.Fatalf("expected recovery validation error, got %v", err)
	}
}

func TestDeleteExpiredRegistrationSessions(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	if err := store.PutRegistrationSession(context.Background(), storage.RegistrationSession{
		ID:        "expired",
		UserID:    "user-1",
		Username:  "alpha",
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put registration session: %v", err)
	}
	if err := store.PutRegistrationSession(context.Background(), storage.RegistrationSession{
		ID:        "active",
		UserID:    "user-2",
		Username:  "beta",
		ExpiresAt: now.Add(time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put registration session: %v", err)
	}

	if err := store.DeleteExpiredRegistrationSessions(context.Background(), now); err != nil {
		t.Fatalf("delete expired registration sessions: %v", err)
	}
	if _, err := store.GetRegistrationSession(context.Background(), "expired"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected expired registration session deleted")
	}
	if _, err := store.GetRegistrationSession(context.Background(), "active"); err != nil {
		t.Fatalf("expected active registration session retained: %v", err)
	}
}

func TestRecoverySessionRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	putTestUser(t, store, "user-1", "alpha", now)

	input := storage.RecoverySession{
		ID:        "recovery-1",
		UserID:    "user-1",
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
	}
	if err := store.PutRecoverySession(context.Background(), input); err != nil {
		t.Fatalf("put recovery session: %v", err)
	}

	got, err := store.GetRecoverySession(context.Background(), "recovery-1")
	if err != nil {
		t.Fatalf("get recovery session: %v", err)
	}
	if got.ID != input.ID || got.UserID != input.UserID {
		t.Fatalf("unexpected recovery session: %+v", got)
	}

	if err := store.DeleteRecoverySession(context.Background(), "recovery-1"); err != nil {
		t.Fatalf("delete recovery session: %v", err)
	}
	if _, err := store.GetRecoverySession(context.Background(), "recovery-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDeleteExpiredRecoverySessions(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	putTestUser(t, store, "user-1", "alpha", now)
	putTestUser(t, store, "user-2", "beta", now)

	if err := store.PutRecoverySession(context.Background(), storage.RecoverySession{
		ID:        "expired",
		UserID:    "user-1",
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("put recovery session: %v", err)
	}
	if err := store.PutRecoverySession(context.Background(), storage.RecoverySession{
		ID:        "active",
		UserID:    "user-2",
		ExpiresAt: now.Add(time.Minute),
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("put recovery session: %v", err)
	}

	if err := store.DeleteExpiredRecoverySessions(context.Background(), now); err != nil {
		t.Fatalf("delete expired recovery sessions: %v", err)
	}
	if _, err := store.GetRecoverySession(context.Background(), "expired"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected expired recovery session deleted")
	}
	if _, err := store.GetRecoverySession(context.Background(), "active"); err != nil {
		t.Fatalf("expected active recovery session retained: %v", err)
	}
}

func TestWebSessionRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 23, 15, 0, 0, 0, time.UTC)

	putTestUser(t, store, "user-1", "primary", now)

	if err := store.PutWebSession(context.Background(), storage.WebSession{ID: "ws-1", UserID: "user-1", CreatedAt: now, ExpiresAt: now.Add(30 * time.Minute)}); err != nil {
		t.Fatalf("put web session: %v", err)
	}
	if err := store.PutWebSession(context.Background(), storage.WebSession{ID: "ws-2", UserID: "user-1", CreatedAt: now, ExpiresAt: now.Add(30 * time.Minute)}); err != nil {
		t.Fatalf("put second web session: %v", err)
	}

	got, err := store.GetWebSession(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("get web session: %v", err)
	}
	if got.ID != "ws-1" || got.UserID != "user-1" {
		t.Fatalf("unexpected web session: %+v", got)
	}

	if err := store.RevokeWebSession(context.Background(), "ws-1", now.Add(time.Minute)); err != nil {
		t.Fatalf("revoke web session: %v", err)
	}
	revoked, err := store.GetWebSession(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("get revoked web session: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatalf("expected revoked_at")
	}

	if err := store.RevokeWebSessionsByUser(context.Background(), "user-1", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("revoke web sessions by user: %v", err)
	}
	second, err := store.GetWebSession(context.Background(), "ws-2")
	if err != nil {
		t.Fatalf("get second web session: %v", err)
	}
	if second.RevokedAt == nil {
		t.Fatalf("expected second session revoked")
	}

	if err := store.DeleteExpiredWebSessions(context.Background(), now.Add(time.Hour)); err != nil {
		t.Fatalf("delete expired web sessions: %v", err)
	}
	if _, err := store.GetWebSession(context.Background(), "ws-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after expiry deletion, got %v", err)
	}
}

func TestPutWebSessionDefaultsCreatedAtAndUpsertsRevocation(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 23, 15, 0, 0, 0, time.UTC)
	putTestUser(t, store, "user-1", "primary", now)
	putTestUser(t, store, "user-2", "secondary", now)

	if err := store.PutWebSession(context.Background(), storage.WebSession{
		ID:        "ws-1",
		UserID:    "user-1",
		ExpiresAt: now.Add(30 * time.Minute),
	}); err != nil {
		t.Fatalf("put web session: %v", err)
	}

	first, err := store.GetWebSession(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("get initial web session: %v", err)
	}
	if first.CreatedAt.IsZero() {
		t.Fatal("created_at = zero, want auto-filled timestamp")
	}

	revokedAt := now.Add(5 * time.Minute)
	if err := store.PutWebSession(context.Background(), storage.WebSession{
		ID:        "ws-1",
		UserID:    "user-2",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
		RevokedAt: &revokedAt,
	}); err != nil {
		t.Fatalf("put updated web session: %v", err)
	}

	updated, err := store.GetWebSession(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("get updated web session: %v", err)
	}
	if updated.UserID != "user-2" || !updated.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("updated web session = %+v", updated)
	}
	if updated.RevokedAt == nil || !updated.RevokedAt.Equal(revokedAt) {
		t.Fatalf("revoked_at = %v, want %v", updated.RevokedAt, revokedAt)
	}
}

func TestWebSessionValidationAndRevocationEdges(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 23, 16, 0, 0, 0, time.UTC)
	putTestUser(t, store, "user-1", "primary", now)

	if err := store.PutWebSession(context.Background(), storage.WebSession{ID: "ws-1", UserID: "user-1"}); err == nil || err.Error() != "Expires at is required." {
		t.Fatalf("expected expires-at validation error, got %v", err)
	}
	if _, err := store.GetWebSession(context.Background(), " "); err == nil || err.Error() != "Session ID is required." {
		t.Fatalf("expected session id validation error, got %v", err)
	}
	if err := store.RevokeWebSession(context.Background(), "missing", now); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found revoke error, got %v", err)
	}
	if err := store.RevokeWebSessionsByUser(context.Background(), " ", now); err == nil || err.Error() != "User ID is required." {
		t.Fatalf("expected user id validation error, got %v", err)
	}

	expired := now.Add(-time.Minute)
	if err := store.PutWebSession(context.Background(), storage.WebSession{
		ID:        "ws-expired",
		UserID:    "user-1",
		CreatedAt: now.Add(-time.Hour),
		ExpiresAt: expired,
	}); err != nil {
		t.Fatalf("put expired web session: %v", err)
	}
	if err := store.DeleteExpiredWebSessions(context.Background(), time.Time{}); err != nil {
		t.Fatalf("delete expired web sessions with zero time: %v", err)
	}
	if _, err := store.GetWebSession(context.Background(), "ws-expired"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected expired session deleted, got %v", err)
	}
}

func TestExtractUpMigration(t *testing.T) {
	content := strings.Join([]string{
		"-- +migrate Up",
		"CREATE TABLE users (id TEXT);",
		"-- +migrate Down",
		"DROP TABLE users;",
	}, "\n")

	up := extractUpMigration(content)
	if !strings.Contains(up, "CREATE TABLE users") {
		t.Fatalf("unexpected up migration: %q", up)
	}
}

func TestIsAlreadyExistsError(t *testing.T) {
	if isAlreadyExistsError(errors.New("record already exists")) != true {
		t.Error("expected true for 'already exists' error")
	}
	if isAlreadyExistsError(errors.New("not found")) != false {
		t.Error("expected false for unrelated error")
	}
}

func openTempStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "auth.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}

func putTestUser(t *testing.T, store *Store, userID string, username string, now time.Time) {
	t.Helper()
	if err := store.PutUser(context.Background(), user.User{
		ID:                    userID,
		Username:              username,
		RecoveryCodeUpdatedAt: now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}
}
