package sqlite

import (
	"context"
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

func TestPutGetAccountProfileRoundTrip(t *testing.T) {
	store := openTempStore(t)

	createdAt := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	userRecord := user.User{
		ID:        "user-1",
		Email:     "testuser",
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if err := store.PutUser(context.Background(), userRecord); err != nil {
		t.Fatalf("put user: %v", err)
	}

	profile := storage.AccountProfile{
		UserID:    userRecord.ID,
		Name:      "Alice",
		Locale:    commonv1.Locale_LOCALE_EN_US,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if err := store.PutAccountProfile(context.Background(), profile); err != nil {
		t.Fatalf("put profile: %v", err)
	}

	got, err := store.GetAccountProfile(context.Background(), userRecord.ID)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if got.UserID != profile.UserID || got.Name != profile.Name || got.Locale != profile.Locale {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestGetAccountProfileNotFound(t *testing.T) {
	store := openTempStore(t)
	_, err := store.GetAccountProfile(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected storage.ErrNotFound, got %v", err)
	}
}

func TestPutAccountProfileRequiresUserID(t *testing.T) {
	store := openTempStore(t)
	err := store.PutAccountProfile(context.Background(), storage.AccountProfile{
		UserID: " ",
	})
	if err == nil {
		t.Fatal("expected error for missing user id")
	}
}

func TestPutGetUserRoundTrip(t *testing.T) {
	store := openTempStore(t)

	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	input := user.User{
		ID:        "user-1",
		Email:     "testuser",
		Locale:    commonv1.Locale_LOCALE_PT_BR,
		CreatedAt: created,
		UpdatedAt: updated,
	}

	if err := store.PutUser(context.Background(), input); err != nil {
		t.Fatalf("put user: %v", err)
	}

	got, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.ID != input.ID || got.Email != input.Email {
		t.Fatalf("unexpected user: %+v", got)
	}
	if got.Locale != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("locale = %v, want %v", got.Locale, commonv1.Locale_LOCALE_PT_BR)
	}
}

func TestPutUserDefaultsLocaleWhenUnset(t *testing.T) {
	store := openTempStore(t)

	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	input := user.User{
		ID:        "user-1",
		Email:     "testuser",
		Locale:    commonv1.Locale_LOCALE_UNSPECIFIED,
		CreatedAt: now,
		UpdatedAt: now,
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

func TestPutUserRequiresID(t *testing.T) {
	store := openTempStore(t)

	err := store.PutUser(context.Background(), user.User{ID: "  "})
	if err == nil {
		t.Fatal("expected error for empty user id")
	}
}

func TestPutUserRequiresEmail(t *testing.T) {
	store := openTempStore(t)

	err := store.PutUser(context.Background(), user.User{
		ID:        "user-1",
		Email:     " ",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestPutUserEnforcesPrimaryEmailUniqueness(t *testing.T) {
	store := openTempStore(t)

	if err := store.PutUser(context.Background(), user.User{
		ID:        "user-1",
		Email:     "shared@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	err := store.PutUser(context.Background(), user.User{
		ID:        "user-2",
		Email:     "shared@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected duplicate email error")
	}
	if _, err := store.GetUser(context.Background(), "user-2"); err != storage.ErrNotFound {
		t.Fatalf("expected user-2 not found, got %v", err)
	}
}

func TestPutUserIsIdempotentForPrimaryEmail(t *testing.T) {
	store := openTempStore(t)

	created := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Minute)
	input := user.User{
		ID:        "user-1",
		Email:     "testuser",
		CreatedAt: created,
		UpdatedAt: created,
	}

	if err := store.PutUser(context.Background(), input); err != nil {
		t.Fatalf("put user: %v", err)
	}
	input.CreatedAt = updated
	input.UpdatedAt = updated
	if err := store.PutUser(context.Background(), input); err != nil {
		t.Fatalf("put user again: %v", err)
	}

	list, err := store.ListUserEmailsByUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list user emails: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 email, got %d", len(list))
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

	for i, id := range []string{"user-1", "user-2", "user-3"} {
		if err := store.PutUser(context.Background(), user.User{
			ID:        id,
			Email:     fmt.Sprintf("user%d", i+1),
			CreatedAt: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
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

func TestContactRoundTripAndOwnerScoping(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 20, 40, 0, 0, time.UTC)

	for i, u := range []user.User{
		{ID: "user-1", Email: "user1@example.com", CreatedAt: now, UpdatedAt: now},
		{ID: "user-2", Email: "user2@example.com", CreatedAt: now, UpdatedAt: now},
		{ID: "user-3", Email: "user3@example.com", CreatedAt: now, UpdatedAt: now},
		{ID: "user-4", Email: "user4@example.com", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutUser(context.Background(), u); err != nil {
			t.Fatalf("put user %d: %v", i+1, err)
		}
	}

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

func TestGetContact(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 20, 40, 0, 0, time.UTC)

	for _, u := range []user.User{
		{ID: "user-1", Email: "user1@example.com", CreatedAt: now, UpdatedAt: now},
		{ID: "user-2", Email: "user2@example.com", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutUser(context.Background(), u); err != nil {
			t.Fatalf("put user %s: %v", u.ID, err)
		}
	}
	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-1",
		ContactUserID: "user-2",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put contact: %v", err)
	}

	got, err := store.GetContact(context.Background(), "user-1", "user-2")
	if err != nil {
		t.Fatalf("get contact: %v", err)
	}
	if got.OwnerUserID != "user-1" || got.ContactUserID != "user-2" {
		t.Fatalf("unexpected contact: %+v", got)
	}
}

func TestGetContactNotFound(t *testing.T) {
	store := openTempStore(t)

	_, err := store.GetContact(context.Background(), "user-1", "user-2")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected storage.ErrNotFound, got %v", err)
	}
}

func TestPutContactIdempotent(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 20, 41, 0, 0, time.UTC)
	later := now.Add(5 * time.Minute)

	if err := store.PutUser(context.Background(), user.User{ID: "user-1", Email: "user1@example.com", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("put user-1: %v", err)
	}
	if err := store.PutUser(context.Background(), user.User{ID: "user-2", Email: "user2@example.com", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("put user-2: %v", err)
	}

	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-1",
		ContactUserID: "user-2",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put contact: %v", err)
	}
	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-1",
		ContactUserID: "user-2",
		CreatedAt:     later,
		UpdatedAt:     later,
	}); err != nil {
		t.Fatalf("put contact again: %v", err)
	}

	page, err := store.ListContacts(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(page.Contacts) != 1 {
		t.Fatalf("contacts len = %d, want 1", len(page.Contacts))
	}
	if !page.Contacts[0].CreatedAt.Equal(now) {
		t.Fatalf("created_at = %v, want %v", page.Contacts[0].CreatedAt, now)
	}
	if !page.Contacts[0].UpdatedAt.Equal(later) {
		t.Fatalf("updated_at = %v, want %v", page.Contacts[0].UpdatedAt, later)
	}
}

func TestDeleteContactIdempotent(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 20, 42, 0, 0, time.UTC)

	if err := store.PutUser(context.Background(), user.User{ID: "user-1", Email: "user1@example.com", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("put user-1: %v", err)
	}
	if err := store.PutUser(context.Background(), user.User{ID: "user-2", Email: "user2@example.com", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("put user-2: %v", err)
	}
	if err := store.PutContact(context.Background(), storage.Contact{
		OwnerUserID:   "user-1",
		ContactUserID: "user-2",
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("put contact: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := store.DeleteContact(context.Background(), "user-1", "user-2"); err != nil {
			t.Fatalf("delete contact %d: %v", i+1, err)
		}
	}

	page, err := store.ListContacts(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(page.Contacts) != 0 {
		t.Fatalf("contacts len = %d, want 0", len(page.Contacts))
	}
}

func TestListContactsPagination(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 20, 43, 0, 0, time.UTC)

	owner := user.User{ID: "user-1", Email: "user1@example.com", CreatedAt: now, UpdatedAt: now}
	if err := store.PutUser(context.Background(), owner); err != nil {
		t.Fatalf("put owner: %v", err)
	}
	for _, id := range []string{"user-2", "user-3", "user-4"} {
		if err := store.PutUser(context.Background(), user.User{
			ID:        id,
			Email:     id + "@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			t.Fatalf("put contact user %s: %v", id, err)
		}
		if err := store.PutContact(context.Background(), storage.Contact{
			OwnerUserID:   "user-1",
			ContactUserID: id,
			CreatedAt:     now,
			UpdatedAt:     now,
		}); err != nil {
			t.Fatalf("put contact %s: %v", id, err)
		}
	}

	first, err := store.ListContacts(context.Background(), "user-1", 2, "")
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first.Contacts) != 2 {
		t.Fatalf("first page contacts len = %d, want 2", len(first.Contacts))
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	second, err := store.ListContacts(context.Background(), "user-1", 2, first.NextPageToken)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second.Contacts) != 1 {
		t.Fatalf("second page contacts len = %d, want 1", len(second.Contacts))
	}
	if second.NextPageToken != "" {
		t.Fatalf("next page token = %q, want empty", second.NextPageToken)
	}
}

func TestPutContactValidationErrors(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 21, 20, 44, 0, 0, time.UTC)

	if err := store.PutUser(context.Background(), user.User{ID: "user-1", Email: "user1@example.com", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("put user-1: %v", err)
	}
	if err := store.PutUser(context.Background(), user.User{ID: "user-2", Email: "user2@example.com", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("put user-2: %v", err)
	}

	cases := []struct {
		name    string
		contact storage.Contact
	}{
		{
			name: "missing owner",
			contact: storage.Contact{
				OwnerUserID:   " ",
				ContactUserID: "user-2",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
		{
			name: "missing contact",
			contact: storage.Contact{
				OwnerUserID:   "user-1",
				ContactUserID: " ",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
		{
			name: "self contact",
			contact: storage.Contact{
				OwnerUserID:   "user-1",
				ContactUserID: "user-1",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := store.PutContact(context.Background(), tc.contact)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestListContactsValidationErrors(t *testing.T) {
	store := openTempStore(t)

	if _, err := store.ListContacts(context.Background(), " ", 10, ""); err == nil {
		t.Fatal("expected error for missing owner user id")
	}
	if _, err := store.ListContacts(context.Background(), "user-1", 0, ""); err == nil {
		t.Fatal("expected error for invalid page size")
	}
}

func TestListContactsContextError(t *testing.T) {
	store := openTempStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.ListContacts(ctx, "user-1", 10, "")
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestGetAuthStatisticsSince(t *testing.T) {
	store := openTempStore(t)

	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	if err := store.PutUser(context.Background(), user.User{
		ID:        "user-1",
		Email:     "testuser",
		CreatedAt: created,
		UpdatedAt: created,
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
		ID:        "user-1",
		Email:     "testuser",
		CreatedAt: created,
		UpdatedAt: created,
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
		ID:        "user-1",
		Email:     "testuser",
		CreatedAt: now,
		UpdatedAt: now,
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
		ID:        "user-1",
		Email:     "testuser",
		CreatedAt: now,
		UpdatedAt: now,
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
	if len(list) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(list))
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

func TestGetUserUsesPrimaryEmail(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	if err := store.PutUser(context.Background(), user.User{
		ID:        "user-1",
		Email:     "primary@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	if err := store.PutUserEmail(context.Background(), storage.UserEmail{
		ID:        "email-2",
		UserID:    "user-1",
		Email:     "secondary@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put user email: %v", err)
	}

	got, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.Email != "primary@example.com" {
		t.Fatalf("expected primary email, got %q", got.Email)
	}

	list, err := store.ListUserEmailsByUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list emails: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(list))
	}
}

func TestPutUserEmailDoesNotDemotePrimary(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	if err := store.PutUser(context.Background(), user.User{
		ID:        "user-1",
		Email:     "primary@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	if err := store.PutUserEmail(context.Background(), storage.UserEmail{
		ID:        "email-primary",
		UserID:    "user-1",
		Email:     "primary@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put user email: %v", err)
	}

	if _, err := store.GetUser(context.Background(), "user-1"); err != nil {
		t.Fatalf("get user before re-upsert: %v", err)
	}

	if err := store.PutUserEmail(context.Background(), storage.UserEmail{
		ID:        "email-primary-updated",
		UserID:    "user-1",
		Email:     "primary@example.com",
		CreatedAt: now,
		UpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("upsert primary email: %v", err)
	}

	if _, err := store.GetUser(context.Background(), "user-1"); err != nil {
		t.Fatalf("expected primary email to remain after upsert: %v", err)
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
