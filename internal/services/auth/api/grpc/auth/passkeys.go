package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/auth/passkey"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AuthService) BeginPasskeyRegistration(ctx context.Context, in *authv1.BeginPasskeyRegistrationRequest) (*authv1.BeginPasskeyRegistrationResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Begin passkey registration request is required.")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "User store is not configured.")
	}
	if s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "Passkey store is not configured.")
	}
	if s.passkeyInitErr != nil || s.passkeyWebAuthn == nil {
		return nil, status.Error(codes.Internal, "Passkey configuration is not available.")
	}
	if s.passkeyParser == nil {
		return nil, status.Error(codes.Internal, "Passkey parser is not configured.")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "User ID is required.")
	}
	baseUser, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	passkeyUser, err := s.loadPasskeyUser(ctx, baseUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Load passkey user: %v", err)
	}

	options := []webauthn.RegistrationOption{
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
	}
	if len(passkeyUser.credentials) > 0 {
		options = append(options, webauthn.WithExclusions(webauthn.Credentials(passkeyUser.credentials).CredentialDescriptors()))
	}

	creation, session, err := s.passkeyWebAuthn.BeginRegistration(passkeyUser, options...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Begin passkey registration: %v", err)
	}

	sessionID, err := s.newPasskeySessionID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Create passkey session: %v", err)
	}
	if err := s.storePasskeySession(ctx, sessionID, passkey.SessionKindRegistration, baseUser.ID, session); err != nil {
		return nil, status.Errorf(codes.Internal, "Store passkey session: %v", err)
	}
	optionsJSON, err := json.Marshal(creation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Encode registration options: %v", err)
	}

	return &authv1.BeginPasskeyRegistrationResponse{
		SessionId:                     sessionID,
		CredentialCreationOptionsJson: optionsJSON,
	}, nil
}

type passkeyProvider interface {
	BeginRegistration(user webauthn.User, opts ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error)
	CreateCredential(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error)
	BeginLogin(user webauthn.User, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	ValidateLogin(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error)
}

type passkeyParser interface {
	ParseCredentialCreationResponseBytes(data []byte) (*protocol.ParsedCredentialCreationData, error)
	ParseCredentialRequestResponseBytes(data []byte) (*protocol.ParsedCredentialAssertionData, error)
}

type defaultPasskeyParser struct{}

func (defaultPasskeyParser) ParseCredentialCreationResponseBytes(data []byte) (*protocol.ParsedCredentialCreationData, error) {
	return protocol.ParseCredentialCreationResponseBytes(data)
}

func (defaultPasskeyParser) ParseCredentialRequestResponseBytes(data []byte) (*protocol.ParsedCredentialAssertionData, error) {
	return protocol.ParseCredentialRequestResponseBytes(data)
}

func (s *AuthService) FinishPasskeyRegistration(ctx context.Context, in *authv1.FinishPasskeyRegistrationRequest) (*authv1.FinishPasskeyRegistrationResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Finish passkey registration request is required.")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "User store is not configured.")
	}
	if s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "Passkey store is not configured.")
	}
	if s.passkeyInitErr != nil || s.passkeyWebAuthn == nil {
		return nil, status.Error(codes.Internal, "Passkey configuration is not available.")
	}
	if s.passkeyParser == nil {
		return nil, status.Error(codes.Internal, "Passkey parser is not configured.")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "Session ID is required.")
	}
	if len(in.GetCredentialResponseJson()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Credential response JSON is required.")
	}

	session, err := s.loadPasskeySession(ctx, sessionID, passkey.SessionKindRegistration)
	if err != nil {
		return nil, err
	}
	if session.UserID == "" {
		return nil, status.Error(codes.Internal, "Passkey session is missing a user ID.")
	}

	baseUser, err := s.store.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	passkeyUser, err := s.loadPasskeyUser(ctx, baseUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Load passkey user: %v", err)
	}

	parsed, err := s.passkeyParser.ParseCredentialCreationResponseBytes(in.GetCredentialResponseJson())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Parse credential response: %v", err)
	}
	credential, err := s.passkeyWebAuthn.CreateCredential(passkeyUser, session.Data, parsed)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Validate credential response: %v", err)
	}

	if err := s.storePasskeyCredential(ctx, baseUser.ID, *credential, false); err != nil {
		return nil, status.Errorf(codes.Internal, "Store passkey credential: %v", err)
	}
	_ = s.passkeyStore.DeletePasskeySession(ctx, sessionID)

	return &authv1.FinishPasskeyRegistrationResponse{
		User:         userToProto(baseUser),
		CredentialId: encodeCredentialID(credential.ID),
	}, nil
}

func (s *AuthService) BeginPasskeyLogin(ctx context.Context, in *authv1.BeginPasskeyLoginRequest) (*authv1.BeginPasskeyLoginResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Begin passkey login request is required.")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "User store is not configured.")
	}
	if s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "Passkey store is not configured.")
	}
	if s.passkeyInitErr != nil || s.passkeyWebAuthn == nil {
		return nil, status.Error(codes.Internal, "Passkey configuration is not available.")
	}
	if s.passkeyParser == nil {
		return nil, status.Error(codes.Internal, "Passkey parser is not configured.")
	}

	username := strings.TrimSpace(in.GetUsername())
	if username == "" {
		return nil, status.Error(codes.InvalidArgument, "Username is required.")
	}
	baseUser, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, handleDomainError(err)
	}
	passkeyUser, err := s.loadPasskeyUser(ctx, baseUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Load passkey user: %v", err)
	}
	assertion, session, err := s.passkeyWebAuthn.BeginLogin(passkeyUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Begin passkey login: %v", err)
	}

	sessionID, err := s.newPasskeySessionID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Create passkey session: %v", err)
	}
	if err := s.storePasskeySession(ctx, sessionID, passkey.SessionKindLogin, baseUser.ID, session); err != nil {
		return nil, status.Errorf(codes.Internal, "Store passkey session: %v", err)
	}
	optionsJSON, err := json.Marshal(assertion)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Encode login options: %v", err)
	}

	return &authv1.BeginPasskeyLoginResponse{
		SessionId:                    sessionID,
		CredentialRequestOptionsJson: optionsJSON,
	}, nil
}

func (s *AuthService) FinishPasskeyLogin(ctx context.Context, in *authv1.FinishPasskeyLoginRequest) (*authv1.FinishPasskeyLoginResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Finish passkey login request is required.")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "User store is not configured.")
	}
	if s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "Passkey store is not configured.")
	}
	if s.passkeyInitErr != nil || s.passkeyWebAuthn == nil {
		return nil, status.Error(codes.Internal, "Passkey configuration is not available.")
	}
	if s.passkeyParser == nil {
		return nil, status.Error(codes.Internal, "Passkey parser is not configured.")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "Session ID is required.")
	}
	if len(in.GetCredentialResponseJson()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Credential response JSON is required.")
	}

	session, err := s.loadPasskeySession(ctx, sessionID, passkey.SessionKindLogin)
	if err != nil {
		return nil, err
	}

	parsed, err := s.passkeyParser.ParseCredentialRequestResponseBytes(in.GetCredentialResponseJson())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Parse credential response: %v", err)
	}

	if session.UserID == "" {
		return nil, status.Error(codes.Internal, "Passkey session is missing a user ID.")
	}
	baseUser, err := s.store.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	passkeyUser, err := s.loadPasskeyUser(ctx, baseUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Load passkey user: %v", err)
	}
	validatedCredential, err := s.passkeyWebAuthn.ValidateLogin(passkeyUser, session.Data, parsed)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Validate passkey login: %v", err)
	}

	if err := s.storePasskeyCredential(ctx, baseUser.ID, *validatedCredential, true); err != nil {
		return nil, status.Errorf(codes.Internal, "Store passkey credential: %v", err)
	}
	_ = s.passkeyStore.DeletePasskeySession(ctx, sessionID)

	if pendingID := strings.TrimSpace(in.GetPendingId()); pendingID != "" {
		if err := s.attachPendingAuthorization(ctx, pendingID, baseUser.ID); err != nil {
			return nil, err
		}
	}

	return &authv1.FinishPasskeyLoginResponse{
		User:         userToProto(baseUser),
		CredentialId: encodeCredentialID(validatedCredential.ID),
	}, nil
}

type passkeyUser struct {
	user        user.User
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte {
	return []byte(u.user.ID)
}

func (u *passkeyUser) WebAuthnName() string {
	return u.user.Username
}

func (u *passkeyUser) WebAuthnDisplayName() string {
	return u.user.Username
}

func (u *passkeyUser) WebAuthnIcon() string {
	return ""
}

func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (s *AuthService) loadPasskeyUser(ctx context.Context, base user.User) (*passkeyUser, error) {
	credentials, err := s.passkeyStore.ListPasskeyCredentials(ctx, base.ID)
	if err != nil {
		return nil, err
	}
	parsed, err := decodeStoredCredentials(credentials)
	if err != nil {
		return nil, err
	}
	return &passkeyUser{user: base, credentials: parsed}, nil
}

func decodeStoredCredentials(records []storage.PasskeyCredential) ([]webauthn.Credential, error) {
	if len(records) == 0 {
		return nil, nil
	}
	credentials := make([]webauthn.Credential, 0, len(records))
	for _, record := range records {
		var credential webauthn.Credential
		if err := json.Unmarshal([]byte(record.CredentialJSON), &credential); err != nil {
			return nil, fmt.Errorf("Decode credential %s: %w", record.CredentialID, err)
		}
		credentials = append(credentials, credential)
	}
	return credentials, nil
}

func (s *AuthService) buildPasskeyCredentialRecord(ctx context.Context, userID string, credential webauthn.Credential, used bool) (storage.PasskeyCredential, error) {
	credentialID := encodeCredentialID(credential.ID)
	now := s.clock().UTC()
	stored, err := s.passkeyStore.GetPasskeyCredential(ctx, credentialID)
	if err != nil && err != storage.ErrNotFound {
		return storage.PasskeyCredential{}, err
	}
	if err == storage.ErrNotFound && used {
		return storage.PasskeyCredential{}, fmt.Errorf("Passkey credential not found.")
	}

	createdAt := now
	if err == nil {
		createdAt = stored.CreatedAt
	}
	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		return storage.PasskeyCredential{}, err
	}
	var lastUsed *time.Time
	if used {
		value := now
		lastUsed = &value
	}
	return storage.PasskeyCredential{
		CredentialID:   credentialID,
		UserID:         userID,
		CredentialJSON: string(credentialJSON),
		CreatedAt:      createdAt,
		UpdatedAt:      now,
		LastUsedAt:     lastUsed,
	}, nil
}

func (s *AuthService) storePasskeyCredential(ctx context.Context, userID string, credential webauthn.Credential, used bool) error {
	record, err := s.buildPasskeyCredentialRecord(ctx, userID, credential, used)
	if err != nil {
		return err
	}
	return s.passkeyStore.PutPasskeyCredential(ctx, record)
}

func (s *AuthService) storePasskeySession(ctx context.Context, sessionID string, kind passkey.SessionKind, userID string, session *webauthn.SessionData) error {
	if session == nil {
		return fmt.Errorf("Session data is required.")
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.passkeyStore.PutPasskeySession(ctx, storage.PasskeySession{
		ID:          sessionID,
		Kind:        string(kind),
		UserID:      userID,
		SessionJSON: string(payload),
		ExpiresAt:   s.clock().UTC().Add(s.passkeyConfig.SessionTTL),
	})
}

type loadedSession struct {
	Data   webauthn.SessionData
	Kind   passkey.SessionKind
	UserID string
}

func (s *AuthService) loadPasskeySession(ctx context.Context, sessionID string, expectedKind passkey.SessionKind) (loadedSession, error) {
	stored, err := s.passkeyStore.GetPasskeySession(ctx, sessionID)
	if err != nil {
		if err == storage.ErrNotFound {
			return loadedSession{}, status.Error(codes.NotFound, "Passkey session not found.")
		}
		return loadedSession{}, status.Errorf(codes.Internal, "Load passkey session: %v", err)
	}
	if stored.Kind != string(expectedKind) {
		return loadedSession{}, status.Error(codes.InvalidArgument, "Passkey session kind mismatch.")
	}
	if stored.ExpiresAt.Before(s.clock().UTC()) {
		_ = s.passkeyStore.DeletePasskeySession(ctx, sessionID)
		return loadedSession{}, status.Error(codes.InvalidArgument, "Passkey session expired.")
	}

	var session webauthn.SessionData
	if err := json.Unmarshal([]byte(stored.SessionJSON), &session); err != nil {
		return loadedSession{}, status.Errorf(codes.Internal, "Decode passkey session: %v", err)
	}
	return loadedSession{Data: session, Kind: expectedKind, UserID: stored.UserID}, nil
}

func (s *AuthService) newPasskeySessionID() (string, error) {
	if s.passkeyIDGenerator != nil {
		return s.passkeyIDGenerator()
	}
	return id.NewID()
}

func encodeCredentialID(raw []byte) string {
	return base64.RawURLEncoding.EncodeToString(raw)
}

func (s *AuthService) attachPendingAuthorization(ctx context.Context, pendingID string, userID string) error {
	if s.oauthStore == nil {
		return status.Error(codes.Internal, "OAuth store is not configured.")
	}
	if strings.TrimSpace(pendingID) == "" || strings.TrimSpace(userID) == "" {
		return status.Error(codes.InvalidArgument, "Pending ID and user ID are required.")
	}
	pending, err := s.oauthStore.GetPendingAuthorization(pendingID)
	if err != nil || pending == nil {
		return status.Error(codes.NotFound, "Authorization session not found.")
	}
	if pending.ExpiresAt.Before(s.clock().UTC()) {
		s.oauthStore.DeletePendingAuthorization(pendingID)
		return status.Error(codes.InvalidArgument, "Authorization session expired.")
	}
	if err := s.oauthStore.UpdatePendingAuthorizationUserID(pendingID, userID); err != nil {
		return status.Errorf(codes.Internal, "Update authorization: %v", err)
	}
	return nil
}
