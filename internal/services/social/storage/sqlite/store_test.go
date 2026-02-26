package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/social/storage"
	msqlite "modernc.org/sqlite"
)

type opaqueWrapError struct {
	cause error
}

func (e opaqueWrapError) Error() string {
	return "wrapped database error"
}

func (e opaqueWrapError) Unwrap() error {
	return e.cause
}

type asSQLiteErrorWithUniqueMessage struct{}

func (e asSQLiteErrorWithUniqueMessage) Error() string {
	return "UNIQUE constraint failed: user_profiles.username"
}

func (e asSQLiteErrorWithUniqueMessage) As(target any) bool {
	sqliteErrPtr, ok := target.(**msqlite.Error)
	if !ok {
		return false
	}
	// Zero value mimics an unexpected/non-unique code while preserving typed matching.
	*sqliteErrPtr = &msqlite.Error{}
	return true
}

func TestContactRoundTripAndOwnerScoping(t *testing.T) {
	store, err := Open(t.TempDir() + "/social.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, time.February, 22, 12, 0, 0, 0, time.UTC)
	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-1",
		ContactUserID: "user-2",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put contact 1->2: %v", err)
	}
	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-1",
		ContactUserID: "user-3",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put contact 1->3: %v", err)
	}
	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-2",
		ContactUserID: "user-1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put contact 2->1: %v", err)
	}

	page, err := store.ListContacts(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(page.Contacts) != 2 {
		t.Fatalf("contacts len = %d, want 2", len(page.Contacts))
	}
	for _, contact := range page.Contacts {
		if contact.OwnerUserID != "user-1" {
			t.Fatalf("owner_user_id = %q, want user-1", contact.OwnerUserID)
		}
	}
}

func TestUserProfileRoundTripUpdateAndLookup(t *testing.T) {
	store, err := Open(t.TempDir() + "/social.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	createdAt := time.Date(2026, time.February, 22, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)
	if err := store.PutUserProfile(context.Background(), storage.UserProfile{
		UserID:        "user-1",
		Username:      "Alice_One",
		Name:          "Alice",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "001",
		Bio:           "Campaign manager",
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}); err != nil {
		t.Fatalf("put user profile: %v", err)
	}

	gotByUser, err := store.GetUserProfileByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user profile by user: %v", err)
	}
	if gotByUser.UserID != "user-1" {
		t.Fatalf("user_id = %q, want user-1", gotByUser.UserID)
	}
	if gotByUser.Username != "alice_one" {
		t.Fatalf("username = %q, want alice_one", gotByUser.Username)
	}
	if gotByUser.Name != "Alice" {
		t.Fatalf("name = %q, want Alice", gotByUser.Name)
	}
	if !gotByUser.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %v, want %v", gotByUser.CreatedAt, createdAt)
	}

	if err := store.PutUserProfile(context.Background(), storage.UserProfile{
		UserID:        "user-1",
		Username:      "Alice-Two",
		Name:          "Alice Two",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "002",
		Bio:           "Updated",
		CreatedAt:     updatedAt,
		UpdatedAt:     updatedAt,
	}); err != nil {
		t.Fatalf("update user profile: %v", err)
	}

	gotByLookup, err := store.GetUserProfileByUsername(context.Background(), "ALICE-two")
	if err != nil {
		t.Fatalf("lookup user profile: %v", err)
	}
	if gotByLookup.UserID != "user-1" {
		t.Fatalf("lookup user_id = %q, want user-1", gotByLookup.UserID)
	}
	if gotByLookup.Username != "alice-two" {
		t.Fatalf("lookup username = %q, want alice-two", gotByLookup.Username)
	}
	if gotByLookup.Name != "Alice Two" {
		t.Fatalf("lookup name = %q, want Alice Two", gotByLookup.Name)
	}
	if gotByLookup.Bio != "Updated" {
		t.Fatalf("lookup bio = %q, want Updated", gotByLookup.Bio)
	}

	if _, err := store.GetUserProfileByUsername(context.Background(), "alice_one"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("old username lookup err = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestUserProfileSameValueUpdateIsNoOp(t *testing.T) {
	store, err := Open(t.TempDir() + "/social.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	initial := time.Date(2026, time.February, 22, 17, 0, 0, 0, time.UTC)
	if err := store.PutUserProfile(context.Background(), storage.UserProfile{
		UserID:        "user-1",
		Username:      "Alice_One",
		Name:          "Alice",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "001",
		Bio:           "Campaign manager",
		CreatedAt:     initial,
		UpdatedAt:     initial,
	}); err != nil {
		t.Fatalf("put initial user profile: %v", err)
	}

	retryAt := initial.Add(2 * time.Hour)
	if err := store.PutUserProfile(context.Background(), storage.UserProfile{
		UserID:        "user-1",
		Username:      "ALICE_ONE",
		Name:          "Alice",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "001",
		Bio:           "Campaign manager",
		CreatedAt:     retryAt,
		UpdatedAt:     retryAt,
	}); err != nil {
		t.Fatalf("put repeated user profile: %v", err)
	}

	got, err := store.GetUserProfileByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user profile by user: %v", err)
	}
	if got.Username != "alice_one" {
		t.Fatalf("username = %q, want alice_one", got.Username)
	}
	if !got.CreatedAt.Equal(initial) {
		t.Fatalf("created_at = %v, want %v", got.CreatedAt, initial)
	}
	if !got.UpdatedAt.Equal(initial) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, initial)
	}
}

func TestUserProfileConflictAcrossUsers(t *testing.T) {
	store, err := Open(t.TempDir() + "/social.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, time.February, 22, 16, 0, 0, 0, time.UTC)
	if err := store.PutUserProfile(context.Background(), storage.UserProfile{
		UserID:    "user-1",
		Username:  "conflict",
		Name:      "Alice",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put user profile user-1: %v", err)
	}

	err = store.PutUserProfile(context.Background(), storage.UserProfile{
		UserID:    "user-2",
		Username:  "Conflict",
		Name:      "Bob",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("put user profile user-2 err = %v, want %v", err, storage.ErrAlreadyExists)
	}
}

func TestIsUserProfileUsernameUniqueViolationUsesSQLiteErrorCode(t *testing.T) {
	store, err := Open(t.TempDir() + "/social.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, time.February, 22, 18, 0, 0, 0, time.UTC).UnixMilli()
	if _, err := store.sqlDB.ExecContext(
		context.Background(),
		`INSERT INTO user_profiles (user_id, username, name, avatar_set_id, avatar_asset_id, bio, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"user-1",
		"alice",
		"Alice",
		"",
		"",
		"",
		now,
		now,
	); err != nil {
		t.Fatalf("seed user profile: %v", err)
	}
	_, err = store.sqlDB.ExecContext(
		context.Background(),
		`INSERT INTO user_profiles (user_id, username, name, avatar_set_id, avatar_asset_id, bio, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"user-2",
		"alice",
		"Alice Two",
		"",
		"",
		"",
		now,
		now,
	)
	if err == nil {
		t.Fatal("expected unique constraint error")
	}

	wrapped := opaqueWrapError{cause: err}
	if !isUserProfileUsernameUniqueViolation(wrapped) {
		t.Fatalf("isUserProfileUsernameUniqueViolation(%T) = false, want true", wrapped)
	}
}

func TestIsUserProfileUsernameUniqueViolationFallsBackToMessageWhenSQLiteCodeIsUnexpected(t *testing.T) {
	err := asSQLiteErrorWithUniqueMessage{}
	if !isUserProfileUsernameUniqueViolation(err) {
		t.Fatalf("isUserProfileUsernameUniqueViolation(%T) = false, want true", err)
	}
}

func TestUserProfileGetNotFound(t *testing.T) {
	store, err := Open(t.TempDir() + "/social.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if _, err := store.GetUserProfileByUserID(context.Background(), "missing-user"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("get by user err = %v, want %v", err, storage.ErrNotFound)
	}
	if _, err := store.GetUserProfileByUsername(context.Background(), "missing-username"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("get by username err = %v, want %v", err, storage.ErrNotFound)
	}
}
