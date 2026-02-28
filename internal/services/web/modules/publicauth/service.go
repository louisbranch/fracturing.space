package publicauth

import (
	"context"
	"encoding/json"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type service struct {
	auth AuthGateway
}

// AuthGateway abstracts authentication operations behind domain types.
type AuthGateway interface {
	// CreateUser creates a new user account, returning the user ID.
	CreateUser(ctx context.Context, email string) (string, error)
	// BeginPasskeyRegistration starts passkey registration for a user.
	BeginPasskeyRegistration(ctx context.Context, userID string) (passkeyChallenge, error)
	// FinishPasskeyRegistration completes registration and returns the user ID.
	FinishPasskeyRegistration(ctx context.Context, sessionID string, credential json.RawMessage) (string, error)
	// BeginPasskeyLogin starts a passkey login flow.
	BeginPasskeyLogin(ctx context.Context) (passkeyChallenge, error)
	// FinishPasskeyLogin completes login and returns the user ID.
	FinishPasskeyLogin(ctx context.Context, sessionID string, credential json.RawMessage) (string, error)
	// CreateWebSession creates a session for the given user, returning the session ID.
	CreateWebSession(ctx context.Context, userID string) (string, error)
	// HasValidWebSession checks whether a session exists and is valid.
	HasValidWebSession(ctx context.Context, sessionID string) bool
	// RevokeWebSession invalidates a web session.
	RevokeWebSession(ctx context.Context, sessionID string) error
}

// passkeyChallenge holds the session and public key from a passkey begin operation.
type passkeyChallenge struct {
	SessionID string
	PublicKey json.RawMessage
}

type passkeyRegisterResult struct {
	SessionID string
	UserID    string
	PublicKey json.RawMessage
}

type passkeyFinish struct {
	SessionID string
	UserID    string
}

func newServiceWithGateway(gateway AuthGateway) service {
	if gateway == nil {
		gateway = unavailableAuthGateway{}
	}
	return service{auth: gateway}
}

func (service) healthBody() string {
	return "ok"
}

func (s service) passkeyLoginStart(ctx context.Context) (passkeyChallenge, error) {
	return s.auth.BeginPasskeyLogin(ctx)
}

func (s service) passkeyLoginFinish(ctx context.Context, sessionID string, credential json.RawMessage) (passkeyFinish, error) {
	if strings.TrimSpace(sessionID) == "" {
		return passkeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "session_id is required")
	}
	if len(credential) == 0 {
		return passkeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "credential is required")
	}
	userID, err := s.auth.FinishPasskeyLogin(ctx, sessionID, credential)
	if err != nil {
		return passkeyFinish{}, err
	}
	webSessionID, err := s.auth.CreateWebSession(ctx, userID)
	if err != nil {
		return passkeyFinish{}, err
	}
	return passkeyFinish{SessionID: webSessionID, UserID: userID}, nil
}

func (s service) passkeyRegisterStart(ctx context.Context, email string) (passkeyRegisterResult, error) {
	if strings.TrimSpace(email) == "" {
		return passkeyRegisterResult{}, apperrors.E(apperrors.KindInvalidInput, "email is required")
	}
	userID, err := s.auth.CreateUser(ctx, strings.TrimSpace(email))
	if err != nil {
		return passkeyRegisterResult{}, err
	}
	challenge, err := s.auth.BeginPasskeyRegistration(ctx, userID)
	if err != nil {
		return passkeyRegisterResult{}, err
	}
	return passkeyRegisterResult{SessionID: challenge.SessionID, UserID: userID, PublicKey: challenge.PublicKey}, nil
}

func (s service) passkeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (passkeyFinish, error) {
	if strings.TrimSpace(sessionID) == "" {
		return passkeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "session_id is required")
	}
	if len(credential) == 0 {
		return passkeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "credential is required")
	}
	userID, err := s.auth.FinishPasskeyRegistration(ctx, sessionID, credential)
	if err != nil {
		return passkeyFinish{}, err
	}
	return passkeyFinish{UserID: userID}, nil
}

func (s service) hasValidWebSession(ctx context.Context, sessionID string) bool {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false
	}
	return s.auth.HasValidWebSession(ctx, sessionID)
}

func (s service) revokeWebSession(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	return s.auth.RevokeWebSession(ctx, sessionID)
}

// --- gRPC gateway ---

type grpcAuthGateway struct {
	client AuthClient
}

// NewGRPCAuthGateway builds an AuthGateway backed by gRPC auth client calls.
func NewGRPCAuthGateway(client AuthClient) AuthGateway {
	if client == nil {
		return unavailableAuthGateway{}
	}
	return newGRPCAuthGateway(client)
}

func newGRPCAuthGateway(client AuthClient) grpcAuthGateway {
	return grpcAuthGateway{client: client}
}

const authServiceUnavailableMessage = "auth service is not configured"

func mapAuthGatewayError(err error, fallbackKind apperrors.Kind, fallbackMessage string) error {
	return mapAuthGatewayErrorWithKey(err, fallbackKind, "", fallbackMessage)
}

func mapAuthGatewayErrorWithKey(err error, fallbackKind apperrors.Kind, fallbackKey string, fallbackMessage string) error {
	return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
		FallbackKind:    fallbackKind,
		FallbackKey:     fallbackKey,
		FallbackMessage: fallbackMessage,
	})
}

func (g grpcAuthGateway) CreateUser(ctx context.Context, email string) (string, error) {
	resp, err := g.client.CreateUser(ctx, &authv1.CreateUserRequest{
		Email:  email,
		Locale: commonv1.Locale_LOCALE_EN_US,
	})
	if err != nil {
		return "", mapAuthGatewayErrorWithKey(err, apperrors.KindInvalidInput, "error.http.failed_to_create_user", "failed to create user")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "auth did not return user id")
	}
	return userID, nil
}

func (g grpcAuthGateway) BeginPasskeyRegistration(ctx context.Context, userID string) (passkeyChallenge, error) {
	resp, err := g.client.BeginPasskeyRegistration(ctx, &authv1.BeginPasskeyRegistrationRequest{UserId: userID})
	if err != nil {
		return passkeyChallenge{}, mapAuthGatewayError(err, apperrors.KindInvalidInput, "failed to start passkey registration")
	}
	sessionID := strings.TrimSpace(resp.GetSessionId())
	if sessionID == "" {
		return passkeyChallenge{}, apperrors.E(apperrors.KindUnknown, "auth did not return registration session")
	}
	return passkeyChallenge{SessionID: sessionID, PublicKey: json.RawMessage(resp.GetCredentialCreationOptionsJson())}, nil
}

func (g grpcAuthGateway) FinishPasskeyRegistration(ctx context.Context, sessionID string, credential json.RawMessage) (string, error) {
	resp, err := g.client.FinishPasskeyRegistration(ctx, &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              sessionID,
		CredentialResponseJson: credential,
	})
	if err != nil {
		return "", mapAuthGatewayError(err, apperrors.KindInvalidInput, "failed to finish passkey registration")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "auth did not return user id")
	}
	return userID, nil
}

func (g grpcAuthGateway) BeginPasskeyLogin(ctx context.Context) (passkeyChallenge, error) {
	resp, err := g.client.BeginPasskeyLogin(ctx, &authv1.BeginPasskeyLoginRequest{})
	if err != nil {
		return passkeyChallenge{}, mapAuthGatewayError(err, apperrors.KindInvalidInput, "failed to start passkey login")
	}
	sessionID := strings.TrimSpace(resp.GetSessionId())
	if sessionID == "" {
		return passkeyChallenge{}, apperrors.E(apperrors.KindUnknown, "auth did not return login session")
	}
	return passkeyChallenge{SessionID: sessionID, PublicKey: json.RawMessage(resp.GetCredentialRequestOptionsJson())}, nil
}

func (g grpcAuthGateway) FinishPasskeyLogin(ctx context.Context, sessionID string, credential json.RawMessage) (string, error) {
	resp, err := g.client.FinishPasskeyLogin(ctx, &authv1.FinishPasskeyLoginRequest{
		SessionId:              sessionID,
		CredentialResponseJson: credential,
	})
	if err != nil {
		return "", mapAuthGatewayError(err, apperrors.KindInvalidInput, "failed to finish passkey login")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "auth did not return user id")
	}
	return userID, nil
}

func (g grpcAuthGateway) CreateWebSession(ctx context.Context, userID string) (string, error) {
	resp, err := g.client.CreateWebSession(ctx, &authv1.CreateWebSessionRequest{UserId: userID})
	if err != nil {
		return "", mapAuthGatewayError(err, apperrors.KindUnknown, "failed to create web session")
	}
	sessionID := strings.TrimSpace(resp.GetSession().GetId())
	if sessionID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "auth did not return web session id")
	}
	return sessionID, nil
}

func (g grpcAuthGateway) HasValidWebSession(ctx context.Context, sessionID string) bool {
	resp, err := g.client.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil || resp == nil || resp.GetSession() == nil {
		return false
	}
	return strings.TrimSpace(resp.GetSession().GetId()) != ""
}

func (g grpcAuthGateway) RevokeWebSession(ctx context.Context, sessionID string) error {
	_, err := g.client.RevokeWebSession(ctx, &authv1.RevokeWebSessionRequest{SessionId: sessionID})
	return mapAuthGatewayError(err, apperrors.KindUnknown, "failed to revoke web session")
}

// --- Unavailable gateway ---

type unavailableAuthGateway struct{}

func (unavailableAuthGateway) CreateUser(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) BeginPasskeyRegistration(context.Context, string) (passkeyChallenge, error) {
	return passkeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) FinishPasskeyRegistration(context.Context, string, json.RawMessage) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) BeginPasskeyLogin(context.Context) (passkeyChallenge, error) {
	return passkeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) FinishPasskeyLogin(context.Context, string, json.RawMessage) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) CreateWebSession(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) HasValidWebSession(context.Context, string) bool {
	return false
}

func (unavailableAuthGateway) RevokeWebSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}
