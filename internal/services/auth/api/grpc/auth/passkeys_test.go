package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	"github.com/louisbranch/fracturing.space/internal/services/auth/passkey"
	"github.com/louisbranch/fracturing.space/internal/services/auth/recoverycode"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

type fakePasskeyStore struct {
	sessions      map[string]storage.PasskeySession
	credentials   map[string]storage.PasskeyCredential
	registrations map[string]storage.RegistrationSession
	recoveries    map[string]storage.RecoverySession
	putErr        error
	getErr        error
	listErr       error
}

func newFakePasskeyStore() *fakePasskeyStore {
	return &fakePasskeyStore{
		sessions:      make(map[string]storage.PasskeySession),
		credentials:   make(map[string]storage.PasskeyCredential),
		registrations: make(map[string]storage.RegistrationSession),
		recoveries:    make(map[string]storage.RecoverySession),
	}
}

func (s *fakePasskeyStore) PutPasskeyCredential(_ context.Context, credential storage.PasskeyCredential) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.credentials[credential.CredentialID] = credential
	return nil
}

func (s *fakePasskeyStore) GetPasskeyCredential(_ context.Context, credentialID string) (storage.PasskeyCredential, error) {
	if s.getErr != nil {
		return storage.PasskeyCredential{}, s.getErr
	}
	credential, ok := s.credentials[credentialID]
	if !ok {
		return storage.PasskeyCredential{}, storage.ErrNotFound
	}
	return credential, nil
}

func (s *fakePasskeyStore) ListPasskeyCredentials(_ context.Context, userID string) ([]storage.PasskeyCredential, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	credentials := make([]storage.PasskeyCredential, 0)
	for _, credential := range s.credentials {
		if credential.UserID == userID {
			credentials = append(credentials, credential)
		}
	}
	return credentials, nil
}

func (s *fakePasskeyStore) DeletePasskeyCredential(_ context.Context, credentialID string) error {
	delete(s.credentials, credentialID)
	return nil
}

func (s *fakePasskeyStore) DeletePasskeyCredentialsByUser(_ context.Context, userID string) error {
	for id, credential := range s.credentials {
		if credential.UserID == userID {
			delete(s.credentials, id)
		}
	}
	return nil
}

func (s *fakePasskeyStore) DeletePasskeyCredentialsByUserExcept(_ context.Context, userID string, credentialID string) error {
	for id, credential := range s.credentials {
		if credential.UserID == userID && id != credentialID {
			delete(s.credentials, id)
		}
	}
	return nil
}

func (s *fakePasskeyStore) PutPasskeySession(_ context.Context, session storage.PasskeySession) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.sessions[session.ID] = session
	return nil
}

func (s *fakePasskeyStore) GetPasskeySession(_ context.Context, id string) (storage.PasskeySession, error) {
	if s.getErr != nil {
		return storage.PasskeySession{}, s.getErr
	}
	session, ok := s.sessions[id]
	if !ok {
		return storage.PasskeySession{}, storage.ErrNotFound
	}
	return session, nil
}

func (s *fakePasskeyStore) DeletePasskeySession(_ context.Context, id string) error {
	delete(s.sessions, id)
	return nil
}

func (s *fakePasskeyStore) DeleteExpiredPasskeySessions(_ context.Context, _ time.Time) error {
	return nil
}

func (s *fakePasskeyStore) PutRegistrationSession(_ context.Context, session storage.RegistrationSession) error {
	s.registrations[session.ID] = session
	return nil
}

func (s *fakePasskeyStore) GetRegistrationSession(_ context.Context, id string) (storage.RegistrationSession, error) {
	session, ok := s.registrations[id]
	if !ok {
		return storage.RegistrationSession{}, storage.ErrNotFound
	}
	return session, nil
}

func (s *fakePasskeyStore) GetRegistrationSessionByUsername(_ context.Context, username string) (storage.RegistrationSession, error) {
	for _, session := range s.registrations {
		if session.Username == username {
			return session, nil
		}
	}
	return storage.RegistrationSession{}, storage.ErrNotFound
}

func (s *fakePasskeyStore) DeleteRegistrationSession(_ context.Context, id string) error {
	delete(s.registrations, id)
	return nil
}

func (s *fakePasskeyStore) DeleteExpiredRegistrationSessions(_ context.Context, _ time.Time) error {
	return nil
}

func (s *fakePasskeyStore) PutRecoverySession(_ context.Context, session storage.RecoverySession) error {
	s.recoveries[session.ID] = session
	return nil
}

func (s *fakePasskeyStore) GetRecoverySession(_ context.Context, id string) (storage.RecoverySession, error) {
	session, ok := s.recoveries[id]
	if !ok {
		return storage.RecoverySession{}, storage.ErrNotFound
	}
	return session, nil
}

func (s *fakePasskeyStore) DeleteRecoverySession(_ context.Context, id string) error {
	delete(s.recoveries, id)
	return nil
}

func (s *fakePasskeyStore) DeleteExpiredRecoverySessions(_ context.Context, _ time.Time) error {
	return nil
}

func TestBeginPasskeyRegistration_Success(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()

	svc := NewAuthService(userStore, passkeyStore, nil)
	fixed := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return fixed }
	svc.passkeyIDGenerator = func() (string, error) { return "session-1", nil }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{}

	resp, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("begin passkey registration: %v", err)
	}
	if got := resp.GetSessionId(); got != "session-1" {
		t.Fatalf("session id = %q, want %q", got, "session-1")
	}
	if len(resp.GetCredentialCreationOptionsJson()) == 0 {
		t.Fatal("expected creation options")
	}
	if stored := passkeyStore.sessions["session-1"]; stored.UserID != "user-1" || stored.Kind != string(passkey.SessionKindRegistration) {
		t.Fatalf("stored session = %+v", stored)
	}
}

func TestBeginPasskeyRegistration_LoadPasskeyUserDecodeFailure(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.credentials["cred-1"] = storage.PasskeyCredential{
		CredentialID:   "cred-1",
		UserID:         "user-1",
		CredentialJSON: "{",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyIDGenerator = func() (string, error) { return "session-1", nil }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{}

	_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestBeginPasskeyRegistration_BeginRegistrationFailure(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyWebAuthn = &fakePasskeyProvider{beginRegistrationErr: errors.New("registration unavailable")}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{}

	_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestPasskeyEndpointGuards(t *testing.T) {
	t.Run("begin registration", func(t *testing.T) {
		tests := []struct {
			name string
			svc  *AuthService
			req  *authv1.BeginPasskeyRegistrationRequest
			want codes.Code
		}{
			{
				name: "nil request",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  nil,
				want: codes.InvalidArgument,
			},
			{
				name: "missing user store",
				svc: func() *AuthService {
					svc := NewAuthService(nil, newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"},
				want: codes.Internal,
			},
			{
				name: "missing passkey store",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), nil, nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"},
				want: codes.Internal,
			},
			{
				name: "missing config",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyInitErr = errors.New("boom")
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"},
				want: codes.Internal,
			},
			{
				name: "missing parser",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = nil
					return svc
				}(),
				req:  &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"},
				want: codes.Internal,
			},
			{
				name: "missing user id",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.BeginPasskeyRegistrationRequest{},
				want: codes.InvalidArgument,
			},
		}

		for _, tt := range tests {
			_, err := tt.svc.BeginPasskeyRegistration(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.want)
		}
	})

	t.Run("finish registration", func(t *testing.T) {
		tests := []struct {
			name string
			svc  *AuthService
			req  *authv1.FinishPasskeyRegistrationRequest
			want codes.Code
		}{
			{
				name: "nil request",
				svc:  NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil),
				req:  nil,
				want: codes.InvalidArgument,
			},
			{
				name: "missing session id",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.FinishPasskeyRegistrationRequest{CredentialResponseJson: []byte("{}")},
				want: codes.InvalidArgument,
			},
			{
				name: "missing credential json",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.FinishPasskeyRegistrationRequest{SessionId: "session-1"},
				want: codes.InvalidArgument,
			},
		}

		for _, tt := range tests {
			_, err := tt.svc.FinishPasskeyRegistration(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.want)
		}
	})

	t.Run("begin login", func(t *testing.T) {
		tests := []struct {
			name string
			svc  *AuthService
			req  *authv1.BeginPasskeyLoginRequest
			want codes.Code
		}{
			{
				name: "nil request",
				svc:  NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil),
				req:  nil,
				want: codes.InvalidArgument,
			},
			{
				name: "missing user store",
				svc: func() *AuthService {
					svc := NewAuthService(nil, newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.BeginPasskeyLoginRequest{Username: "alpha"},
				want: codes.Internal,
			},
			{
				name: "missing passkey store",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), nil, nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.BeginPasskeyLoginRequest{Username: "alpha"},
				want: codes.Internal,
			},
			{
				name: "missing config",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyInitErr = errors.New("boom")
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.BeginPasskeyLoginRequest{Username: "alpha"},
				want: codes.Internal,
			},
			{
				name: "missing parser",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = nil
					return svc
				}(),
				req:  &authv1.BeginPasskeyLoginRequest{Username: "alpha"},
				want: codes.Internal,
			},
		}

		for _, tt := range tests {
			_, err := tt.svc.BeginPasskeyLogin(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.want)
		}
	})

	t.Run("finish login", func(t *testing.T) {
		tests := []struct {
			name string
			svc  *AuthService
			req  *authv1.FinishPasskeyLoginRequest
			want codes.Code
		}{
			{
				name: "nil request",
				svc:  NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil),
				req:  nil,
				want: codes.InvalidArgument,
			},
			{
				name: "missing session id",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.FinishPasskeyLoginRequest{CredentialResponseJson: []byte("{}")},
				want: codes.InvalidArgument,
			},
			{
				name: "missing credential json",
				svc: func() *AuthService {
					svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
					svc.passkeyWebAuthn = &fakePasskeyProvider{}
					svc.passkeyInitErr = nil
					svc.passkeyParser = &fakePasskeyParser{}
					return svc
				}(),
				req:  &authv1.FinishPasskeyLoginRequest{SessionId: "session-2"},
				want: codes.InvalidArgument,
			},
		}

		for _, tt := range tests {
			_, err := tt.svc.FinishPasskeyLogin(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.want)
		}
	})
}

func TestBeginPasskeyRegistration_SessionCreationAndStorageFailures(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	t.Run("session id generator failure", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		svc := NewAuthService(userStore, passkeyStore, nil)
		svc.passkeyIDGenerator = func() (string, error) { return "", errors.New("id fail") }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{}

		_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
		grpcassert.StatusCode(t, err, codes.Internal)
		if len(passkeyStore.sessions) != 0 {
			t.Fatalf("expected no stored sessions, got %+v", passkeyStore.sessions)
		}
	})

	t.Run("store session failure", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.putErr = errors.New("put fail")
		svc := NewAuthService(userStore, passkeyStore, nil)
		svc.passkeyIDGenerator = func() (string, error) { return "session-1", nil }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{}

		_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
		grpcassert.StatusCode(t, err, codes.Internal)
		if _, ok := passkeyStore.sessions["session-1"]; ok {
			t.Fatal("expected session storage failure to avoid persisted session")
		}
	})
}

func TestBeginPasskeyLogin_RequiresUsername(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{}

	_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestBeginPasskeyLogin_Success(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()
	credential := webauthn.Credential{ID: []byte("cred-1")}
	payload, err := json.Marshal(credential)
	if err != nil {
		t.Fatalf("marshal credential: %v", err)
	}
	passkeyStore.credentials[encodeCredentialID(credential.ID)] = storage.PasskeyCredential{
		CredentialID:   encodeCredentialID(credential.ID),
		UserID:         "user-1",
		CredentialJSON: string(payload),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyIDGenerator = func() (string, error) { return "session-2", nil }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{}

	resp, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{Username: "alpha"})
	if err != nil {
		t.Fatalf("begin passkey login: %v", err)
	}
	if got := resp.GetSessionId(); got != "session-2" {
		t.Fatalf("session id = %q, want %q", got, "session-2")
	}
	if stored := passkeyStore.sessions["session-2"]; stored.UserID != "user-1" || stored.Kind != string(passkey.SessionKindLogin) {
		t.Fatalf("stored session = %+v", stored)
	}
}

func TestBeginPasskeyLogin_LoadPasskeyUserDecodeFailure(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.credentials["cred-1"] = storage.PasskeyCredential{
		CredentialID:   "cred-1",
		UserID:         "user-1",
		CredentialJSON: "{",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyIDGenerator = func() (string, error) { return "session-2", nil }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{}

	_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{Username: "alpha"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestBeginPasskeyLogin_BeginLoginFailure(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyWebAuthn = &fakePasskeyProvider{beginLoginErr: errors.New("login unavailable")}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{}

	_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{Username: "alpha"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestBeginPasskeyLogin_SessionCreationAndStorageFailures(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	t.Run("session id generator failure", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		svc := NewAuthService(userStore, passkeyStore, nil)
		svc.passkeyIDGenerator = func() (string, error) { return "", errors.New("id fail") }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{}

		_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{Username: "alpha"})
		grpcassert.StatusCode(t, err, codes.Internal)
		if len(passkeyStore.sessions) != 0 {
			t.Fatalf("expected no stored sessions, got %+v", passkeyStore.sessions)
		}
	})

	t.Run("store session failure", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.putErr = errors.New("put fail")
		svc := NewAuthService(userStore, passkeyStore, nil)
		svc.passkeyIDGenerator = func() (string, error) { return "session-2", nil }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{}

		_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{Username: "alpha"})
		grpcassert.StatusCode(t, err, codes.Internal)
		if _, ok := passkeyStore.sessions["session-2"]; ok {
			t.Fatal("expected session storage failure to avoid persisted session")
		}
	})
}

func TestFinishPasskeyRegistration_Success(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        string(passkey.SessionKindRegistration),
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &webauthn.Credential{ID: []byte("cred-reg")}}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	resp, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("finish passkey registration: %v", err)
	}
	if resp.GetUser().GetId() != "user-1" || resp.GetCredentialId() != encodeCredentialID([]byte("cred-reg")) {
		t.Fatalf("response = %#v", resp)
	}
	if _, ok := passkeyStore.sessions["session-1"]; ok {
		t.Fatal("expected session deleted")
	}
	stored, ok := passkeyStore.credentials[resp.GetCredentialId()]
	if !ok || stored.UserID != "user-1" {
		t.Fatalf("stored credential = %+v", stored)
	}
}

func TestFinishPasskeyRegistration_ParseErrorPreservesSession(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        string(passkey.SessionKindRegistration),
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{creationErr: errors.New("bad creation response")}

	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("{}"),
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
	if _, ok := passkeyStore.sessions["session-1"]; !ok {
		t.Fatal("expected session retained on parse failure")
	}
}

func TestFinishPasskeyRegistration_CreateCredentialFailurePreservesSession(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        string(passkey.SessionKindRegistration),
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{createCredentialErr: errors.New("credential rejected")}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("{}"),
	})
	grpcassert.StatusCode(t, err, codes.Internal)
	if _, ok := passkeyStore.sessions["session-1"]; !ok {
		t.Fatal("expected session retained on credential validation failure")
	}
}

func TestFinishPasskeyRegistration_StoreCredentialFailurePreservesSession(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.putErr = errors.New("put fail")
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        string(passkey.SessionKindRegistration),
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &webauthn.Credential{ID: []byte("cred-reg")}}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("{}"),
	})
	grpcassert.StatusCode(t, err, codes.Internal)
	if _, ok := passkeyStore.sessions["session-1"]; !ok {
		t.Fatal("expected session retained on credential storage failure")
	}
}

func TestFinishPasskeyRegistration_MissingSessionUserID(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        string(passkey.SessionKindRegistration),
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("{}"),
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestFinishPasskeyRegistration_LoadSessionPropagation(t *testing.T) {
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	t.Run("session not found", func(t *testing.T) {
		svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
		svc.clock = func() time.Time { return now }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

		_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
			SessionId:              "missing",
			CredentialResponseJson: []byte("{}"),
		})
		grpcassert.StatusCode(t, err, codes.NotFound)
	})

	t.Run("session kind mismatch", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.sessions["session-1"] = storage.PasskeySession{
			ID:          "session-1",
			Kind:        string(passkey.SessionKindLogin),
			UserID:      "user-1",
			SessionJSON: "{}",
			ExpiresAt:   now.Add(5 * time.Minute),
		}

		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

		_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
			SessionId:              "session-1",
			CredentialResponseJson: []byte("{}"),
		})
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
		if _, ok := passkeyStore.sessions["session-1"]; !ok {
			t.Fatal("expected mismatched session to remain")
		}
	})

	t.Run("session expired deletes reservation", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.sessions["session-1"] = storage.PasskeySession{
			ID:          "session-1",
			Kind:        string(passkey.SessionKindRegistration),
			UserID:      "user-1",
			SessionJSON: "{}",
			ExpiresAt:   now.Add(-time.Minute),
		}

		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

		_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
			SessionId:              "session-1",
			CredentialResponseJson: []byte("{}"),
		})
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
		if _, ok := passkeyStore.sessions["session-1"]; ok {
			t.Fatal("expected expired session deleted")
		}
	})
}

func TestFinishPasskeyLogin_Success(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	credential := webauthn.Credential{ID: []byte("cred-login")}
	payload, err := json.Marshal(credential)
	if err != nil {
		t.Fatalf("marshal credential: %v", err)
	}
	passkeyStore.credentials[encodeCredentialID(credential.ID)] = storage.PasskeyCredential{
		CredentialID:   encodeCredentialID(credential.ID),
		UserID:         "user-1",
		CredentialJSON: string(payload),
		CreatedAt:      now.Add(-time.Hour),
		UpdatedAt:      now.Add(-time.Hour),
	}
	passkeyStore.sessions["session-2"] = storage.PasskeySession{
		ID:          "session-2",
		Kind:        string(passkey.SessionKindLogin),
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &credential}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}

	resp, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              "session-2",
		CredentialResponseJson: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("finish passkey login: %v", err)
	}
	if resp.GetUser().GetId() != "user-1" || resp.GetCredentialId() != encodeCredentialID(credential.ID) {
		t.Fatalf("response = %#v", resp)
	}
	if _, ok := passkeyStore.sessions["session-2"]; ok {
		t.Fatal("expected session deleted")
	}
	stored := passkeyStore.credentials[resp.GetCredentialId()]
	if stored.LastUsedAt == nil || !stored.LastUsedAt.Equal(now) {
		t.Fatalf("stored credential = %+v", stored)
	}
}

func TestFinishPasskeyLogin_ParseErrorPreservesSession(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-2"] = storage.PasskeySession{
		ID:          "session-2",
		Kind:        string(passkey.SessionKindLogin),
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{assertionErr: errors.New("bad assertion response")}

	_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              "session-2",
		CredentialResponseJson: []byte("{}"),
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
	if _, ok := passkeyStore.sessions["session-2"]; !ok {
		t.Fatal("expected session retained on parse failure")
	}
}

func TestFinishPasskeyLogin_ValidateFailurePreservesSession(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-2"] = storage.PasskeySession{
		ID:          "session-2",
		Kind:        string(passkey.SessionKindLogin),
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{validateLoginErr: errors.New("assertion rejected")}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}

	_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              "session-2",
		CredentialResponseJson: []byte("{}"),
	})
	grpcassert.StatusCode(t, err, codes.Internal)
	if _, ok := passkeyStore.sessions["session-2"]; !ok {
		t.Fatal("expected session retained on validation failure")
	}
}

func TestFinishPasskeyLogin_StoreCredentialFailurePreservesSession(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	credential := webauthn.Credential{ID: []byte("cred-login")}
	payload, err := json.Marshal(credential)
	if err != nil {
		t.Fatalf("marshal credential: %v", err)
	}
	passkeyStore.credentials[encodeCredentialID(credential.ID)] = storage.PasskeyCredential{
		CredentialID:   encodeCredentialID(credential.ID),
		UserID:         "user-1",
		CredentialJSON: string(payload),
		CreatedAt:      now.Add(-time.Hour),
		UpdatedAt:      now.Add(-time.Hour),
	}
	passkeyStore.sessions["session-2"] = storage.PasskeySession{
		ID:          "session-2",
		Kind:        string(passkey.SessionKindLogin),
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}
	passkeyStore.putErr = errors.New("put fail")

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &credential}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}

	_, err = svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              "session-2",
		CredentialResponseJson: []byte("{}"),
	})
	grpcassert.StatusCode(t, err, codes.Internal)
	if _, ok := passkeyStore.sessions["session-2"]; !ok {
		t.Fatal("expected session retained on credential storage failure")
	}
}

func TestFinishPasskeyLogin_MissingStoredCredentialPreservesSession(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-2"] = storage.PasskeySession{
		ID:          "session-2",
		Kind:        string(passkey.SessionKindLogin),
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &webauthn.Credential{ID: []byte("cred-missing")}}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}

	_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              "session-2",
		CredentialResponseJson: []byte("{}"),
	})
	grpcassert.StatusCode(t, err, codes.Internal)
	if _, ok := passkeyStore.sessions["session-2"]; !ok {
		t.Fatal("expected session retained when stored credential is missing")
	}
}

func TestFinishPasskeyLogin_MissingSessionUserID(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-2"] = storage.PasskeySession{
		ID:          "session-2",
		Kind:        string(passkey.SessionKindLogin),
		SessionJSON: "{}",
		ExpiresAt:   now.Add(5 * time.Minute),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return now }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}

	_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              "session-2",
		CredentialResponseJson: []byte("{}"),
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestFinishPasskeyLogin_LoadSessionPropagation(t *testing.T) {
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	t.Run("session not found", func(t *testing.T) {
		svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
		svc.clock = func() time.Time { return now }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}

		_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
			SessionId:              "missing",
			CredentialResponseJson: []byte("{}"),
		})
		grpcassert.StatusCode(t, err, codes.NotFound)
	})

	t.Run("session kind mismatch", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.sessions["session-2"] = storage.PasskeySession{
			ID:          "session-2",
			Kind:        string(passkey.SessionKindRegistration),
			UserID:      "user-1",
			SessionJSON: "{}",
			ExpiresAt:   now.Add(5 * time.Minute),
		}

		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}

		_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
			SessionId:              "session-2",
			CredentialResponseJson: []byte("{}"),
		})
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
		if _, ok := passkeyStore.sessions["session-2"]; !ok {
			t.Fatal("expected mismatched session to remain")
		}
	})

	t.Run("session expired deletes reservation", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.sessions["session-2"] = storage.PasskeySession{
			ID:          "session-2",
			Kind:        string(passkey.SessionKindLogin),
			UserID:      "user-1",
			SessionJSON: "{}",
			ExpiresAt:   now.Add(-time.Minute),
		}

		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }
		svc.passkeyWebAuthn = &fakePasskeyProvider{}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}

		_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
			SessionId:              "session-2",
			CredentialResponseJson: []byte("{}"),
		})
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
		if _, ok := passkeyStore.sessions["session-2"]; ok {
			t.Fatal("expected expired session deleted")
		}
	})
}

func TestFinishPasskeyLogin_PendingAuthorizationPaths(t *testing.T) {
	now := time.Now().UTC()

	newService := func(t *testing.T, oauthStore *oauth.Store) (*AuthService, *fakePasskeyStore, webauthn.Credential) {
		t.Helper()
		userStore := newFakeUserStore()
		userStore.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
		passkeyStore := newFakePasskeyStore()
		credential := webauthn.Credential{ID: []byte("cred-login")}
		payload, err := json.Marshal(credential)
		if err != nil {
			t.Fatalf("marshal credential: %v", err)
		}
		passkeyStore.credentials[encodeCredentialID(credential.ID)] = storage.PasskeyCredential{
			CredentialID:   encodeCredentialID(credential.ID),
			UserID:         "user-1",
			CredentialJSON: string(payload),
			CreatedAt:      now.Add(-time.Hour),
			UpdatedAt:      now.Add(-time.Hour),
		}
		passkeyStore.sessions["session-2"] = storage.PasskeySession{
			ID:          "session-2",
			Kind:        string(passkey.SessionKindLogin),
			UserID:      "user-1",
			SessionJSON: "{}",
			ExpiresAt:   now.Add(5 * time.Minute),
		}

		svc := NewAuthService(userStore, passkeyStore, oauthStore)
		svc.clock = func() time.Time { return now }
		svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &credential}
		svc.passkeyInitErr = nil
		svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}
		return svc, passkeyStore, credential
	}

	t.Run("success updates pending authorization user", func(t *testing.T) {
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

		svc, passkeyStore, credential := newService(t, oauthStore)
		resp, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
			SessionId:              "session-2",
			CredentialResponseJson: []byte("{}"),
			PendingId:              pendingID,
		})
		if err != nil {
			t.Fatalf("finish passkey login with pending auth: %v", err)
		}
		if resp.GetCredentialId() != encodeCredentialID(credential.ID) {
			t.Fatalf("credential id = %q, want %q", resp.GetCredentialId(), encodeCredentialID(credential.ID))
		}
		pending, err := oauthStore.GetPendingAuthorization(pendingID)
		if err != nil {
			t.Fatalf("get pending authorization: %v", err)
		}
		if pending == nil || pending.UserID != "user-1" {
			t.Fatalf("pending authorization = %#v", pending)
		}
		if _, ok := passkeyStore.sessions["session-2"]; ok {
			t.Fatal("expected session deleted after successful pending auth attach")
		}
	})

	t.Run("expired pending authorization returns invalid argument after login success work", func(t *testing.T) {
		oauthStore := openTempOAuthStore(t)
		pendingID, err := oauthStore.CreatePendingAuthorization(oauth.AuthorizationRequest{
			ResponseType:        "code",
			ClientID:            "test-client",
			RedirectURI:         "http://localhost:5555/callback",
			CodeChallenge:       "challenge",
			CodeChallengeMethod: "S256",
		}, -time.Minute)
		if err != nil {
			t.Fatalf("create pending authorization: %v", err)
		}

		svc, passkeyStore, _ := newService(t, oauthStore)
		_, err = svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
			SessionId:              "session-2",
			CredentialResponseJson: []byte("{}"),
			PendingId:              pendingID,
		})
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
		pending, err := oauthStore.GetPendingAuthorization(pendingID)
		if err != nil {
			t.Fatalf("get pending authorization after expiry cleanup: %v", err)
		}
		if pending != nil {
			t.Fatalf("expected expired pending authorization deleted, got %#v", pending)
		}
		if _, ok := passkeyStore.sessions["session-2"]; ok {
			t.Fatal("expected passkey session deleted before pending auth attach failure")
		}
	})

	t.Run("missing oauth store returns internal after login success work", func(t *testing.T) {
		svc, passkeyStore, _ := newService(t, nil)
		_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
			SessionId:              "session-2",
			CredentialResponseJson: []byte("{}"),
			PendingId:              "pending-1",
		})
		grpcassert.StatusCode(t, err, codes.Internal)
		if _, ok := passkeyStore.sessions["session-2"]; ok {
			t.Fatal("expected passkey session deleted before pending auth attach failure")
		}
	})

	t.Run("missing pending authorization returns not found after login success work", func(t *testing.T) {
		oauthStore := openTempOAuthStore(t)
		svc, passkeyStore, _ := newService(t, oauthStore)
		_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
			SessionId:              "session-2",
			CredentialResponseJson: []byte("{}"),
			PendingId:              "missing",
		})
		grpcassert.StatusCode(t, err, codes.NotFound)
		if _, ok := passkeyStore.sessions["session-2"]; ok {
			t.Fatal("expected passkey session deleted before pending auth attach failure")
		}
	})
}

func TestPasskeyUserWebAuthnMethods(t *testing.T) {
	u := &passkeyUser{
		user:        user.User{ID: "user-1", Username: "alpha"},
		credentials: []webauthn.Credential{{ID: []byte("cred-1")}},
	}
	if got := string(u.WebAuthnID()); got != "user-1" {
		t.Fatalf("WebAuthnID() = %q", got)
	}
	if got := u.WebAuthnName(); got != "alpha" {
		t.Fatalf("WebAuthnName() = %q", got)
	}
	if got := u.WebAuthnDisplayName(); got != "alpha" {
		t.Fatalf("WebAuthnDisplayName() = %q", got)
	}
	if got := u.WebAuthnIcon(); got != "" {
		t.Fatalf("WebAuthnIcon() = %q", got)
	}
	if got := u.WebAuthnCredentials(); len(got) != 1 || !bytes.Equal(got[0].ID, []byte("cred-1")) {
		t.Fatalf("WebAuthnCredentials() = %#v", got)
	}
}

func TestDecodeStoredCredentials_InvalidJSON(t *testing.T) {
	_, err := decodeStoredCredentials([]storage.PasskeyCredential{{
		CredentialID:   "cred-1",
		CredentialJSON: "{",
	}})
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestBuildPasskeyCredentialRecord_UsedCredentialMissing(t *testing.T) {
	passkeyStore := newFakePasskeyStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
	svc.clock = func() time.Time { return now }

	_, err := svc.buildPasskeyCredentialRecord(context.Background(), "user-1", webauthn.Credential{ID: []byte("cred-1")}, true)
	if err == nil {
		t.Fatal("expected missing used credential error")
	}
}

func TestBuildPasskeyCredentialRecord_PreservesCreatedAtAndSetsLastUsed(t *testing.T) {
	passkeyStore := newFakePasskeyStore()
	createdAt := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
	now := createdAt.Add(2 * time.Hour)
	credentialID := encodeCredentialID([]byte("cred-1"))
	passkeyStore.credentials[credentialID] = storage.PasskeyCredential{
		CredentialID:   credentialID,
		UserID:         "user-1",
		CredentialJSON: "{}",
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}

	svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
	svc.clock = func() time.Time { return now }

	record, err := svc.buildPasskeyCredentialRecord(context.Background(), "user-1", webauthn.Credential{ID: []byte("cred-1")}, true)
	if err != nil {
		t.Fatalf("build passkey credential record: %v", err)
	}
	if !record.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %v, want %v", record.CreatedAt, createdAt)
	}
	if !record.UpdatedAt.Equal(now) {
		t.Fatalf("updated_at = %v, want %v", record.UpdatedAt, now)
	}
	if record.LastUsedAt == nil || !record.LastUsedAt.Equal(now) {
		t.Fatalf("last_used_at = %v, want %v", record.LastUsedAt, now)
	}
}

func TestLoadPasskeySession_ErrorPaths(t *testing.T) {
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	t.Run("kind mismatch", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.sessions["session-1"] = storage.PasskeySession{
			ID:          "session-1",
			Kind:        string(passkey.SessionKindLogin),
			UserID:      "user-1",
			SessionJSON: "{}",
			ExpiresAt:   now.Add(5 * time.Minute),
		}

		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }

		_, err := svc.loadPasskeySession(context.Background(), "session-1", passkey.SessionKindRegistration)
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
	})

	t.Run("expired deletes session", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.sessions["session-1"] = storage.PasskeySession{
			ID:          "session-1",
			Kind:        string(passkey.SessionKindLogin),
			UserID:      "user-1",
			SessionJSON: "{}",
			ExpiresAt:   now.Add(-time.Minute),
		}

		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }

		_, err := svc.loadPasskeySession(context.Background(), "session-1", passkey.SessionKindLogin)
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
		if _, ok := passkeyStore.sessions["session-1"]; ok {
			t.Fatal("expected expired session deleted")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.sessions["session-1"] = storage.PasskeySession{
			ID:          "session-1",
			Kind:        string(passkey.SessionKindLogin),
			UserID:      "user-1",
			SessionJSON: "{",
			ExpiresAt:   now.Add(5 * time.Minute),
		}

		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return now }

		_, err := svc.loadPasskeySession(context.Background(), "session-1", passkey.SessionKindLogin)
		grpcassert.StatusCode(t, err, codes.Internal)
	})
}

func TestLoadPasskeyUserAndStorageHelpers(t *testing.T) {
	t.Run("load passkey user list error", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.listErr = errors.New("list fail")
		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)

		_, err := svc.loadPasskeyUser(context.Background(), user.User{ID: "user-1", Username: "alpha"})
		if err == nil || err.Error() != "list fail" {
			t.Fatalf("loadPasskeyUser() error = %v, want list fail", err)
		}
	})

	t.Run("store passkey credential put error", func(t *testing.T) {
		passkeyStore := newFakePasskeyStore()
		passkeyStore.putErr = errors.New("put fail")
		svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
		svc.clock = func() time.Time { return time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC) }

		err := svc.storePasskeyCredential(context.Background(), "user-1", webauthn.Credential{ID: []byte("cred-1")}, false)
		if err == nil || err.Error() != "put fail" {
			t.Fatalf("storePasskeyCredential() error = %v, want put fail", err)
		}
	})

	t.Run("store passkey session requires session data", func(t *testing.T) {
		svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
		err := svc.storePasskeySessionWithTTL(context.Background(), "session-1", passkey.SessionKindLogin, "user-1", nil, time.Minute)
		if err == nil || err.Error() != "Session data is required." {
			t.Fatalf("storePasskeySessionWithTTL() error = %v", err)
		}
	})

	t.Run("new passkey session id falls back to generator", func(t *testing.T) {
		svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
		id, err := svc.newPasskeySessionID()
		if err != nil {
			t.Fatalf("newPasskeySessionID() error = %v", err)
		}
		if id == "" {
			t.Fatal("newPasskeySessionID() = empty, want non-empty")
		}
	})
}

func TestDefaultPasskeyParserRejectsInvalidPayloads(t *testing.T) {
	parser := defaultPasskeyParser{}

	if _, err := parser.ParseCredentialCreationResponseBytes([]byte("not-json")); err == nil {
		t.Fatal("expected creation parse error")
	}
	if _, err := parser.ParseCredentialRequestResponseBytes([]byte("not-json")); err == nil {
		t.Fatal("expected request parse error")
	}
}

func TestBeginAccountRegistration_Success(t *testing.T) {
	t.Parallel()

	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }
	svc.passkeyConfig.SessionTTL = 5 * time.Minute
	svc.passkeyConfig.SignupSessionTTL = 2 * time.Minute
	svc.idGenerator = func() (string, error) { return "user-123", nil }
	svc.passkeyIDGenerator = func() (string, error) { return "reg-1", nil }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil

	resp, err := svc.BeginAccountRegistration(context.Background(), &authv1.BeginAccountRegistrationRequest{Username: "  ALICE  "})
	if err != nil {
		t.Fatalf("begin account registration: %v", err)
	}
	if got := resp.GetSessionId(); got != "reg-1" {
		t.Fatalf("session id = %q, want %q", got, "reg-1")
	}
	registration := passkeyStore.registrations["reg-1"]
	if registration.UserID != "user-123" {
		t.Fatalf("registration user id = %q, want %q", registration.UserID, "user-123")
	}
	if registration.Username != "alice" {
		t.Fatalf("registration username = %q, want %q", registration.Username, "alice")
	}
	if got := registration.ExpiresAt; !got.Equal(now.Add(2 * time.Minute)) {
		t.Fatalf("registration ExpiresAt = %v, want %v", got, now.Add(2*time.Minute))
	}
	if stored := passkeyStore.sessions["reg-1"]; !stored.ExpiresAt.Equal(now.Add(2 * time.Minute)) {
		t.Fatalf("passkey session ExpiresAt = %v, want %v", stored.ExpiresAt, now.Add(2*time.Minute))
	}
}

func TestBeginAccountRegistration_DeletesExpiredReservationAndSucceeds(t *testing.T) {
	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	if err := store.PutPasskeySession(context.Background(), storage.PasskeySession{
		ID:          "expired-reg",
		Kind:        string(passkey.SessionKindRegistration),
		UserID:      "user-old",
		SessionJSON: "{}",
		ExpiresAt:   now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("put expired passkey session: %v", err)
	}
	if err := store.PutRegistrationSession(context.Background(), storage.RegistrationSession{
		ID:        "expired-reg",
		UserID:    "user-old",
		Username:  "alice",
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now.Add(-2 * time.Minute),
		UpdatedAt: now.Add(-2 * time.Minute),
	}); err != nil {
		t.Fatalf("put expired registration: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return now }
	svc.idGenerator = func() (string, error) { return "user-123", nil }
	svc.passkeyIDGenerator = func() (string, error) { return "reg-1", nil }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil

	resp, err := svc.BeginAccountRegistration(context.Background(), &authv1.BeginAccountRegistrationRequest{Username: "alice"})
	if err != nil {
		t.Fatalf("begin account registration: %v", err)
	}
	if got := resp.GetSessionId(); got != "reg-1" {
		t.Fatalf("session id = %q, want %q", got, "reg-1")
	}
	if _, err := store.GetRegistrationSession(context.Background(), "expired-reg"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected expired reservation cleanup, got %v", err)
	}
	if _, err := store.GetPasskeySession(context.Background(), "expired-reg"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected expired passkey session cleanup, got %v", err)
	}
}

func TestBeginAccountRegistration_ReservationFailureDoesNotLeakPasskeySession(t *testing.T) {
	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	if err := store.PutRegistrationSession(context.Background(), storage.RegistrationSession{
		ID:        "active-reg",
		UserID:    "user-old",
		Username:  "alice",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("put active registration: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return now }
	svc.idGenerator = func() (string, error) { return "user-123", nil }
	svc.passkeyIDGenerator = func() (string, error) { return "reg-1", nil }
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil

	_, err := svc.BeginAccountRegistration(context.Background(), &authv1.BeginAccountRegistrationRequest{Username: "alice"})
	grpcassert.StatusCode(t, err, codes.AlreadyExists)
	if _, err := store.GetPasskeySession(context.Background(), "reg-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no leaked passkey session, got %v", err)
	}
}

func TestFinishAccountRegistration_StagesSignupUntilAcknowledged(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	svc := NewAuthService(store, store, nil)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	ids := []string{"user-1"}
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

	finishResp, err := svc.FinishAccountRegistration(context.Background(), &authv1.FinishAccountRegistrationRequest{
		SessionId:              beginResp.GetSessionId(),
		CredentialResponseJson: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("finish account registration: %v", err)
	}
	if finishResp.GetRecoveryCode() == "" {
		t.Fatal("expected recovery code")
	}
	if got := finishResp.GetSession().GetId(); got != "" {
		t.Fatalf("web session id = %q, want empty", got)
	}
	if got := finishResp.GetUser().GetId(); got != "" {
		t.Fatalf("user id = %q, want empty", got)
	}

	if _, err := store.GetUser(context.Background(), "user-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected staged signup to avoid user creation, got %v", err)
	}
	if _, err := store.GetPasskeyCredential(context.Background(), encodeCredentialID([]byte("cred-1"))); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no stored passkey before acknowledge, got %v", err)
	}
	registration, err := store.GetRegistrationSession(context.Background(), beginResp.GetSessionId())
	if err != nil {
		t.Fatalf("get staged registration: %v", err)
	}
	if registration.CredentialID != encodeCredentialID([]byte("cred-1")) {
		t.Fatalf("credential id = %q, want %q", registration.CredentialID, encodeCredentialID([]byte("cred-1")))
	}
	if registration.CredentialJSON == "" {
		t.Fatal("expected staged credential json")
	}
	if !recoverycode.Verify(finishResp.GetRecoveryCode(), registration.RecoveryCodeHash) {
		t.Fatal("expected staged recovery hash to verify")
	}
	if _, err := store.GetPasskeySession(context.Background(), beginResp.GetSessionId()); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected passkey session cleanup after staging, got %v", err)
	}
	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 10, now, time.Minute)
	if err != nil {
		t.Fatalf("lease signup outbox events: %v", err)
	}
	if len(leased) != 0 {
		t.Fatalf("leased events len = %d, want 0", len(leased))
	}
}

func TestAcknowledgeAccountRegistration_Success(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	svc := NewAuthService(store, store, nil)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
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
	finishResp, err := svc.FinishAccountRegistration(context.Background(), &authv1.FinishAccountRegistrationRequest{
		SessionId:              beginResp.GetSessionId(),
		CredentialResponseJson: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("finish account registration: %v", err)
	}

	ackResp, err := svc.AcknowledgeAccountRegistration(context.Background(), &authv1.AcknowledgeAccountRegistrationRequest{
		SessionId: beginResp.GetSessionId(),
	})
	if err != nil {
		t.Fatalf("acknowledge account registration: %v", err)
	}
	if got := ackResp.GetUser().GetUsername(); got != "alpha" {
		t.Fatalf("username = %q, want %q", got, "alpha")
	}
	if ackResp.GetSession().GetId() != "ws-1" {
		t.Fatalf("web session id = %q, want %q", ackResp.GetSession().GetId(), "ws-1")
	}

	storedUser, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if !recoverycode.Verify(finishResp.GetRecoveryCode(), storedUser.RecoveryCodeHash) {
		t.Fatal("expected stored recovery hash to verify")
	}
	if _, err := store.GetPasskeyCredential(context.Background(), encodeCredentialID([]byte("cred-1"))); err != nil {
		t.Fatalf("get stored passkey: %v", err)
	}
	if _, err := store.GetRegistrationSession(context.Background(), beginResp.GetSessionId()); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected registration cleanup after acknowledge, got %v", err)
	}
	leased, err := store.LeaseIntegrationOutboxEvents(context.Background(), "worker-1", 10, now, time.Minute)
	if err != nil {
		t.Fatalf("lease signup outbox events: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased events len = %d, want 1", len(leased))
	}
	if leased[0].EventType != "auth.signup_completed" {
		t.Fatalf("event type = %q, want %q", leased[0].EventType, "auth.signup_completed")
	}
}

func TestAcknowledgeAccountRegistration_PersistFailureLeavesRegistrationState(t *testing.T) {
	t.Parallel()

	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
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
	svc := NewAuthService(userStore, passkeyStore, oauthStore)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	userStore.signupPutErr = errors.New("boom")
	ids := []string{"user-1", "ws-1", "event-1"}
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

	_, err = svc.FinishAccountRegistration(context.Background(), &authv1.FinishAccountRegistrationRequest{
		SessionId:              beginResp.GetSessionId(),
		CredentialResponseJson: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("finish account registration: %v", err)
	}

	_, err = svc.AcknowledgeAccountRegistration(context.Background(), &authv1.AcknowledgeAccountRegistrationRequest{
		SessionId: beginResp.GetSessionId(),
		PendingId: pendingID,
	})
	grpcassert.StatusCode(t, err, codes.Internal)
	if _, ok := userStore.users["user-1"]; ok {
		t.Fatal("expected user persistence rollback")
	}
	if _, ok := passkeyStore.registrations["reg-1"]; !ok {
		t.Fatal("expected staged registration session to remain for retry")
	}
	if _, ok := passkeyStore.sessions["reg-1"]; ok {
		t.Fatal("expected passkey session cleanup after staging")
	}
	pending, err := oauthStore.GetPendingAuthorization(pendingID)
	if err != nil {
		t.Fatalf("get pending authorization: %v", err)
	}
	if pending == nil {
		t.Fatal("expected pending authorization to remain")
	}
	if pending.UserID != "" {
		t.Fatalf("pending authorization user id = %q, want empty", pending.UserID)
	}
}

func TestAcknowledgeAccountRegistration_ExpiredSessionDeletesReservation(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	if err := store.PutRegistrationSession(context.Background(), storage.RegistrationSession{
		ID:               "reg-1",
		UserID:           "user-1",
		Username:         "alpha",
		Locale:           "en-US",
		RecoveryCodeHash: "hash-1",
		CredentialID:     "cred-1",
		CredentialJSON:   `{"id":"cred-1"}`,
		ExpiresAt:        now.Add(-time.Minute),
		CreatedAt:        now.Add(-2 * time.Minute),
		UpdatedAt:        now.Add(-2 * time.Minute),
	}); err != nil {
		t.Fatalf("put registration session: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return now }

	_, err := svc.AcknowledgeAccountRegistration(context.Background(), &authv1.AcknowledgeAccountRegistrationRequest{
		SessionId: "reg-1",
	})
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
	if _, err := store.GetRegistrationSession(context.Background(), "reg-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected expired staged signup cleanup, got %v", err)
	}
}

func TestBeginAccountRecovery_Success(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	code, hash, err := recoverycode.Generate(bytes.NewReader(bytes.Repeat([]byte{2}, 64)))
	if err != nil {
		t.Fatalf("generate recovery code: %v", err)
	}
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	if err := store.PutUser(context.Background(), user.User{
		ID:               "user-1",
		Username:         "alpha",
		RecoveryCodeHash: hash,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("put user: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return now }
	svc.idGenerator = func() (string, error) { return "recovery-1", nil }

	resp, err := svc.BeginAccountRecovery(context.Background(), &authv1.BeginAccountRecoveryRequest{
		Username:     "alpha",
		RecoveryCode: code,
	})
	if err != nil {
		t.Fatalf("begin account recovery: %v", err)
	}
	if got := resp.GetRecoverySessionId(); got != "recovery-1" {
		t.Fatalf("recovery session id = %q, want %q", got, "recovery-1")
	}

	storedUser, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if storedUser.RecoveryReservedSessionID != "recovery-1" {
		t.Fatalf("reserved recovery session = %q, want %q", storedUser.RecoveryReservedSessionID, "recovery-1")
	}
}

func TestFinishRecoveryPasskeyRegistration_Success(t *testing.T) {
	t.Parallel()

	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	oldCode, oldHash, err := recoverycode.Generate(bytes.NewReader(bytes.Repeat([]byte{3}, 64)))
	if err != nil {
		t.Fatalf("generate recovery code: %v", err)
	}
	userRecord := user.User{
		ID:               "user-1",
		Username:         "alpha",
		RecoveryCodeHash: oldHash,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.PutUser(context.Background(), userRecord); err != nil {
		t.Fatalf("put user: %v", err)
	}
	if err := store.PutWebSession(context.Background(), storage.WebSession{
		ID:        "ws-old",
		UserID:    "user-1",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("put web session: %v", err)
	}
	oldCredential := webauthn.Credential{ID: []byte("old-cred")}
	oldPayload, err := json.Marshal(oldCredential)
	if err != nil {
		t.Fatalf("marshal old credential: %v", err)
	}
	if err := store.PutPasskeyCredential(context.Background(), storage.PasskeyCredential{
		CredentialID:   encodeCredentialID(oldCredential.ID),
		UserID:         "user-1",
		CredentialJSON: string(oldPayload),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("put old credential: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	ids := []string{"recovery-1", "ws-new"}
	svc.idGenerator = func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}
	svc.passkeyIDGenerator = func() (string, error) { return "passkey-1", nil }
	svc.clock = func() time.Time { return now }
	svc.randReader = bytes.NewReader(bytes.Repeat([]byte{4}, 64))
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &webauthn.Credential{ID: []byte("new-cred")}}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	beginRecovery, err := svc.BeginAccountRecovery(context.Background(), &authv1.BeginAccountRecoveryRequest{
		Username:     "alpha",
		RecoveryCode: oldCode,
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
	finishResp, err := svc.FinishRecoveryPasskeyRegistration(context.Background(), &authv1.FinishRecoveryPasskeyRegistrationRequest{
		RecoverySessionId:      beginRecovery.GetRecoverySessionId(),
		SessionId:              beginPasskey.GetSessionId(),
		CredentialResponseJson: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("finish recovery passkey registration: %v", err)
	}
	if finishResp.GetSession().GetId() != "ws-new" {
		t.Fatalf("new web session id = %q, want %q", finishResp.GetSession().GetId(), "ws-new")
	}
	if finishResp.GetRecoveryCode() == "" {
		t.Fatal("expected rotated recovery code")
	}

	storedUser, err := store.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if recoverycode.Verify(oldCode, storedUser.RecoveryCodeHash) {
		t.Fatal("expected old recovery code to be invalid after rotation")
	}
	if !recoverycode.Verify(finishResp.GetRecoveryCode(), storedUser.RecoveryCodeHash) {
		t.Fatal("expected new recovery code to verify")
	}
	if _, err := store.GetPasskeyCredential(context.Background(), encodeCredentialID([]byte("old-cred"))); err != storage.ErrNotFound {
		t.Fatalf("old credential err = %v, want not found", err)
	}
	newCredential, err := store.GetPasskeyCredential(context.Background(), encodeCredentialID([]byte("new-cred")))
	if err != nil {
		t.Fatalf("get new credential: %v", err)
	}
	if newCredential.UserID != "user-1" {
		t.Fatalf("new credential user id = %q, want %q", newCredential.UserID, "user-1")
	}
	webSession, err := store.GetWebSession(context.Background(), "ws-old")
	if err != nil {
		t.Fatalf("get old web session: %v", err)
	}
	if webSession.RevokedAt == nil {
		t.Fatal("expected old web session revoked")
	}
}

func TestAttachPendingAuthorizationExpired(t *testing.T) {
	oauthStore := openTempOAuthStore(t)
	pendingID, err := oauthStore.CreatePendingAuthorization(oauth.AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	}, -time.Minute)
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}

	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), oauthStore)
	if err := svc.attachPendingAuthorization(context.Background(), pendingID, "user-1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAttachPendingAuthorizationGuardBranches(t *testing.T) {
	t.Run("missing oauth store", func(t *testing.T) {
		svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
		err := svc.attachPendingAuthorization(context.Background(), "pending-1", "user-1")
		grpcassert.StatusCode(t, err, codes.Internal)
	})

	t.Run("missing ids", func(t *testing.T) {
		svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), openTempOAuthStore(t))
		err := svc.attachPendingAuthorization(context.Background(), " ", "user-1")
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), openTempOAuthStore(t))
		err := svc.attachPendingAuthorization(context.Background(), "missing", "user-1")
		grpcassert.StatusCode(t, err, codes.NotFound)
	})
}

func TestListPasskeys_Success(t *testing.T) {
	passkeyStore := newFakePasskeyStore()
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	lastUsed := now.Add(10 * time.Minute)
	passkeyStore.credentials["cred-1"] = storage.PasskeyCredential{
		CredentialID:   "cred-1",
		UserID:         "user-1",
		CredentialJSON: "{}",
		CreatedAt:      now,
		UpdatedAt:      now.Add(time.Minute),
		LastUsedAt:     &lastUsed,
	}
	passkeyStore.credentials["cred-2"] = storage.PasskeyCredential{
		CredentialID:   "cred-2",
		UserID:         "user-1",
		CredentialJSON: "{}",
		CreatedAt:      now.Add(2 * time.Minute),
		UpdatedAt:      now.Add(3 * time.Minute),
	}
	passkeyStore.credentials["cred-3"] = storage.PasskeyCredential{
		CredentialID:   "cred-3",
		UserID:         "user-2",
		CredentialJSON: "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	svc := NewAuthService(newFakeUserStore(), passkeyStore, nil)
	resp, err := svc.ListPasskeys(context.Background(), &authv1.ListPasskeysRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("list passkeys: %v", err)
	}
	if len(resp.GetPasskeys()) != 2 {
		t.Fatalf("passkeys = %#v", resp.GetPasskeys())
	}
	foundLastUsed := false
	for _, item := range resp.GetPasskeys() {
		if item.GetCredentialId() == "cred-1" {
			foundLastUsed = true
			if item.GetLastUsedAt() == nil || !item.GetLastUsedAt().AsTime().Equal(lastUsed) {
				t.Fatalf("last used = %v", item.GetLastUsedAt())
			}
		}
	}
	if !foundLastUsed {
		t.Fatalf("expected cred-1 in response: %#v", resp.GetPasskeys())
	}
}

type fakePasskeyProvider struct {
	credential           *webauthn.Credential
	beginRegistrationErr error
	createCredentialErr  error
	beginLoginErr        error
	validateLoginErr     error
}

func (f *fakePasskeyProvider) BeginRegistration(user webauthn.User, opts ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	if f.beginRegistrationErr != nil {
		return nil, nil, f.beginRegistrationErr
	}
	return &protocol.CredentialCreation{}, &webauthn.SessionData{}, nil
}

func (f *fakePasskeyProvider) CreateCredential(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
	if f.createCredentialErr != nil {
		return nil, f.createCredentialErr
	}
	if f.credential != nil {
		return f.credential, nil
	}
	return &webauthn.Credential{ID: []byte("cred")}, nil
}

func (f *fakePasskeyProvider) BeginLogin(user webauthn.User, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	if f.beginLoginErr != nil {
		return nil, nil, f.beginLoginErr
	}
	return &protocol.CredentialAssertion{}, &webauthn.SessionData{}, nil
}

func (f *fakePasskeyProvider) ValidateLogin(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error) {
	if f.validateLoginErr != nil {
		return nil, f.validateLoginErr
	}
	if f.credential != nil {
		return f.credential, nil
	}
	return &webauthn.Credential{ID: []byte("cred")}, nil
}

type fakePasskeyParser struct {
	creation     *protocol.ParsedCredentialCreationData
	creationErr  error
	assertion    *protocol.ParsedCredentialAssertionData
	assertionErr error
}

func (f *fakePasskeyParser) ParseCredentialCreationResponseBytes(_ []byte) (*protocol.ParsedCredentialCreationData, error) {
	if f.creationErr != nil {
		return nil, f.creationErr
	}
	if f.creation != nil {
		return f.creation, nil
	}
	return &protocol.ParsedCredentialCreationData{}, nil
}

func (f *fakePasskeyParser) ParseCredentialRequestResponseBytes(_ []byte) (*protocol.ParsedCredentialAssertionData, error) {
	if f.assertionErr != nil {
		return nil, f.assertionErr
	}
	if f.assertion != nil {
		return f.assertion, nil
	}
	return &protocol.ParsedCredentialAssertionData{}, nil
}

func openTempOAuthStore(t *testing.T) *oauth.Store {
	t.Helper()
	path := t.TempDir() + "/auth.db"
	authStore, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open auth store: %v", err)
	}
	t.Cleanup(func() { authStore.Close() })
	return oauth.NewStore(authStore.DB())
}
