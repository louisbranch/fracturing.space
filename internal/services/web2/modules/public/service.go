package public

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web2/platform/errors"
)

type service struct {
	auth authGateway
}

type authGateway interface {
	CreateUser(context.Context, *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error)
	BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest) (*authv1.BeginPasskeyRegistrationResponse, error)
	FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest) (*authv1.FinishPasskeyRegistrationResponse, error)
	BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest) (*authv1.BeginPasskeyLoginResponse, error)
	FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest) (*authv1.FinishPasskeyLoginResponse, error)
	CreateWebSession(context.Context, *authv1.CreateWebSessionRequest) (*authv1.CreateWebSessionResponse, error)
	GetWebSession(context.Context, *authv1.GetWebSessionRequest) (*authv1.GetWebSessionResponse, error)
	RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest) (*authv1.RevokeWebSessionResponse, error)
}

type grpcAuthGateway struct {
	client module.AuthClient
}

type unavailableAuthGateway struct{}

const authServiceUnavailableMessage = "auth service is not configured"

type passkeyStart struct {
	sessionID string
	publicKey json.RawMessage
}

type passkeyRegisterStart struct {
	sessionID string
	userID    string
	publicKey json.RawMessage
}

type passkeyFinish struct {
	sessionID string
	userID    string
}

func newService(deps module.Dependencies) service {
	if deps.AuthClient != nil {
		return service{auth: grpcAuthGateway{client: deps.AuthClient}}
	}
	return service{auth: unavailableAuthGateway{}}
}

func (service) healthBody() string {
	return "ok"
}

func mapAuthGatewayError(err error, fallbackKind apperrors.Kind, fallbackMessage string) error {
	if err == nil {
		return nil
	}
	// TODO(web2-errors): preserve richer upstream grpc/app error kinds so fallback mapping does not blur invalid-input vs dependency failures.
	if apperrors.HTTPStatus(err) == http.StatusServiceUnavailable {
		return apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
	}
	return apperrors.E(fallbackKind, fallbackMessage)
}

func (s service) passkeyLoginStart(ctx context.Context) (passkeyStart, error) {
	resp, err := s.auth.BeginPasskeyLogin(ctx, &authv1.BeginPasskeyLoginRequest{})
	if err != nil {
		return passkeyStart{}, mapAuthGatewayError(err, apperrors.KindInvalidInput, "failed to start passkey login")
	}
	if strings.TrimSpace(resp.GetSessionId()) == "" {
		return passkeyStart{}, apperrors.E(apperrors.KindUnknown, "auth did not return login session")
	}
	return passkeyStart{sessionID: resp.GetSessionId(), publicKey: json.RawMessage(resp.GetCredentialRequestOptionsJson())}, nil
}

func (s service) passkeyLoginFinish(ctx context.Context, sessionID string, credential json.RawMessage) (passkeyFinish, error) {
	if strings.TrimSpace(sessionID) == "" {
		return passkeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "session_id is required")
	}
	if len(credential) == 0 {
		return passkeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "credential is required")
	}
	resp, err := s.auth.FinishPasskeyLogin(ctx, &authv1.FinishPasskeyLoginRequest{SessionId: sessionID, CredentialResponseJson: credential})
	if err != nil {
		return passkeyFinish{}, mapAuthGatewayError(err, apperrors.KindInvalidInput, "failed to finish passkey login")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return passkeyFinish{}, apperrors.E(apperrors.KindUnknown, "auth did not return user id")
	}
	session, err := s.auth.CreateWebSession(ctx, &authv1.CreateWebSessionRequest{UserId: userID})
	if err != nil {
		return passkeyFinish{}, mapAuthGatewayError(err, apperrors.KindUnknown, "failed to create web session")
	}
	webSessionID := strings.TrimSpace(session.GetSession().GetId())
	if webSessionID == "" {
		return passkeyFinish{}, apperrors.E(apperrors.KindUnknown, "auth did not return web session id")
	}
	return passkeyFinish{sessionID: webSessionID, userID: userID}, nil
}

func (s service) passkeyRegisterStart(ctx context.Context, email string) (passkeyRegisterStart, error) {
	if strings.TrimSpace(email) == "" {
		return passkeyRegisterStart{}, apperrors.E(apperrors.KindInvalidInput, "email is required")
	}
	created, err := s.auth.CreateUser(ctx, &authv1.CreateUserRequest{Email: strings.TrimSpace(email), Locale: commonv1.Locale_LOCALE_EN_US})
	if err != nil {
		return passkeyRegisterStart{}, mapAuthGatewayError(err, apperrors.KindInvalidInput, "failed to create user")
	}
	userID := strings.TrimSpace(created.GetUser().GetId())
	if userID == "" {
		return passkeyRegisterStart{}, apperrors.E(apperrors.KindUnknown, "auth did not return user id")
	}
	begin, err := s.auth.BeginPasskeyRegistration(ctx, &authv1.BeginPasskeyRegistrationRequest{UserId: userID})
	if err != nil {
		return passkeyRegisterStart{}, mapAuthGatewayError(err, apperrors.KindInvalidInput, "failed to start passkey registration")
	}
	if strings.TrimSpace(begin.GetSessionId()) == "" {
		return passkeyRegisterStart{}, apperrors.E(apperrors.KindUnknown, "auth did not return registration session")
	}
	return passkeyRegisterStart{sessionID: begin.GetSessionId(), userID: userID, publicKey: json.RawMessage(begin.GetCredentialCreationOptionsJson())}, nil
}

func (s service) passkeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (passkeyFinish, error) {
	if strings.TrimSpace(sessionID) == "" {
		return passkeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "session_id is required")
	}
	if len(credential) == 0 {
		return passkeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "credential is required")
	}
	resp, err := s.auth.FinishPasskeyRegistration(ctx, &authv1.FinishPasskeyRegistrationRequest{SessionId: sessionID, CredentialResponseJson: credential})
	if err != nil {
		return passkeyFinish{}, mapAuthGatewayError(err, apperrors.KindInvalidInput, "failed to finish passkey registration")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return passkeyFinish{}, apperrors.E(apperrors.KindUnknown, "auth did not return user id")
	}
	return passkeyFinish{userID: userID}, nil
}

func (s service) hasValidWebSession(ctx context.Context, sessionID string) bool {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false
	}
	resp, err := s.auth.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil {
		return false
	}
	if resp == nil || resp.GetSession() == nil {
		return false
	}
	return strings.TrimSpace(resp.GetSession().GetId()) != ""
}

func (s service) revokeWebSession(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	_, err := s.auth.RevokeWebSession(ctx, &authv1.RevokeWebSessionRequest{SessionId: sessionID})
	if err != nil {
		return mapAuthGatewayError(err, apperrors.KindUnknown, "failed to revoke web session")
	}
	return nil
}

func (g grpcAuthGateway) CreateUser(ctx context.Context, req *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
	return g.client.CreateUser(ctx, req)
}

func (g grpcAuthGateway) BeginPasskeyRegistration(ctx context.Context, req *authv1.BeginPasskeyRegistrationRequest) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return g.client.BeginPasskeyRegistration(ctx, req)
}

func (g grpcAuthGateway) FinishPasskeyRegistration(ctx context.Context, req *authv1.FinishPasskeyRegistrationRequest) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return g.client.FinishPasskeyRegistration(ctx, req)
}

func (g grpcAuthGateway) BeginPasskeyLogin(ctx context.Context, req *authv1.BeginPasskeyLoginRequest) (*authv1.BeginPasskeyLoginResponse, error) {
	return g.client.BeginPasskeyLogin(ctx, req)
}

func (g grpcAuthGateway) FinishPasskeyLogin(ctx context.Context, req *authv1.FinishPasskeyLoginRequest) (*authv1.FinishPasskeyLoginResponse, error) {
	return g.client.FinishPasskeyLogin(ctx, req)
}

func (g grpcAuthGateway) CreateWebSession(ctx context.Context, req *authv1.CreateWebSessionRequest) (*authv1.CreateWebSessionResponse, error) {
	return g.client.CreateWebSession(ctx, req)
}

func (g grpcAuthGateway) GetWebSession(ctx context.Context, req *authv1.GetWebSessionRequest) (*authv1.GetWebSessionResponse, error) {
	return g.client.GetWebSession(ctx, req)
}

func (g grpcAuthGateway) RevokeWebSession(ctx context.Context, req *authv1.RevokeWebSessionRequest) (*authv1.RevokeWebSessionResponse, error) {
	return g.client.RevokeWebSession(ctx, req)
}

func (unavailableAuthGateway) CreateUser(context.Context, *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest) (*authv1.BeginPasskeyLoginResponse, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest) (*authv1.FinishPasskeyLoginResponse, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) CreateWebSession(context.Context, *authv1.CreateWebSessionRequest) (*authv1.CreateWebSessionResponse, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) GetWebSession(context.Context, *authv1.GetWebSessionRequest) (*authv1.GetWebSessionResponse, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableAuthGateway) RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest) (*authv1.RevokeWebSessionResponse, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}
