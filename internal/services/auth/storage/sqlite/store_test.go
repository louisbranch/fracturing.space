package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if got.ID != input.ID || got.DisplayName != input.DisplayName {
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
