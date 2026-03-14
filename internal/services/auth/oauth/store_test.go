package oauth

import (
	"testing"
	"time"

	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
)

func TestStoreEnsureDBNilSafe(t *testing.T) {
	var nilStore *Store
	if err := nilStore.ensureDB(); err == nil {
		t.Error("expected error from nil store")
	}

	store := &Store{}
	if err := store.ensureDB(); err == nil {
		t.Error("expected error from store with nil db")
	}
}

func TestStoreDeleteAccessTokenNilSafe(t *testing.T) {
	var nilStore *Store
	nilStore.DeleteAccessToken("token") // should not panic

	store := &Store{}
	store.DeleteAccessToken("token") // should not panic
}

func TestStoreDeleteAuthorizationCodeNilSafe(t *testing.T) {
	var nilStore *Store
	nilStore.DeleteAuthorizationCode("code") // should not panic

	store := &Store{}
	store.DeleteAuthorizationCode("code") // should not panic
}

func TestStoreDeletePendingAuthorizationNilSafe(t *testing.T) {
	var nilStore *Store
	nilStore.DeletePendingAuthorization("id") // should not panic

	store := &Store{}
	store.DeletePendingAuthorization("id") // should not panic
}

func TestStoreCleanupExpiredNilSafe(t *testing.T) {
	var nilStore *Store
	nilStore.CleanupExpired(time.Now()) // should not panic

	store := &Store{}
	store.CleanupExpired(time.Now()) // should not panic
}

func TestStoreOperationsWithDB(t *testing.T) {
	path := t.TempDir() + "/auth.db"
	authStore, err := authsqlite.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { authStore.Close() })

	store := NewStore(authStore.DB())

	t.Run("CreateAndValidateAccessToken", func(t *testing.T) {
		token, err := store.CreateAccessToken("client-1", "user-1", "openid", time.Hour)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		if token.Token == "" {
			t.Fatal("expected non-empty token")
		}

		// Validate the token.
		access, ok, err := store.ValidateAccessToken(token.Token)
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if !ok {
			t.Fatal("expected token to be valid")
		}
		if access.UserID != "user-1" {
			t.Errorf("expected user_id %q, got %q", "user-1", access.UserID)
		}

		// Validate with non-existent token.
		_, ok, err = store.ValidateAccessToken("nonexistent")
		if err != nil {
			t.Fatalf("validate nonexistent: %v", err)
		}
		if ok {
			t.Error("expected invalid for nonexistent token")
		}

		// Delete and re-validate.
		store.DeleteAccessToken(token.Token)
		_, ok, err = store.ValidateAccessToken(token.Token)
		if err != nil {
			t.Fatalf("validate deleted: %v", err)
		}
		if ok {
			t.Error("expected invalid after deletion")
		}
	})

	t.Run("MarkAuthorizationCodeUsed", func(t *testing.T) {
		code, err := store.CreateAuthorizationCode(AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "client-1",
			RedirectURI:         "http://localhost/cb",
			CodeChallenge:       "challenge",
			CodeChallengeMethod: "S256",
			Scope:               "openid",
		}, "user-1", time.Hour)
		if err != nil {
			t.Fatalf("create code: %v", err)
		}

		// First mark: should succeed.
		used, err := store.MarkAuthorizationCodeUsed(code.Code)
		if err != nil {
			t.Fatalf("mark used: %v", err)
		}
		if !used {
			t.Error("expected used=true on first mark")
		}

		// Second mark: should return false (already used).
		used, err = store.MarkAuthorizationCodeUsed(code.Code)
		if err != nil {
			t.Fatalf("mark used again: %v", err)
		}
		if used {
			t.Error("expected used=false on second mark")
		}

		// Verify GetAuthorizationCode shows used=true.
		stored, err := store.GetAuthorizationCode(code.Code)
		if err != nil {
			t.Fatalf("get code: %v", err)
		}
		if !stored.Used {
			t.Error("expected stored code to be marked as used")
		}
	})

	t.Run("GetAccessToken", func(t *testing.T) {
		token, err := store.CreateAccessToken("c2", "u2", "profile", time.Hour)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		got, err := store.GetAccessToken(token.Token)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.ClientID != "c2" || got.UserID != "u2" {
			t.Errorf("unexpected token: %+v", got)
		}

		// Non-existent token.
		got, err = store.GetAccessToken("does-not-exist")
		if err != nil {
			t.Fatalf("get nonexistent: %v", err)
		}
		if got != nil {
			t.Error("expected nil for nonexistent token")
		}
	})

	t.Run("ValidateExpiredToken", func(t *testing.T) {
		token, err := store.CreateAccessToken("c3", "u3", "", 1*time.Nanosecond)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		time.Sleep(2 * time.Millisecond)

		_, ok, err := store.ValidateAccessToken(token.Token)
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if ok {
			t.Error("expected expired token to be invalid")
		}
	})

	t.Run("PendingAuthorizationLifecycle", func(t *testing.T) {
		id, err := store.CreatePendingAuthorization(AuthorizationRequest{
			ResponseType: "code",
			ClientID:     "c1",
			RedirectURI:  "http://localhost/cb",
		}, time.Hour)
		if err != nil {
			t.Fatalf("create pending: %v", err)
		}

		pending, err := store.GetPendingAuthorization(id)
		if err != nil {
			t.Fatalf("get pending: %v", err)
		}
		if pending.Request.ClientID != "c1" {
			t.Errorf("expected client_id %q, got %q", "c1", pending.Request.ClientID)
		}

		if err := store.UpdatePendingAuthorizationUserID(id, "user-5"); err != nil {
			t.Fatalf("update user_id: %v", err)
		}

		pending, err = store.GetPendingAuthorization(id)
		if err != nil {
			t.Fatalf("get after update: %v", err)
		}
		if pending.UserID != "user-5" {
			t.Errorf("expected user_id %q, got %q", "user-5", pending.UserID)
		}

		store.DeletePendingAuthorization(id)
		pending, err = store.GetPendingAuthorization(id)
		if err != nil {
			t.Fatalf("get after delete: %v", err)
		}
		if pending != nil {
			t.Error("expected nil after deletion")
		}
	})

	t.Run("CleanupExpired", func(t *testing.T) {
		// Create items that expire immediately.
		token, _ := store.CreateAccessToken("cx", "ux", "", 1*time.Nanosecond)
		code, _ := store.CreateAuthorizationCode(AuthorizationRequest{
			ResponseType:  "code",
			ClientID:      "cx",
			RedirectURI:   "http://localhost/cb",
			CodeChallenge: "ch",
		}, "ux", 1*time.Nanosecond)
		time.Sleep(2 * time.Millisecond)

		store.CleanupExpired(time.Now().UTC())

		// Verify tokens and codes are cleaned up.
		got, _ := store.GetAccessToken(token.Token)
		if got != nil {
			t.Error("expected access token to be cleaned up")
		}
		gotCode, _ := store.GetAuthorizationCode(code.Code)
		if gotCode != nil {
			t.Error("expected auth code to be cleaned up")
		}
	})
}

func TestGenerateToken(t *testing.T) {
	token, err := generateToken(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token) != 32 { // hex-encoded 16 bytes = 32 chars
		t.Errorf("expected 32-char token, got %d: %q", len(token), token)
	}

	token2, _ := generateToken(16)
	if token == token2 {
		t.Error("expected unique tokens")
	}
}
