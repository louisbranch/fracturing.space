package auth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	"github.com/louisbranch/fracturing.space/internal/services/auth/passkey"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakePasskeyStore struct {
	sessions    map[string]storage.PasskeySession
	credentials map[string]storage.PasskeyCredential
	putErr      error
	getErr      error
	listErr     error
}

func newFakePasskeyStore() *fakePasskeyStore {
	return &fakePasskeyStore{
		sessions:    make(map[string]storage.PasskeySession),
		credentials: make(map[string]storage.PasskeyCredential),
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

func TestBeginPasskeyRegistration_Success(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()

	svc := NewAuthService(userStore, passkeyStore, nil)
	fixed := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return fixed }
	svc.passkeyIDGenerator = func() (string, error) { return "session-1", nil }

	resp, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("begin passkey registration: %v", err)
	}
	if resp.GetSessionId() == "" {
		t.Fatalf("expected session id")
	}
	if len(resp.GetCredentialCreationOptionsJson()) == 0 {
		t.Fatalf("expected creation options json")
	}
	stored, ok := passkeyStore.sessions[resp.GetSessionId()]
	if !ok {
		t.Fatalf("expected stored session")
	}
	if stored.UserID != "user-1" {
		t.Fatalf("stored user id = %q, want %q", stored.UserID, "user-1")
	}
	if stored.Kind != "registration" {
		t.Fatalf("stored kind = %q, want %q", stored.Kind, "registration")
	}
	if !stored.ExpiresAt.After(fixed) {
		t.Fatalf("expected expiry after now")
	}
}

func TestBeginPasskeyRegistration_MissingUserID(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)

	_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestBeginPasskeyRegistration_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	_, err := svc.BeginPasskeyRegistration(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestBeginPasskeyRegistration_NilStore(t *testing.T) {
	svc := NewAuthService(nil, newFakePasskeyStore(), nil)
	_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestBeginPasskeyRegistration_NilPasskeyStore(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestBeginPasskeyRegistration_WebAuthnNil(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	svc.passkeyWebAuthn = nil
	_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestBeginPasskeyRegistration_UserNotFound(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)

	_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "missing"})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.NotFound)
}

func TestBeginPasskeyRegistration_StoreSessionError(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()
	passkeyStore.putErr = status.Error(codes.Internal, "store failed")

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyWebAuthn = &fakePasskeyProvider{}

	_, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestBeginPasskeyRegistration_WithExistingCredentials(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()

	credential := webauthn.Credential{ID: []byte("cred-1")}
	jsonPayload, err := json.Marshal(credential)
	if err != nil {
		t.Fatalf("marshal credential: %v", err)
	}
	passkeyStore.credentials[encodeCredentialID(credential.ID)] = storage.PasskeyCredential{
		CredentialID:   encodeCredentialID(credential.ID),
		UserID:         "user-1",
		CredentialJSON: string(jsonPayload),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyIDGenerator = func() (string, error) { return "session-3", nil }

	resp, err := svc.BeginPasskeyRegistration(context.Background(), &authv1.BeginPasskeyRegistrationRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("begin passkey registration: %v", err)
	}
	if resp.GetSessionId() == "" {
		t.Fatalf("expected session id")
	}
}

func TestBeginPasskeyLogin_Discoverable(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyIDGenerator = func() (string, error) { return "session-2", nil }

	resp, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{})
	if err != nil {
		t.Fatalf("begin passkey login: %v", err)
	}
	if resp.GetSessionId() == "" {
		t.Fatalf("expected session id")
	}
	if len(resp.GetCredentialRequestOptionsJson()) == 0 {
		t.Fatalf("expected request options json")
	}
	stored, ok := passkeyStore.sessions[resp.GetSessionId()]
	if !ok {
		t.Fatalf("expected stored session")
	}
	if stored.UserID != "" {
		t.Fatalf("stored user id = %q, want empty", stored.UserID)
	}
	if stored.Kind != "login" {
		t.Fatalf("stored kind = %q, want %q", stored.Kind, "login")
	}
}

func TestBeginPasskeyLogin_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	_, err := svc.BeginPasskeyLogin(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestBeginPasskeyLogin_NilStore(t *testing.T) {
	svc := NewAuthService(nil, newFakePasskeyStore(), nil)
	_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestBeginPasskeyLogin_NilPasskeyStore(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestBeginPasskeyLogin_WebAuthnNil(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	svc.passkeyWebAuthn = nil
	_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestBeginPasskeyLogin_UserNotFound(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)

	_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{UserId: "missing"})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.NotFound)
}

func TestBeginPasskeyLogin_StoreSessionError(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	passkeyStore.putErr = status.Error(codes.Internal, "store failed")

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyWebAuthn = &fakePasskeyProvider{}

	_, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestBeginPasskeyLogin_WithUserID(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()

	credential := webauthn.Credential{ID: []byte("cred-1")}
	jsonPayload, err := json.Marshal(credential)
	if err != nil {
		t.Fatalf("marshal credential: %v", err)
	}
	passkeyStore.credentials[encodeCredentialID(credential.ID)] = storage.PasskeyCredential{
		CredentialID:   encodeCredentialID(credential.ID),
		UserID:         "user-1",
		CredentialJSON: string(jsonPayload),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyIDGenerator = func() (string, error) { return "session-4", nil }

	resp, err := svc.BeginPasskeyLogin(context.Background(), &authv1.BeginPasskeyLoginRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("begin passkey login: %v", err)
	}
	if resp.GetSessionId() == "" {
		t.Fatalf("expected session id")
	}
}

func TestFinishPasskeyLogin_MissingSession(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)

	_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{SessionId: "missing", CredentialResponseJson: []byte("{}")})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.NotFound)
}

func TestFinishPasskeyLogin_MissingFields(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{SessionId: ""})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestFinishPasskeyLogin_InvalidCredentialJSON(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)

	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        "login",
		SessionJSON: "{}",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("not-json"),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestFinishPasskeyLogin_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	_, err := svc.FinishPasskeyLogin(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestFinishPasskeyLogin_NilPasskeyStore(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{SessionId: "session", CredentialResponseJson: []byte("{}")})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestFinishPasskeyRegistration_MissingSession(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)

	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{SessionId: "missing", CredentialResponseJson: []byte("{}")})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.NotFound)
}

func TestFinishPasskeyRegistration_MissingFields(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{SessionId: ""})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestFinishPasskeyRegistration_InvalidCredentialJSON(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)

	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        "registration",
		UserID:      "user-1",
		SessionJSON: "{}",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("not-json"),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestFinishPasskeyRegistration_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	_, err := svc.FinishPasskeyRegistration(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestFinishPasskeyRegistration_NilPasskeyStore(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{SessionId: "session", CredentialResponseJson: []byte("{}")})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestLoadPasskeySessionKindMismatch(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        "registration",
		SessionJSON: "{}",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	svc := NewAuthService(userStore, passkeyStore, nil)

	_, err := svc.loadPasskeySession(context.Background(), "session-1", passkey.SessionKindLogin)
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestLoadPasskeySessionExpired(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        "login",
		SessionJSON: "{}",
		ExpiresAt:   time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC),
	}
	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return time.Date(2026, 2, 12, 10, 1, 0, 0, time.UTC) }

	_, err := svc.loadPasskeySession(context.Background(), "session-1", passkey.SessionKindLogin)
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.InvalidArgument)
	if _, ok := passkeyStore.sessions["session-1"]; ok {
		t.Fatalf("expected session to be deleted")
	}
}

func TestStorePasskeyCredential(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC) }

	credential := webauthn.Credential{ID: []byte("cred-1")}
	if err := svc.storePasskeyCredential(context.Background(), "user-1", credential, false); err != nil {
		t.Fatalf("store credential: %v", err)
	}
	stored, ok := passkeyStore.credentials[encodeCredentialID(credential.ID)]
	if !ok {
		t.Fatalf("expected credential stored")
	}
	if stored.LastUsedAt != nil {
		t.Fatalf("expected nil last used at")
	}
}

func TestStorePasskeyCredentialUsedMissing(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)

	credential := webauthn.Credential{ID: []byte("cred-1")}
	if err := svc.storePasskeyCredential(context.Background(), "user-1", credential, true); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDecodeStoredCredentialsInvalidJSON(t *testing.T) {
	_, err := decodeStoredCredentials([]storage.PasskeyCredential{{
		CredentialID:   "cred-1",
		CredentialJSON: "not-json",
	}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDefaultPasskeyParserErrors(t *testing.T) {
	parser := defaultPasskeyParser{}
	if _, err := parser.ParseCredentialCreationResponseBytes([]byte("not-json")); err == nil {
		t.Fatalf("expected creation parse error")
	}
	if _, err := parser.ParseCredentialRequestResponseBytes([]byte("not-json")); err == nil {
		t.Fatalf("expected request parse error")
	}
}

func TestPasskeyUserMethods(t *testing.T) {
	credential := webauthn.Credential{ID: []byte("cred-1")}
	user := passkeyUser{user: user.User{ID: "user-1", DisplayName: "Alpha"}, credentials: []webauthn.Credential{credential}}
	if string(user.WebAuthnID()) != "user-1" {
		t.Fatalf("expected WebAuthnID")
	}
	if user.WebAuthnName() != "user-1" {
		t.Fatalf("expected WebAuthnName to be user id")
	}
	if user.WebAuthnDisplayName() != "Alpha" {
		t.Fatalf("expected WebAuthnDisplayName")
	}
	if user.WebAuthnIcon() != "" {
		t.Fatalf("expected empty WebAuthnIcon")
	}
	if len(user.WebAuthnCredentials()) != 1 {
		t.Fatalf("expected credentials")
	}
}

func TestLoadPasskeyUser(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	base := user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	userStore.users["user-1"] = base

	credential := webauthn.Credential{ID: []byte("cred-1")}
	jsonPayload, err := json.Marshal(credential)
	if err != nil {
		t.Fatalf("marshal credential: %v", err)
	}
	passkeyStore.credentials[encodeCredentialID(credential.ID)] = storage.PasskeyCredential{
		CredentialID:   encodeCredentialID(credential.ID),
		UserID:         "user-1",
		CredentialJSON: string(jsonPayload),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	loaded, err := svc.loadPasskeyUser(context.Background(), base)
	if err != nil {
		t.Fatalf("load passkey user: %v", err)
	}
	if len(loaded.credentials) != 1 {
		t.Fatalf("expected credential loaded")
	}
}

func TestStorePasskeySession(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.clock = func() time.Time { return time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC) }

	sessionData := webauthn.SessionData{}
	if err := svc.storePasskeySession(context.Background(), "session-1", passkey.SessionKindLogin, "user-1", &sessionData); err != nil {
		t.Fatalf("store session: %v", err)
	}
	stored, ok := passkeyStore.sessions["session-1"]
	if !ok {
		t.Fatalf("expected stored session")
	}
	if stored.Kind != "login" {
		t.Fatalf("unexpected kind")
	}
}

func TestPasskeyUserHandler(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	base := user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	userStore.users["user-1"] = base

	credential := webauthn.Credential{ID: []byte("cred-1")}
	jsonPayload, err := json.Marshal(credential)
	if err != nil {
		t.Fatalf("marshal credential: %v", err)
	}
	passkeyStore.credentials[encodeCredentialID(credential.ID)] = storage.PasskeyCredential{
		CredentialID:   encodeCredentialID(credential.ID),
		UserID:         "user-1",
		CredentialJSON: string(jsonPayload),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	svc := NewAuthService(userStore, passkeyStore, nil)
	handler := svc.passkeyUserHandler(context.Background())
	userResult, err := handler(nil, []byte("user-1"))
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if userResult == nil {
		t.Fatalf("expected user")
	}
}

func TestPasskeyUserHandlerMissingUserHandle(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	handler := svc.passkeyUserHandler(context.Background())
	if _, err := handler(nil, nil); err == nil {
		t.Fatalf("expected error")
	}
}

func TestStorePasskeyCredentialSetsLastUsedAt(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	credential := webauthn.Credential{ID: []byte("cred-1")}
	passkeyStore.credentials[encodeCredentialID(credential.ID)] = storage.PasskeyCredential{
		CredentialID:   encodeCredentialID(credential.ID),
		UserID:         "user-1",
		CredentialJSON: "{}",
		CreatedAt:      now.Add(-time.Hour),
		UpdatedAt:      now.Add(-time.Hour),
	}

	if err := svc.storePasskeyCredential(context.Background(), "user-1", credential, true); err != nil {
		t.Fatalf("store credential: %v", err)
	}
	stored := passkeyStore.credentials[encodeCredentialID(credential.ID)]
	if stored.LastUsedAt == nil {
		t.Fatalf("expected last used at")
	}
}

func TestStorePasskeySessionRequiresSession(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)

	if err := svc.storePasskeySession(context.Background(), "session-1", passkey.SessionKindLogin, "user-1", nil); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadPasskeySessionInvalidJSON(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        "login",
		SessionJSON: "not-json",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	svc := NewAuthService(userStore, passkeyStore, nil)

	_, err := svc.loadPasskeySession(context.Background(), "session-1", passkey.SessionKindLogin)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadPasskeyUserListError(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	passkeyStore.listErr = status.Error(codes.Internal, "list failed")
	svc := NewAuthService(userStore, passkeyStore, nil)

	_, err := svc.loadPasskeyUser(context.Background(), user.User{ID: "user-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestFinishPasskeyRegistrationParserMissing(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyParser = nil

	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{SessionId: "session", CredentialResponseJson: []byte("{}")})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestFinishPasskeyLoginParserMissing(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyParser = nil

	_, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{SessionId: "session", CredentialResponseJson: []byte("{}")})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
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
		t.Fatalf("expected error")
	}
}

func TestAttachPendingAuthorizationNoStore(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), newFakePasskeyStore(), nil)
	if err := svc.attachPendingAuthorization(context.Background(), "pending", "user-1"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFinishPasskeyRegistration_Success(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()

	store := openTempOAuthStore(t)
	svc := NewAuthService(userStore, passkeyStore, store)
	svc.passkeyWebAuthn = &fakePasskeyProvider{credential: &webauthn.Credential{ID: []byte("cred-1")}}
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	sessionData := webauthn.SessionData{}
	payload, err := json.Marshal(sessionData)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        "registration",
		UserID:      "user-1",
		SessionJSON: string(payload),
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	resp, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("finish passkey registration: %v", err)
	}
	if resp.GetCredentialId() == "" {
		t.Fatalf("expected credential id")
	}
	if _, ok := passkeyStore.credentials[resp.GetCredentialId()]; !ok {
		t.Fatalf("expected credential stored")
	}
	if _, ok := passkeyStore.sessions["session-1"]; ok {
		t.Fatalf("expected session deleted")
	}
}

func TestFinishPasskeyRegistrationMissingUserID(t *testing.T) {
	userStore := newFakeUserStore()
	passkeyStore := newFakePasskeyStore()
	svc := NewAuthService(userStore, passkeyStore, nil)
	svc.passkeyWebAuthn = &fakePasskeyProvider{}
	svc.passkeyParser = &fakePasskeyParser{creation: &protocol.ParsedCredentialCreationData{}}

	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        "registration",
		SessionJSON: "{}",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	_, err := svc.FinishPasskeyRegistration(context.Background(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("{}"),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	assertStatusCode(t, err, codes.Internal)
}

func TestFinishPasskeyLogin_AttachesPendingAuthorization(t *testing.T) {
	userStore := newFakeUserStore()
	userStore.users["user-1"] = user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	passkeyStore := newFakePasskeyStore()

	oauthStore := openTempOAuthStore(t)
	pendingID, err := oauthStore.CreatePendingAuthorization(oauth.AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            "test-client",
		RedirectURI:         "http://localhost:5555/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	}, time.Minute)
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}

	svc := NewAuthService(userStore, passkeyStore, oauthStore)
	svc.passkeyWebAuthn = &fakePasskeyProvider{
		loginUser:  &passkeyUser{user: userStore.users["user-1"]},
		credential: &webauthn.Credential{ID: []byte("cred-1")},
	}
	svc.passkeyParser = &fakePasskeyParser{assertion: &protocol.ParsedCredentialAssertionData{}}
	passkeyStore.credentials[encodeCredentialID([]byte("cred-1"))] = storage.PasskeyCredential{
		CredentialID:   encodeCredentialID([]byte("cred-1")),
		UserID:         "user-1",
		CredentialJSON: "{}",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	sessionData := webauthn.SessionData{}
	payload, err := json.Marshal(sessionData)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	passkeyStore.sessions["session-1"] = storage.PasskeySession{
		ID:          "session-1",
		Kind:        "login",
		UserID:      "user-1",
		SessionJSON: string(payload),
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	resp, err := svc.FinishPasskeyLogin(context.Background(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              "session-1",
		CredentialResponseJson: []byte("{}"),
		PendingId:              pendingID,
	})
	if err != nil {
		t.Fatalf("finish passkey login: %v", err)
	}
	if resp.GetCredentialId() == "" {
		t.Fatalf("expected credential id")
	}
	if pending, err := oauthStore.GetPendingAuthorization(pendingID); err != nil || pending == nil || pending.UserID != "user-1" {
		t.Fatalf("expected pending authorization updated, got %v", err)
	}
}

type fakePasskeyProvider struct {
	credential           *webauthn.Credential
	loginUser            webauthn.User
	beginRegistrationErr error
	beginLoginErr        error
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

func (f *fakePasskeyProvider) BeginDiscoverableLogin(opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	if f.beginLoginErr != nil {
		return nil, nil, f.beginLoginErr
	}
	return &protocol.CredentialAssertion{}, &webauthn.SessionData{}, nil
}

func (f *fakePasskeyProvider) ValidatePasskeyLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (webauthn.User, *webauthn.Credential, error) {
	if f.loginUser == nil {
		return nil, nil, status.Error(codes.Unauthenticated, "missing user")
	}
	credential := f.credential
	if credential == nil {
		credential = &webauthn.Credential{ID: []byte("cred")}
	}
	return f.loginUser, credential, nil
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
