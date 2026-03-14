package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/social/storage"
)

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

func TestUserProfileRoundTripAndUpdate(t *testing.T) {
	store, err := Open(t.TempDir() + "/social.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	createdAt := time.Date(2026, time.February, 22, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)
	if err := store.PutUserProfile(context.Background(), storage.UserProfile{
		UserID:        "user-1",
		Name:          "Alice",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "001",
		Bio:           "Campaign manager",
		Pronouns:      "she/her",
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}); err != nil {
		t.Fatalf("put user profile: %v", err)
	}

	got, err := store.GetUserProfileByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user profile by user: %v", err)
	}
	if got.UserID != "user-1" || got.Name != "Alice" || got.Pronouns != "she/her" {
		t.Fatalf("unexpected profile: %+v", got)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %v, want %v", got.CreatedAt, createdAt)
	}

	if err := store.PutUserProfile(context.Background(), storage.UserProfile{
		UserID:        "user-1",
		Name:          "Alice Two",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "002",
		Bio:           "Updated",
		Pronouns:      "they/them",
		CreatedAt:     updatedAt,
		UpdatedAt:     updatedAt,
	}); err != nil {
		t.Fatalf("update user profile: %v", err)
	}

	got, err = store.GetUserProfileByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get updated user profile: %v", err)
	}
	if got.Name != "Alice Two" || got.Bio != "Updated" || got.Pronouns != "they/them" {
		t.Fatalf("unexpected updated profile: %+v", got)
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
		Name:          "Alice",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "001",
		Bio:           "Campaign manager",
		Pronouns:      "she/her",
		CreatedAt:     initial,
		UpdatedAt:     initial,
	}); err != nil {
		t.Fatalf("put initial user profile: %v", err)
	}

	retryAt := initial.Add(2 * time.Hour)
	if err := store.PutUserProfile(context.Background(), storage.UserProfile{
		UserID:        "user-1",
		Name:          "Alice",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "001",
		Bio:           "Campaign manager",
		Pronouns:      "she/her",
		CreatedAt:     retryAt,
		UpdatedAt:     retryAt,
	}); err != nil {
		t.Fatalf("put repeated user profile: %v", err)
	}

	got, err := store.GetUserProfileByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user profile by user: %v", err)
	}
	if !got.CreatedAt.Equal(initial) {
		t.Fatalf("created_at = %v, want %v", got.CreatedAt, initial)
	}
	if !got.UpdatedAt.Equal(initial) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, initial)
	}
}

func TestUserProfileNotFound(t *testing.T) {
	store, err := Open(t.TempDir() + "/social.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if _, err := store.GetUserProfileByUserID(context.Background(), "missing-user"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("get user profile err = %v, want %v", err, storage.ErrNotFound)
	}
}
