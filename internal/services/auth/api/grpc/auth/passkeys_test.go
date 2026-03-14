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

func TestBeginPasskeyLogin_RequiresUsername(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyInitErr = nil
	svc.passkeyParser = &fakePasskeyParser{}

	_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
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

func TestBeginAccountRegistration_Success(t *testing.T) {
	t.Parallel()

	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)
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
}

func TestBeginAccountRegistration_DeletesExpiredReservationAndSucceeds(t *testing.T) {
	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
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
	assertStatusCode(t, err, codes.Internal)
	if _, err := store.GetPasskeySession(context.Background(), "reg-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no leaked passkey session, got %v", err)
	}
}

func TestFinishAccountRegistration_Success(t *testing.T) {
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
	if got := finishResp.GetUser().GetUsername(); got != "alpha" {
		t.Fatalf("username = %q, want %q", got, "alpha")
	}
	if finishResp.GetRecoveryCode() == "" {
		t.Fatal("expected recovery code")
	}
	if finishResp.GetSession().GetId() != "ws-1" {
		t.Fatalf("web session id = %q, want %q", finishResp.GetSession().GetId(), "ws-1")
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

func TestFinishAccountRegistration_PersistFailureLeavesRegistrationState(t *testing.T) {
	t.Parallel()

	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)
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
	assertStatusCode(t, err, codes.Internal)
	if _, ok := userStore.users["user-1"]; ok {
		t.Fatal("expected user persistence rollback")
	}
	if _, ok := passkeyStore.registrations["reg-1"]; !ok {
		t.Fatal("expected registration session to remain for retry")
	}
	if _, ok := passkeyStore.sessions["reg-1"]; !ok {
		t.Fatal("expected passkey session to remain for retry")
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

type fakePasskeyProvider struct {
	credential           *webauthn.Credential
	beginRegistrationErr error
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
	creation  *protocol.ParsedCredentialCreationData
	assertion *protocol.ParsedCredentialAssertionData
}

func (f *fakePasskeyParser) ParseCredentialCreationResponseBytes(_ []byte) (*protocol.ParsedCredentialCreationData, error) {
	if f.creation != nil {
		return f.creation, nil
	}
	return &protocol.ParsedCredentialCreationData{}, nil
}

func (f *fakePasskeyParser) ParseCredentialRequestResponseBytes(_ []byte) (*protocol.ParsedCredentialAssertionData, error) {
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
