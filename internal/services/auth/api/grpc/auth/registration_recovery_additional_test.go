package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	"github.com/louisbranch/fracturing.space/internal/services/auth/recoverycode"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestBeginAccountRegistration_StorePasskeySessionFailureDeletesReservation(t *testing.T) {
	t.Parallel()

	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	passkeyStore.putErr = errors.New("store passkey session boom")

	svc := NewAuthService(userStore, passkeyStore, nil)
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }
	svc.idGenerator = func() (string, error) { return "user-1", nil }
	svc.passkeyIDGenerator = func() (string, error) { return "reg-1", nil }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil

	_, err := svc.BeginAccountRegistration(context.Background(), &authv1.BeginAccountRegistrationRequest{Username: "alpha"})
	grpcassert.StatusCode(t, err, codes.Internal)

	if _, err := passkeyStore.GetRegistrationSession(context.Background(), "reg-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected registration cleanup, got %v", err)
	}
	if _, err := passkeyStore.GetPasskeySession(context.Background(), "reg-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no leaked passkey session, got %v", err)
	}
}

func TestAcknowledgeAccountRegistration_NotReady(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	if err := store.PutRegistrationSession(context.Background(), storage.RegistrationSession{
		ID:        "reg-1",
		UserID:    "user-1",
		Username:  "alpha",
		Locale:    "en-US",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put registration session: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return now }

	_, err := svc.AcknowledgeAccountRegistration(context.Background(), &authv1.AcknowledgeAccountRegistrationRequest{
		SessionId: "reg-1",
	})
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)

	if _, err := store.GetRegistrationSession(context.Background(), "reg-1"); err != nil {
		t.Fatalf("staged registration should remain for retry, got %v", err)
	}
}

func TestAcknowledgeAccountRegistration_PendingAuthorizationFailureLeavesStagedSignup(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	svc := NewAuthService(store, store, openTempOAuthStore(t))
	now := time.Date(2026, 3, 27, 11, 0, 0, 0, time.UTC)
	ids := []string{"user-1", "event-1", "ws-1"}
	svc.idGenerator = func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}
	svc.passkeyIDGenerator = func() (string, error) { return "reg-1", nil }
	svc.clock = func() time.Time { return now }
	svc.randReader = bytes.NewReader(bytes.Repeat([]byte{1}, 64))
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &webauthn.Credential{ID: []byte("cred-1")}}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	beginResp, err := svc.BeginAccountRegistration(context.Background(), &authv1.BeginAccountRegistrationRequest{Username: "alpha"})
	if err != nil {
		t.Fatalf("begin account registration: %v", err)
	}
	if _, err := svc.FinishAccountRegistration(context.Background(), &authv1.FinishAccountRegistrationRequest{
		SessionId:              beginResp.GetSessionId(),
		CredentialResponseJson: []byte("{}"),
	}); err != nil {
		t.Fatalf("finish account registration: %v", err)
	}

	_, err = svc.AcknowledgeAccountRegistration(context.Background(), &authv1.AcknowledgeAccountRegistrationRequest{
		SessionId: beginResp.GetSessionId(),
		PendingId: "missing",
	})
	grpcassert.StatusCode(t, err, codes.NotFound)

	if _, err := store.GetRegistrationSession(context.Background(), beginResp.GetSessionId()); err != nil {
		t.Fatalf("expected staged registration to remain after pending auth failure, got %v", err)
	}
	if _, err := store.GetUser(context.Background(), "user-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no user creation, got %v", err)
	}
}

func TestAcknowledgeAccountRegistration_BuildWebSessionFailureRollsBackPendingAuthorization(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	oauthStore := openTempOAuthStore(t)
	pendingID, err := oauthStore.CreatePendingAuthorization(oauth.AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	}, time.Hour)
	if err != nil {
		t.Fatalf("create pending authorization: %v", err)
	}

	svc := NewAuthService(store, store, oauthStore)
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	ids := []string{"user-1", "event-1"}
	svc.idGenerator = func() (string, error) {
		if len(ids) == 0 {
			return "", errors.New("no more ids")
		}
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}
	svc.passkeyIDGenerator = func() (string, error) { return "reg-1", nil }
	svc.clock = func() time.Time { return now }
	svc.randReader = bytes.NewReader(bytes.Repeat([]byte{2}, 64))
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &webauthn.Credential{ID: []byte("cred-1")}}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	beginResp, err := svc.BeginAccountRegistration(context.Background(), &authv1.BeginAccountRegistrationRequest{Username: "alpha"})
	if err != nil {
		t.Fatalf("begin account registration: %v", err)
	}
	if _, err := svc.FinishAccountRegistration(context.Background(), &authv1.FinishAccountRegistrationRequest{
		SessionId:              beginResp.GetSessionId(),
		CredentialResponseJson: []byte("{}"),
	}); err != nil {
		t.Fatalf("finish account registration: %v", err)
	}

	_, err = svc.AcknowledgeAccountRegistration(context.Background(), &authv1.AcknowledgeAccountRegistrationRequest{
		SessionId: beginResp.GetSessionId(),
		PendingId: pendingID,
	})
	grpcassert.StatusCode(t, err, codes.Internal)

	pending, err := oauthStore.GetPendingAuthorization(pendingID)
	if err != nil {
		t.Fatalf("get pending authorization: %v", err)
	}
	if pending.UserID != "" {
		t.Fatalf("pending authorization user id = %q, want empty", pending.UserID)
	}
	if _, err := store.GetRegistrationSession(context.Background(), beginResp.GetSessionId()); err != nil {
		t.Fatalf("expected staged registration to remain after rollback, got %v", err)
	}
}

func TestBeginAccountRecovery_PutUserFailureDeletesRecoverySession(t *testing.T) {
	t.Parallel()

	code, hash, err := recoverycode.Generate(bytes.NewReader(bytes.Repeat([]byte{3}, 64)))
	if err != nil {
		t.Fatalf("generate recovery code: %v", err)
	}
	now := time.Date(2026, 3, 27, 13, 0, 0, 0, time.UTC)
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{
		ID:               "user-1",
		Username:         "alpha",
		RecoveryCodeHash: hash,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	userStore.putErr = errors.New("reserve boom")
	passkeyStore := newFakePasskeyStore()

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.idGenerator = func() (string, error) { return "recovery-1", nil }

	_, err = svc.BeginAccountRecovery(context.Background(), &authv1.BeginAccountRecoveryRequest{
		Username:     "alpha",
		RecoveryCode: code,
	})
	grpcassert.StatusCode(t, err, codes.Internal)

	if _, err := passkeyStore.GetRecoverySession(context.Background(), "recovery-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected recovery session cleanup, got %v", err)
	}
}

func TestBeginRecoveryPasskeyRegistration_ExpiredRecoverySessionDeletesReservation(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	now := time.Date(2026, 3, 27, 14, 0, 0, 0, time.UTC)
	if err := store.PutUser(context.Background(), user.User{
		ID:                        "user-1",
		Username:                  "alpha",
		RecoveryReservedSessionID: "recovery-1",
		RecoveryReservedUntil:     ptrTime(now.Add(-time.Minute)),
		CreatedAt:                 now.Add(-2 * time.Hour),
		UpdatedAt:                 now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}
	if err := store.PutRecoverySession(context.Background(), storage.RecoverySession{
		ID:        "recovery-1",
		UserID:    "user-1",
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now.Add(-2 * time.Minute),
	}); err != nil {
		t.Fatalf("put recovery session: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return now }

	_, err := svc.BeginRecoveryPasskeyRegistration(context.Background(), &authv1.BeginRecoveryPasskeyRegistrationRequest{
		RecoverySessionId: "recovery-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)

	if _, err := store.GetRecoverySession(context.Background(), "recovery-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected expired recovery session cleanup, got %v", err)
	}
}

func TestFinishRecoveryPasskeyRegistration_AttachesPendingAuthorization(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	oauthStore := openTempOAuthStore(t)
	pendingID, err := oauthStore.CreatePendingAuthorization(oauth.AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	}, time.Hour)
	if err != nil {
		t.Fatalf("create pending authorization: %v", err)
	}

	now := time.Date(2026, 3, 27, 15, 0, 0, 0, time.UTC)
	code, hash, err := recoverycode.Generate(bytes.NewReader(bytes.Repeat([]byte{4}, 64)))
	if err != nil {
		t.Fatalf("generate recovery code: %v", err)
	}
	if err := store.PutUser(context.Background(), user.User{
		ID:               "user-1",
		Username:         "alpha",
		RecoveryCodeHash: hash,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	svc := NewAuthService(store, store, oauthStore)
	ids := []string{"recovery-1", "ws-new"}
	svc.idGenerator = func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}
	svc.passkeyIDGenerator = func() (string, error) { return "passkey-1", nil }
	svc.clock = func() time.Time { return now }
	svc.randReader = bytes.NewReader(bytes.Repeat([]byte{5}, 64))
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &webauthn.Credential{ID: []byte("new-cred")}}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	beginRecovery, err := svc.BeginAccountRecovery(context.Background(), &authv1.BeginAccountRecoveryRequest{
		Username:     "alpha",
		RecoveryCode: code,
	})
	if err != nil {
		t.Fatalf("begin account recovery: %v", err)
	}
	beginPasskey, err := svc.BeginRecoveryPasskeyRegistration(context.Background(), &authv1.BeginRecoveryPasskeyRegistrationRequest{
		RecoverySessionId: beginRecovery.GetRecoverySessionId(),
	})
	if err != nil {
		t.Fatalf("begin recovery passkey registration: %v", err)
	}
	if _, err := svc.FinishRecoveryPasskeyRegistration(context.Background(), &authv1.FinishRecoveryPasskeyRegistrationRequest{
		RecoverySessionId:      beginRecovery.GetRecoverySessionId(),
		SessionId:              beginPasskey.GetSessionId(),
		CredentialResponseJson: []byte("{}"),
		PendingId:              pendingID,
	}); err != nil {
		t.Fatalf("finish recovery passkey registration: %v", err)
	}

	pending, err := oauthStore.GetPendingAuthorization(pendingID)
	if err != nil {
		t.Fatalf("get pending authorization: %v", err)
	}
	if pending.UserID != "user-1" {
		t.Fatalf("pending authorization user id = %q, want user-1", pending.UserID)
	}
}

func TestBeginAccountRegistration_RejectsExistingUserAndReservedUsername(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 27, 16, 0, 0, 0, time.UTC)

	t.Run("existing user", func(t *testing.T) {
		t.Parallel()

		userStore := newFakeUserStore()
		userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
		svc := NewAuthService(userStore, newFakePasskeyStore(), nil)
		svc.clock = func() time.Time { return now }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil

		_, err := svc.BeginAccountRegistration(context.Background(), &authv1.BeginAccountRegistrationRequest{Username: "alpha"})
		grpcassert.StatusCode(t, err, codes.AlreadyExists)
	})

	t.Run("reserved username", func(t *testing.T) {
		t.Parallel()

		passkeyStore := newFakePasskeyStore()
		passkeyStore.registrations["reg-1"] = storage.RegistrationSession{ID: "reg-1", Username: "alpha"}
		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil

		_, err := svc.BeginAccountRegistration(context.Background(), &authv1.BeginAccountRegistrationRequest{Username: "alpha"})
		grpcassert.StatusCode(t, err, codes.AlreadyExists)
	})
}

func TestFinishAccountRegistration_GuardsAndExpiryCleanup(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 27, 16, 30, 0, 0, time.UTC)

	t.Run("expired session deletes registration and passkey state", func(t *testing.T) {
		t.Parallel()

		passkeyStore := newFakePasskeyStore()
		passkeyStore.registrations["reg-1"] = storage.RegistrationSession{
			ID:        "reg-1",
			UserID:    "user-1",
			Username:  "alpha",
			ExpiresAt: now.Add(-time.Minute),
		}
		passkeyStore.sessions["reg-1"] = storage.PasskeySession{ID: "reg-1", UserID: "user-1"}
		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }

		_, err := svc.FinishAccountRegistration(context.Background(), &authv1.FinishAccountRegistrationRequest{
			SessionId:              "reg-1",
			CredentialResponseJson: []byte("{}"),
		})
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
		if _, err := passkeyStore.GetRegistrationSession(context.Background(), "reg-1"); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("expected expired registration cleanup, got %v", err)
		}
		if _, err := passkeyStore.GetPasskeySession(context.Background(), "reg-1"); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("expected expired passkey session cleanup, got %v", err)
		}
	})

	t.Run("staged registration cannot finish twice", func(t *testing.T) {
		t.Parallel()

		passkeyStore := newFakePasskeyStore()
		passkeyStore.registrations["reg-1"] = storage.RegistrationSession{
			ID:             "reg-1",
			UserID:         "user-1",
			Username:       "alpha",
			ExpiresAt:      now.Add(time.Hour),
			CredentialID:   "cred-1",
			CredentialJSON: "{}",
		}
		passkeyStore.sessions["reg-1"] = storage.PasskeySession{
			ID:          "reg-1",
			UserID:      "user-1",
			Kind:        "registration",
			SessionJSON: "{}",
			ExpiresAt:   now.Add(time.Hour),
		}
		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }

		_, err := svc.FinishAccountRegistration(context.Background(), &authv1.FinishAccountRegistrationRequest{
			SessionId:              "reg-1",
			CredentialResponseJson: []byte("{}"),
		})
		grpcassert.StatusCode(t, err, codes.FailedPrecondition)
	})
}

func TestRegistrationRecoverySessionLoadHelpers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 27, 17, 0, 0, 0, time.UTC)

	t.Run("load active registration session handles missing and expired rows", func(t *testing.T) {
		t.Parallel()

		passkeyStore := newFakePasskeyStore()
		passkeyStore.registrations["reg-expired"] = storage.RegistrationSession{
			ID:        "reg-expired",
			ExpiresAt: now.Add(-time.Minute),
		}
		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }

		_, err := svc.loadActiveRegistrationSession(context.Background(), "")
		grpcassert.StatusCode(t, err, codes.InvalidArgument)

		_, err = svc.loadActiveRegistrationSession(context.Background(), "missing")
		grpcassert.StatusCode(t, err, codes.NotFound)

		_, err = svc.loadActiveRegistrationSession(context.Background(), "reg-expired")
		grpcassert.StatusCode(t, err, codes.FailedPrecondition)
		if _, err := passkeyStore.GetRegistrationSession(context.Background(), "reg-expired"); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("expected expired registration cleanup, got %v", err)
		}
	})

	t.Run("load recovery session user handles mismatch and expiry", func(t *testing.T) {
		t.Parallel()

		store := openTempAuthStore(t)
		if err := store.PutUser(context.Background(), user.User{
			ID:                        "user-1",
			Username:                  "alpha",
			RecoveryReservedSessionID: "other-session",
			CreatedAt:                 now,
			UpdatedAt:                 now,
		}); err != nil {
			t.Fatalf("put user: %v", err)
		}
		if err := store.PutRecoverySession(context.Background(), storage.RecoverySession{
			ID:        "recovery-1",
			UserID:    "user-1",
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		}); err != nil {
			t.Fatalf("put recovery session: %v", err)
		}
		if err := store.PutRecoverySession(context.Background(), storage.RecoverySession{
			ID:        "expired",
			UserID:    "user-1",
			ExpiresAt: now.Add(-time.Minute),
			CreatedAt: now.Add(-time.Hour),
		}); err != nil {
			t.Fatalf("put expired recovery session: %v", err)
		}

		svc := NewAuthService(store, store, nil)
		svc.clock = func() time.Time { return now }

		_, _, err := svc.loadRecoverySessionUser(context.Background(), "")
		grpcassert.StatusCode(t, err, codes.InvalidArgument)

		_, _, err = svc.loadRecoverySessionUser(context.Background(), "expired")
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
		if _, err := store.GetRecoverySession(context.Background(), "expired"); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("expected expired recovery cleanup, got %v", err)
		}

		_, _, err = svc.loadRecoverySessionUser(context.Background(), "recovery-1")
		grpcassert.StatusCode(t, err, codes.FailedPrecondition)
	})
}

func TestBeginAccountRecovery_RejectsActiveReservationAndBadCode(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 27, 17, 30, 0, 0, time.UTC)
	code, hash, err := recoverycode.Generate(bytes.NewReader(bytes.Repeat([]byte{6}, 64)))
	if err != nil {
		t.Fatalf("generate recovery code: %v", err)
	}

	t.Run("active reservation", func(t *testing.T) {
		t.Parallel()

		userStore := newFakeUserStore()
		userStore.users["user-1"] = user.User{
			ID:                        "user-1",
			Username:                  "alpha",
			RecoveryCodeHash:          hash,
			RecoveryReservedSessionID: "recovery-1",
			RecoveryReservedUntil:     ptrTime(now.Add(time.Hour)),
			CreatedAt:                 now,
			UpdatedAt:                 now,
		}
		svc := NewAuthService(userStore, newFakePasskeyStore(), nil)
		svc.clock = func() time.Time { return now }

		_, err := svc.BeginAccountRecovery(context.Background(), &authv1.BeginAccountRecoveryRequest{
			Username:     "alpha",
			RecoveryCode: code,
		})
		grpcassert.StatusCode(t, err, codes.FailedPrecondition)
	})

	t.Run("invalid recovery code", func(t *testing.T) {
		t.Parallel()

		userStore := newFakeUserStore()
		userStore.users["user-1"] = user.User{
			ID:               "user-1",
			Username:         "alpha",
			RecoveryCodeHash: hash,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		svc := NewAuthService(userStore, newFakePasskeyStore(), nil)
		svc.clock = func() time.Time { return now }

		_, err := svc.BeginAccountRecovery(context.Background(), &authv1.BeginAccountRecoveryRequest{
			Username:     "alpha",
			RecoveryCode: "WRONG-WRONG-WRONG",
		})
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
	})
}

func TestFinishRecoveryPasskeyRegistration_ErrorTail(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 27, 18, 0, 0, 0, time.UTC)

	t.Run("passkey session user mismatch", func(t *testing.T) {
		t.Parallel()

		store := openTempAuthStore(t)
		if err := store.PutUser(context.Background(), user.User{
			ID:                        "user-1",
			Username:                  "alpha",
			RecoveryReservedSessionID: "recovery-1",
			RecoveryReservedUntil:     ptrTime(now.Add(time.Hour)),
			CreatedAt:                 now,
			UpdatedAt:                 now,
		}); err != nil {
			t.Fatalf("put user: %v", err)
		}
		if err := store.PutRecoverySession(context.Background(), storage.RecoverySession{
			ID:        "recovery-1",
			UserID:    "user-1",
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		}); err != nil {
			t.Fatalf("put recovery session: %v", err)
		}
		if err := store.PutPasskeySession(context.Background(), storage.PasskeySession{
			ID:          "passkey-1",
			UserID:      "other-user",
			Kind:        "registration",
			SessionJSON: "{}",
			ExpiresAt:   now.Add(time.Hour),
		}); err != nil {
			t.Fatalf("put passkey session: %v", err)
		}

		svc := NewAuthService(store, store, nil)
		svc.clock = func() time.Time { return now }

		_, err := svc.FinishRecoveryPasskeyRegistration(context.Background(), &authv1.FinishRecoveryPasskeyRegistrationRequest{
			RecoverySessionId:      "recovery-1",
			SessionId:              "passkey-1",
			CredentialResponseJson: []byte("{}"),
		})
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
	})

	t.Run("list passkeys surfaces store failures", func(t *testing.T) {
		t.Parallel()

		passkeyStore := newFakePasskeyStore()
		passkeyStore.listErr = errors.New("list boom")
		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)

		_, err := svc.ListPasskeys(context.Background(), &authv1.ListPasskeysRequest{UserId: "user-1"})
		grpcassert.StatusCode(t, err, codes.Internal)
	})
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
