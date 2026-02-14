package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
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

func TestPutGetUserRoundTrip(t *testing.T) {
	store := openTempStore(t)

	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	input := user.User{
		ID:          "user-1",
		DisplayName: "User",
		Locale:      commonv1.Locale_LOCALE_PT_BR,
		CreatedAt:   created,
		UpdatedAt:   updated,
	}

	if err := store.PutUser(context.Background(), input); err != nil {
		t.Fatalf("put user: %v", err)
	}

	got, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.ID != input.ID || got.DisplayName != input.DisplayName || got.Locale != input.Locale {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestPutUserRequiresID(t *testing.T) {
	store := openTempStore(t)

	err := store.PutUser(context.Background(), user.User{ID: "  "})
	if err == nil {
		t.Fatal("expected error for empty user id")
	}
}

func TestGetUserNotFound(t *testing.T) {
	store := openTempStore(t)

	_, err := store.GetUser(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if err != storage.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGetUserRequiresID(t *testing.T) {
	store := openTempStore(t)

	_, err := store.GetUser(context.Background(), " ")
	if err == nil {
		t.Fatal("expected error for empty user id")
	}
}

func TestListUsersPagination(t *testing.T) {
	store := openTempStore(t)

	for _, id := range []string{"user-1", "user-2", "user-3"} {
		if err := store.PutUser(context.Background(), user.User{
			ID:          id,
			DisplayName: "User",
			Locale:      platformi18n.DefaultLocale(),
			CreatedAt:   time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
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

func TestListUsersInvalidPageSize(t *testing.T) {
	store := openTempStore(t)

	if _, err := store.ListUsers(context.Background(), 0, ""); err == nil {
		t.Fatal("expected error for invalid page size")
	}
}

func TestListUsersContextError(t *testing.T) {
	store := openTempStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.ListUsers(ctx, 1, "")
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestGetAuthStatisticsSince(t *testing.T) {
	store := openTempStore(t)

	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	if err := store.PutUser(context.Background(), user.User{
		ID:          "user-1",
		DisplayName: "User",
		Locale:      platformi18n.DefaultLocale(),
		CreatedAt:   created,
		UpdatedAt:   created,
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
}

func TestGetAuthStatisticsAllTime(t *testing.T) {
	store := openTempStore(t)

	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	if err := store.PutUser(context.Background(), user.User{
		ID:          "user-1",
		DisplayName: "User",
		Locale:      platformi18n.DefaultLocale(),
		CreatedAt:   created,
		UpdatedAt:   created,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	stats, err := store.GetAuthStatistics(context.Background(), nil)
	if err != nil {
		t.Fatalf("get auth statistics: %v", err)
	}
	if stats.UserCount != 1 {
		t.Fatalf("expected 1 user, got %d", stats.UserCount)
	}
}

func TestPasskeyCredentialRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	if err := store.PutUser(context.Background(), user.User{
		ID:          "user-1",
		DisplayName: "User",
		Locale:      platformi18n.DefaultLocale(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

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

	list, err := store.ListPasskeyCredentials(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list passkeys: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(list))
	}

	if err := store.DeletePasskeyCredential(context.Background(), "cred-1"); err != nil {
		t.Fatalf("delete passkey: %v", err)
	}
	if _, err := store.GetPasskeyCredential(context.Background(), "cred-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestPasskeyCredentialRequiresFields(t *testing.T) {
	store := openTempStore(t)

	if err := store.PutPasskeyCredential(context.Background(), storage.PasskeyCredential{}); err == nil {
		t.Fatalf("expected error for empty credential")
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

func TestUserEmailRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	if err := store.PutUser(context.Background(), user.User{
		ID:          "user-1",
		DisplayName: "User",
		Locale:      platformi18n.DefaultLocale(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	input := storage.UserEmail{
		ID:        "email-1",
		UserID:    "user-1",
		Email:     "alpha@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.PutUserEmail(context.Background(), input); err != nil {
		t.Fatalf("put email: %v", err)
	}

	got, err := store.GetUserEmailByEmail(context.Background(), "alpha@example.com")
	if err != nil {
		t.Fatalf("get email: %v", err)
	}
	if got.Email != input.Email || got.UserID != input.UserID {
		t.Fatalf("unexpected email: %+v", got)
	}

	list, err := store.ListUserEmailsByUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list emails: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 email, got %d", len(list))
	}

	verifiedAt := now.Add(time.Minute)
	if err := store.VerifyUserEmail(context.Background(), "user-1", "alpha@example.com", verifiedAt); err != nil {
		t.Fatalf("verify email: %v", err)
	}
	verified, err := store.GetUserEmailByEmail(context.Background(), "alpha@example.com")
	if err != nil {
		t.Fatalf("get email: %v", err)
	}
	if verified.VerifiedAt == nil {
		t.Fatalf("expected verified_at")
	}
}

func TestMagicLinkRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	link := storage.MagicLink{
		Token:     "token-1",
		UserID:    "user-1",
		Email:     "alpha@example.com",
		PendingID: "pending-1",
		CreatedAt: now,
		ExpiresAt: now.Add(10 * time.Minute),
	}
	if err := store.PutMagicLink(context.Background(), link); err != nil {
		t.Fatalf("put magic link: %v", err)
	}

	got, err := store.GetMagicLink(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("get magic link: %v", err)
	}
	if got.Token != link.Token || got.PendingID != link.PendingID {
		t.Fatalf("unexpected magic link: %+v", got)
	}

	usedAt := now.Add(time.Minute)
	if err := store.MarkMagicLinkUsed(context.Background(), "token-1", usedAt); err != nil {
		t.Fatalf("mark used: %v", err)
	}
	used, err := store.GetMagicLink(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("get magic link: %v", err)
	}
	if used.UsedAt == nil {
		t.Fatalf("expected used_at")
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
