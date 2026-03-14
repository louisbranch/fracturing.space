package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/auth/passkey"
	"github.com/louisbranch/fracturing.space/internal/services/auth/recoverycode"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BeginAccountRegistration reserves a username and starts the first passkey ceremony.
func (s *AuthService) BeginAccountRegistration(ctx context.Context, in *authv1.BeginAccountRegistrationRequest) (*authv1.BeginAccountRegistrationResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Begin account registration request is required.")
	}
	if s.store == nil || s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "Auth stores are not configured.")
	}
	if s.passkeyInitErr != nil || s.passkeyWebAuthn == nil {
		return nil, status.Error(codes.Internal, "Passkey configuration is not available.")
	}

	normalized, err := user.NormalizeCreateUserInput(user.CreateUserInput{
		Username: in.GetUsername(),
		Locale:   in.GetLocale(),
	})
	if err != nil {
		return nil, handleDomainError(err)
	}
	if _, err := s.store.GetUserByUsername(ctx, normalized.Username); err == nil {
		return nil, status.Error(codes.AlreadyExists, "Username is already in use.")
	} else if err != storage.ErrNotFound {
		return nil, handleDomainError(err)
	}

	userID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Generate user ID: %v", err)
	}
	passkeyUser := &passkeyUser{user: user.User{ID: userID, Username: normalized.Username}}
	creation, session, err := s.passkeyWebAuthn.BeginRegistration(passkeyUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Begin account registration: %v", err)
	}

	sessionID, err := s.newPasskeySessionID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Create registration session: %v", err)
	}
	now := s.clock().UTC()
	if err := s.passkeyStore.DeleteExpiredRegistrationSessions(ctx, now); err != nil {
		return nil, status.Errorf(codes.Internal, "Delete expired registration sessions: %v", err)
	}
	if err := s.passkeyStore.PutRegistrationSession(ctx, storage.RegistrationSession{
		ID:        sessionID,
		UserID:    userID,
		Username:  normalized.Username,
		Locale:    platformi18n.LocaleString(normalized.Locale),
		ExpiresAt: now.Add(s.passkeyConfig.SessionTTL),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "Store registration session: %v", err)
	}
	if err := s.storePasskeySession(ctx, sessionID, passkey.SessionKindRegistration, userID, session); err != nil {
		_ = s.passkeyStore.DeleteRegistrationSession(ctx, sessionID)
		return nil, status.Errorf(codes.Internal, "Store passkey session: %v", err)
	}
	optionsJSON, err := marshalCreationOptions(creation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Encode registration options: %v", err)
	}
	return &authv1.BeginAccountRegistrationResponse{
		SessionId:                     sessionID,
		CredentialCreationOptionsJson: optionsJSON,
	}, nil
}

// FinishAccountRegistration creates the account, stores the first passkey, and issues recovery state.
func (s *AuthService) FinishAccountRegistration(ctx context.Context, in *authv1.FinishAccountRegistrationRequest) (*authv1.FinishAccountRegistrationResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Finish account registration request is required.")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "Session ID is required.")
	}
	registration, err := s.passkeyStore.GetRegistrationSession(ctx, sessionID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, status.Error(codes.NotFound, "Registration session not found.")
		}
		return nil, status.Errorf(codes.Internal, "Get registration session: %v", err)
	}
	if registration.ExpiresAt.Before(s.clock().UTC()) {
		_ = s.passkeyStore.DeleteRegistrationSession(ctx, sessionID)
		return nil, status.Error(codes.InvalidArgument, "Registration session expired.")
	}
	loaded, err := s.loadPasskeySession(ctx, sessionID, passkey.SessionKindRegistration)
	if err != nil {
		return nil, err
	}
	if loaded.UserID == "" || loaded.UserID != registration.UserID {
		return nil, status.Error(codes.Internal, "Registration session user mismatch.")
	}
	if len(in.GetCredentialResponseJson()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Credential response JSON is required.")
	}

	locale := parseStoredLocale(registration.Locale)
	created, err := user.CreateUser(user.CreateUserInput{
		Username: registration.Username,
		Locale:   locale,
	}, s.clock, func() (string, error) { return registration.UserID, nil })
	if err != nil {
		return nil, handleDomainError(err)
	}
	recoveryCode, recoveryHash, err := s.generateRecoveryCode()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Generate recovery code: %v", err)
	}
	created.RecoveryCodeHash = recoveryHash

	passkeyUser := &passkeyUser{user: created}
	parsed, err := s.passkeyParser.ParseCredentialCreationResponseBytes(in.GetCredentialResponseJson())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Parse credential response: %v", err)
	}
	credential, err := s.passkeyWebAuthn.CreateCredential(passkeyUser, loaded.Data, parsed)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Validate credential response: %v", err)
	}
	storedCredential, err := s.buildPasskeyCredentialRecord(ctx, created.ID, *credential, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Prepare passkey credential: %v", err)
	}
	event, err := s.signupCompletedOutboxEvent(created, "passkey")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Prepare signup outbox event: %v", err)
	}
	webSession, err := s.buildWebSession(created.ID, defaultWebSessionTTL)
	if err != nil {
		return nil, err
	}
	if err := s.persistRegisteredUser(ctx, created, storedCredential, webSession, event); err != nil {
		return nil, status.Errorf(codes.Internal, "Persist registered user: %v", err)
	}
	_ = s.passkeyStore.DeletePasskeySession(ctx, sessionID)
	_ = s.passkeyStore.DeleteRegistrationSession(ctx, sessionID)
	return &authv1.FinishAccountRegistrationResponse{
		User:         userToProto(created),
		CredentialId: encodeCredentialID(credential.ID),
		Session:      webSessionToProto(webSession),
		RecoveryCode: recoveryCode,
	}, nil
}

func (s *AuthService) persistRegisteredUser(ctx context.Context, baseUser user.User, credential storage.PasskeyCredential, session storage.WebSession, event storage.IntegrationOutboxEvent) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("User store is not configured.")
	}
	if s.passkeyStore == nil {
		return fmt.Errorf("Passkey store is not configured.")
	}

	txStore, ok := s.passkeyStore.(storage.UserSignupTransactionalStore)
	if !ok {
		return fmt.Errorf("Signup persistence is not configured.")
	}
	return txStore.PutUserPasskeyWithIntegrationOutboxEvent(ctx, baseUser, credential, session, event)
}

// BeginAccountRecovery verifies a recovery code and creates a narrow recovery session.
func (s *AuthService) BeginAccountRecovery(ctx context.Context, in *authv1.BeginAccountRecoveryRequest) (*authv1.BeginAccountRecoveryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Begin account recovery request is required.")
	}
	if s.store == nil || s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "Auth stores are not configured.")
	}
	baseUser, err := s.store.GetUserByUsername(ctx, strings.TrimSpace(in.GetUsername()))
	if err != nil {
		return nil, handleDomainError(err)
	}
	now := s.clock().UTC()
	if baseUser.RecoveryReservedSessionID != "" && baseUser.RecoveryReservedUntil != nil {
		if baseUser.RecoveryReservedUntil.After(now) {
			return nil, status.Error(codes.FailedPrecondition, "Recovery is already in progress.")
		}
		baseUser.RecoveryReservedSessionID = ""
		baseUser.RecoveryReservedUntil = nil
		baseUser.UpdatedAt = now
		if err := s.store.PutUser(ctx, baseUser); err != nil {
			return nil, status.Errorf(codes.Internal, "Clear expired recovery reservation: %v", err)
		}
	}
	if !recoverycode.Verify(in.GetRecoveryCode(), baseUser.RecoveryCodeHash) {
		return nil, status.Error(codes.InvalidArgument, "Recovery code is invalid.")
	}

	recoverySessionID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Generate recovery session ID: %v", err)
	}
	expiresAt := now.Add(defaultRecoverySessionTTL)
	if err := s.passkeyStore.PutRecoverySession(ctx, storage.RecoverySession{
		ID:        recoverySessionID,
		UserID:    baseUser.ID,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "Put recovery session: %v", err)
	}
	baseUser.RecoveryReservedSessionID = recoverySessionID
	baseUser.RecoveryReservedUntil = &expiresAt
	baseUser.UpdatedAt = now
	if err := s.store.PutUser(ctx, baseUser); err != nil {
		return nil, status.Errorf(codes.Internal, "Reserve recovery session: %v", err)
	}
	return &authv1.BeginAccountRecoveryResponse{RecoverySessionId: recoverySessionID}, nil
}

// BeginRecoveryPasskeyRegistration starts replacement passkey registration for a recovery flow.
func (s *AuthService) BeginRecoveryPasskeyRegistration(ctx context.Context, in *authv1.BeginRecoveryPasskeyRegistrationRequest) (*authv1.BeginPasskeyRegistrationResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Begin recovery passkey registration request is required.")
	}
	recoverySession, baseUser, err := s.loadRecoverySessionUser(ctx, in.GetRecoverySessionId())
	if err != nil {
		return nil, err
	}
	passkeyUser, err := s.loadPasskeyUser(ctx, baseUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Load passkey user: %v", err)
	}
	options := []webauthn.RegistrationOption{}
	if len(passkeyUser.credentials) > 0 {
		options = append(options, webauthn.WithExclusions(webauthn.Credentials(passkeyUser.credentials).CredentialDescriptors()))
	}
	creation, session, err := s.passkeyWebAuthn.BeginRegistration(passkeyUser, options...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Begin recovery registration: %v", err)
	}
	sessionID, err := s.newPasskeySessionID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Create passkey session: %v", err)
	}
	if err := s.storePasskeySession(ctx, sessionID, passkey.SessionKindRegistration, recoverySession.UserID, session); err != nil {
		return nil, status.Errorf(codes.Internal, "Store passkey session: %v", err)
	}
	optionsJSON, err := marshalCreationOptions(creation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Encode registration options: %v", err)
	}
	return &authv1.BeginPasskeyRegistrationResponse{
		SessionId:                     sessionID,
		CredentialCreationOptionsJson: optionsJSON,
	}, nil
}

// FinishRecoveryPasskeyRegistration completes recovery and rekeys the account.
func (s *AuthService) FinishRecoveryPasskeyRegistration(ctx context.Context, in *authv1.FinishRecoveryPasskeyRegistrationRequest) (*authv1.FinishAccountRegistrationResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Finish recovery passkey registration request is required.")
	}
	recoverySession, baseUser, err := s.loadRecoverySessionUser(ctx, in.GetRecoverySessionId())
	if err != nil {
		return nil, err
	}
	loaded, err := s.loadPasskeySession(ctx, strings.TrimSpace(in.GetSessionId()), passkey.SessionKindRegistration)
	if err != nil {
		return nil, err
	}
	if loaded.UserID != recoverySession.UserID {
		return nil, status.Error(codes.InvalidArgument, "Passkey session user mismatch.")
	}
	passkeyUser, err := s.loadPasskeyUser(ctx, baseUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Load passkey user: %v", err)
	}
	parsed, err := s.passkeyParser.ParseCredentialCreationResponseBytes(in.GetCredentialResponseJson())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Parse credential response: %v", err)
	}
	credential, err := s.passkeyWebAuthn.CreateCredential(passkeyUser, loaded.Data, parsed)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Validate credential response: %v", err)
	}
	if err := s.storePasskeyCredential(ctx, baseUser.ID, *credential, false); err != nil {
		return nil, status.Errorf(codes.Internal, "Store recovery passkey credential: %v", err)
	}
	recoveryCode, recoveryHash, err := s.generateRecoveryCode()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Generate recovery code: %v", err)
	}
	now := s.clock().UTC()
	baseUser.RecoveryCodeHash = recoveryHash
	baseUser.RecoveryReservedSessionID = ""
	baseUser.RecoveryReservedUntil = nil
	baseUser.RecoveryCodeUpdatedAt = now
	baseUser.UpdatedAt = now
	if err := s.store.PutUser(ctx, baseUser); err != nil {
		return nil, status.Errorf(codes.Internal, "Rotate recovery code: %v", err)
	}
	if err := s.webSessionStore.RevokeWebSessionsByUser(ctx, baseUser.ID, now); err != nil {
		return nil, status.Errorf(codes.Internal, "Revoke web sessions: %v", err)
	}
	if err := s.passkeyStore.DeletePasskeyCredentialsByUserExcept(ctx, baseUser.ID, encodeCredentialID(credential.ID)); err != nil {
		return nil, status.Errorf(codes.Internal, "Delete old passkeys: %v", err)
	}
	_ = s.passkeyStore.DeletePasskeySession(ctx, strings.TrimSpace(in.GetSessionId()))
	_ = s.passkeyStore.DeleteRecoverySession(ctx, recoverySession.ID)

	if pendingID := strings.TrimSpace(in.GetPendingId()); pendingID != "" {
		if err := s.attachPendingAuthorization(ctx, pendingID, baseUser.ID); err != nil {
			return nil, err
		}
	}

	webSession, err := s.createWebSession(ctx, baseUser.ID, defaultWebSessionTTL)
	if err != nil {
		return nil, err
	}
	return &authv1.FinishAccountRegistrationResponse{
		User:         userToProto(baseUser),
		CredentialId: encodeCredentialID(credential.ID),
		Session:      webSessionToProto(webSession),
		RecoveryCode: recoveryCode,
	}, nil
}

// ListPasskeys lists registered passkeys for one user.
func (s *AuthService) ListPasskeys(ctx context.Context, in *authv1.ListPasskeysRequest) (*authv1.ListPasskeysResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "List passkeys request is required.")
	}
	if s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "Passkey store is not configured.")
	}
	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "User ID is required.")
	}
	rows, err := s.passkeyStore.ListPasskeyCredentials(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "List passkeys: %v", err)
	}
	response := &authv1.ListPasskeysResponse{Passkeys: make([]*authv1.PasskeyCredentialSummary, 0, len(rows))}
	for _, row := range rows {
		item := &authv1.PasskeyCredentialSummary{
			CredentialId: row.CredentialID,
			CreatedAt:    timestamppb.New(row.CreatedAt),
			UpdatedAt:    timestamppb.New(row.UpdatedAt),
		}
		if row.LastUsedAt != nil {
			item.LastUsedAt = timestamppb.New(*row.LastUsedAt)
		}
		response.Passkeys = append(response.Passkeys, item)
	}
	return response, nil
}

func parseStoredLocale(raw string) commonv1.Locale {
	if parsed, ok := platformi18n.ParseLocale(raw); ok {
		return parsed
	}
	return platformi18n.DefaultLocale()
}

func (s *AuthService) loadRecoverySessionUser(ctx context.Context, recoverySessionID string) (storage.RecoverySession, user.User, error) {
	if s.passkeyStore == nil || s.store == nil {
		return storage.RecoverySession{}, user.User{}, status.Error(codes.Internal, "Auth stores are not configured.")
	}
	recoverySessionID = strings.TrimSpace(recoverySessionID)
	if recoverySessionID == "" {
		return storage.RecoverySession{}, user.User{}, status.Error(codes.InvalidArgument, "Recovery session ID is required.")
	}
	recoverySession, err := s.passkeyStore.GetRecoverySession(ctx, recoverySessionID)
	if err != nil {
		if err == storage.ErrNotFound {
			return storage.RecoverySession{}, user.User{}, status.Error(codes.NotFound, "Recovery session not found.")
		}
		return storage.RecoverySession{}, user.User{}, status.Errorf(codes.Internal, "Get recovery session: %v", err)
	}
	if recoverySession.ExpiresAt.Before(s.clock().UTC()) {
		_ = s.passkeyStore.DeleteRecoverySession(ctx, recoverySessionID)
		return storage.RecoverySession{}, user.User{}, status.Error(codes.InvalidArgument, "Recovery session expired.")
	}
	baseUser, err := s.store.GetUser(ctx, recoverySession.UserID)
	if err != nil {
		return storage.RecoverySession{}, user.User{}, handleDomainError(err)
	}
	if baseUser.RecoveryReservedSessionID != recoverySessionID {
		return storage.RecoverySession{}, user.User{}, status.Error(codes.FailedPrecondition, "Recovery session is no longer active.")
	}
	return recoverySession, baseUser, nil
}

func marshalCreationOptions(creation *protocol.CredentialCreation) ([]byte, error) {
	return json.Marshal(creation)
}
