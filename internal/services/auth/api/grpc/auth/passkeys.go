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
		return nil, status.Error(codes.InvalidArgument, "begin passkey registration request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}
	if s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "passkey store is not configured")
	}
	if s.passkeyInitErr != nil || s.passkeyWebAuthn == nil {
		return nil, status.Error(codes.Internal, "passkey configuration is not available")
	}
	if s.passkeyParser == nil {
		return nil, status.Error(codes.Internal, "passkey parser is not configured")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	baseUser, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	passkeyUser, err := s.loadPasskeyUser(ctx, baseUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load passkey user: %v", err)
	}

	options := []webauthn.RegistrationOption{
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
	}
	if len(passkeyUser.credentials) > 0 {
		options = append(options, webauthn.WithExclusions(webauthn.Credentials(passkeyUser.credentials).CredentialDescriptors()))
	}

	creation, session, err := s.passkeyWebAuthn.BeginRegistration(passkeyUser, options...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin passkey registration: %v", err)
	}

	sessionID, err := s.newPasskeySessionID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create passkey session: %v", err)
	}
	if err := s.storePasskeySession(ctx, sessionID, passkey.SessionKindRegistration, baseUser.ID, session); err != nil {
		return nil, status.Errorf(codes.Internal, "store passkey session: %v", err)
	}
	optionsJSON, err := json.Marshal(creation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode registration options: %v", err)
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
	BeginDiscoverableLogin(opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	ValidatePasskeyLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (webauthn.User, *webauthn.Credential, error)
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
		return nil, status.Error(codes.InvalidArgument, "finish passkey registration request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}
	if s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "passkey store is not configured")
	}
	if s.passkeyInitErr != nil || s.passkeyWebAuthn == nil {
		return nil, status.Error(codes.Internal, "passkey configuration is not available")
	}
	if s.passkeyParser == nil {
		return nil, status.Error(codes.Internal, "passkey parser is not configured")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if len(in.GetCredentialResponseJson()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "credential response json is required")
	}

	session, err := s.loadPasskeySession(ctx, sessionID, passkey.SessionKindRegistration)
	if err != nil {
		return nil, err
	}
	if session.UserID == "" {
		return nil, status.Error(codes.Internal, "passkey session missing user id")
	}

	baseUser, err := s.store.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	passkeyUser, err := s.loadPasskeyUser(ctx, baseUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load passkey user: %v", err)
	}

	parsed, err := s.passkeyParser.ParseCredentialCreationResponseBytes(in.GetCredentialResponseJson())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parse credential response: %v", err)
	}
	credential, err := s.passkeyWebAuthn.CreateCredential(passkeyUser, session.Data, parsed)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "validate credential response: %v", err)
	}

	if err := s.storePasskeyCredential(ctx, baseUser.ID, *credential, false); err != nil {
		return nil, status.Errorf(codes.Internal, "store passkey credential: %v", err)
	}
	_ = s.passkeyStore.DeletePasskeySession(ctx, sessionID)

	return &authv1.FinishPasskeyRegistrationResponse{
		User:         userToProto(baseUser),
		CredentialId: encodeCredentialID(credential.ID),
	}, nil
}

func (s *AuthService) BeginPasskeyLogin(ctx context.Context, in *authv1.BeginPasskeyLoginRequest) (*authv1.BeginPasskeyLoginResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "begin passkey login request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}
	if s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "passkey store is not configured")
	}
	if s.passkeyInitErr != nil || s.passkeyWebAuthn == nil {
		return nil, status.Error(codes.Internal, "passkey configuration is not available")
	}
	if s.passkeyParser == nil {
		return nil, status.Error(codes.Internal, "passkey parser is not configured")
	}

	userID := strings.TrimSpace(in.GetUserId())
	var (
		assertion *protocol.CredentialAssertion
		session   *webauthn.SessionData
		err       error
	)

	if userID == "" {
		assertion, session, err = s.passkeyWebAuthn.BeginDiscoverableLogin()
	} else {
		baseUser, err := s.store.GetUser(ctx, userID)
		if err != nil {
			return nil, handleDomainError(err)
		}
		passkeyUser, err := s.loadPasskeyUser(ctx, baseUser)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load passkey user: %v", err)
		}
		assertion, session, err = s.passkeyWebAuthn.BeginLogin(passkeyUser)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin passkey login: %v", err)
	}

	sessionID, err := s.newPasskeySessionID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create passkey session: %v", err)
	}
	if err := s.storePasskeySession(ctx, sessionID, passkey.SessionKindLogin, userID, session); err != nil {
		return nil, status.Errorf(codes.Internal, "store passkey session: %v", err)
	}
	optionsJSON, err := json.Marshal(assertion)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode login options: %v", err)
	}

	return &authv1.BeginPasskeyLoginResponse{
		SessionId:                    sessionID,
		CredentialRequestOptionsJson: optionsJSON,
	}, nil
}

func (s *AuthService) FinishPasskeyLogin(ctx context.Context, in *authv1.FinishPasskeyLoginRequest) (*authv1.FinishPasskeyLoginResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "finish passkey login request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}
	if s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "passkey store is not configured")
	}
	if s.passkeyInitErr != nil || s.passkeyWebAuthn == nil {
		return nil, status.Error(codes.Internal, "passkey configuration is not available")
	}
	if s.passkeyParser == nil {
		return nil, status.Error(codes.Internal, "passkey parser is not configured")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if len(in.GetCredentialResponseJson()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "credential response json is required")
	}

	session, err := s.loadPasskeySession(ctx, sessionID, passkey.SessionKindLogin)
	if err != nil {
		return nil, err
	}

	parsed, err := s.passkeyParser.ParseCredentialRequestResponseBytes(in.GetCredentialResponseJson())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parse credential response: %v", err)
	}

	validatedUser, validatedCredential, err := s.passkeyWebAuthn.ValidatePasskeyLogin(s.passkeyUserHandler(ctx), session.Data, parsed)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "validate passkey login: %v", err)
	}

	userRecord, ok := validatedUser.(*passkeyUser)
	if !ok {
		return nil, status.Error(codes.Internal, "passkey user type mismatch")
	}

	if err := s.storePasskeyCredential(ctx, userRecord.user.ID, *validatedCredential, true); err != nil {
		return nil, status.Errorf(codes.Internal, "store passkey credential: %v", err)
	}
	_ = s.passkeyStore.DeletePasskeySession(ctx, sessionID)

	if pendingID := strings.TrimSpace(in.GetPendingId()); pendingID != "" {
		if err := s.attachPendingAuthorization(ctx, pendingID, userRecord.user.ID); err != nil {
			return nil, err
		}
	}

	return &authv1.FinishPasskeyLoginResponse{
		User:         userToProto(userRecord.user),
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
	return u.user.ID
}

func (u *passkeyUser) WebAuthnDisplayName() string {
	return u.user.DisplayName
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
			return nil, fmt.Errorf("decode credential %s: %w", record.CredentialID, err)
		}
		credentials = append(credentials, credential)
	}
	return credentials, nil
}

func (s *AuthService) storePasskeyCredential(ctx context.Context, userID string, credential webauthn.Credential, used bool) error {
	credentialID := encodeCredentialID(credential.ID)
	now := s.clock().UTC()
	stored, err := s.passkeyStore.GetPasskeyCredential(ctx, credentialID)
	if err != nil && err != storage.ErrNotFound {
		return err
	}
	if err == storage.ErrNotFound && used {
		return fmt.Errorf("passkey credential not found")
	}

	createdAt := now
	if err == nil {
		createdAt = stored.CreatedAt
	}
	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	var lastUsed *time.Time
	if used {
		value := now
		lastUsed = &value
	}
	return s.passkeyStore.PutPasskeyCredential(ctx, storage.PasskeyCredential{
		CredentialID:   credentialID,
		UserID:         userID,
		CredentialJSON: string(credentialJSON),
		CreatedAt:      createdAt,
		UpdatedAt:      now,
		LastUsedAt:     lastUsed,
	})
}

func (s *AuthService) storePasskeySession(ctx context.Context, sessionID string, kind passkey.SessionKind, userID string, session *webauthn.SessionData) error {
	if session == nil {
		return fmt.Errorf("session data is required")
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
			return loadedSession{}, status.Error(codes.NotFound, "passkey session not found")
		}
		return loadedSession{}, status.Errorf(codes.Internal, "load passkey session: %v", err)
	}
	if stored.Kind != string(expectedKind) {
		return loadedSession{}, status.Error(codes.InvalidArgument, "passkey session kind mismatch")
	}
	if stored.ExpiresAt.Before(s.clock().UTC()) {
		_ = s.passkeyStore.DeletePasskeySession(ctx, sessionID)
		return loadedSession{}, status.Error(codes.InvalidArgument, "passkey session expired")
	}

	var session webauthn.SessionData
	if err := json.Unmarshal([]byte(stored.SessionJSON), &session); err != nil {
		return loadedSession{}, status.Errorf(codes.Internal, "decode passkey session: %v", err)
	}
	return loadedSession{Data: session, Kind: expectedKind, UserID: stored.UserID}, nil
}

func (s *AuthService) passkeyUserHandler(ctx context.Context) webauthn.DiscoverableUserHandler {
	return func(_, userHandle []byte) (webauthn.User, error) {
		userID := string(userHandle)
		if strings.TrimSpace(userID) == "" {
			return nil, fmt.Errorf("user handle is required")
		}
		baseUser, err := s.store.GetUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		return s.loadPasskeyUser(ctx, baseUser)
	}
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
		return status.Error(codes.Internal, "oauth store is not configured")
	}
	if strings.TrimSpace(pendingID) == "" || strings.TrimSpace(userID) == "" {
		return status.Error(codes.InvalidArgument, "pending id and user id are required")
	}
	pending, err := s.oauthStore.GetPendingAuthorization(pendingID)
	if err != nil || pending == nil {
		return status.Error(codes.NotFound, "authorization session not found")
	}
	if pending.ExpiresAt.Before(s.clock().UTC()) {
		s.oauthStore.DeletePendingAuthorization(pendingID)
		return status.Error(codes.InvalidArgument, "authorization session expired")
	}
	if err := s.oauthStore.UpdatePendingAuthorizationUserID(pendingID, userID); err != nil {
		return status.Errorf(codes.Internal, "update authorization: %v", err)
	}
	return nil
}
