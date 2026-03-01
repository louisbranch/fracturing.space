package gateway

import (
	"context"
	"encoding/json"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

// AuthClient performs passkey and user bootstrap operations.
type AuthClient interface {
	CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error)
	BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error)
	FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error)
	BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error)
	FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error)
	CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error)
	GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error)
	RevokeWebSession(context.Context, *authv1.RevokeWebSessionRequest, ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error)
}

// GRPCGateway maps auth gRPC responses into publicauth app contracts.
type GRPCGateway struct {
	Client AuthClient
}

// NewGRPCGateway builds a publicauth gateway backed by gRPC auth client calls.
func NewGRPCGateway(client AuthClient) publicauthapp.Gateway {
	if client == nil {
		return publicauthapp.NewUnavailableGateway()
	}
	return newGRPCGateway(client)
}

func newGRPCGateway(client AuthClient) GRPCGateway {
	return GRPCGateway{Client: client}
}

func mapGRPCError(err error, fallbackKind apperrors.Kind, fallbackMessage string) error {
	return mapGRPCErrorWithKey(err, fallbackKind, "", fallbackMessage)
}

func mapGRPCErrorWithKey(err error, fallbackKind apperrors.Kind, fallbackKey string, fallbackMessage string) error {
	return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
		FallbackKind:    fallbackKind,
		FallbackKey:     fallbackKey,
		FallbackMessage: fallbackMessage,
	})
}

func (g GRPCGateway) CreateUser(ctx context.Context, email string) (string, error) {
	resp, err := g.Client.CreateUser(ctx, &authv1.CreateUserRequest{
		Email:  email,
		Locale: commonv1.Locale_LOCALE_EN_US,
	})
	if err != nil {
		return "", mapGRPCErrorWithKey(err, apperrors.KindInvalidInput, "error.http.failed_to_create_user", "failed to create user")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "auth did not return user id")
	}
	return userID, nil
}

func (g GRPCGateway) BeginPasskeyRegistration(ctx context.Context, userID string) (publicauthapp.PasskeyChallenge, error) {
	resp, err := g.Client.BeginPasskeyRegistration(ctx, &authv1.BeginPasskeyRegistrationRequest{UserId: userID})
	if err != nil {
		return publicauthapp.PasskeyChallenge{}, mapGRPCError(err, apperrors.KindInvalidInput, "failed to start passkey registration")
	}
	sessionID := strings.TrimSpace(resp.GetSessionId())
	if sessionID == "" {
		return publicauthapp.PasskeyChallenge{}, apperrors.E(apperrors.KindUnknown, "auth did not return registration session")
	}
	return publicauthapp.PasskeyChallenge{SessionID: sessionID, PublicKey: json.RawMessage(resp.GetCredentialCreationOptionsJson())}, nil
}

func (g GRPCGateway) FinishPasskeyRegistration(ctx context.Context, sessionID string, credential json.RawMessage) (string, error) {
	resp, err := g.Client.FinishPasskeyRegistration(ctx, &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              sessionID,
		CredentialResponseJson: credential,
	})
	if err != nil {
		return "", mapGRPCError(err, apperrors.KindInvalidInput, "failed to finish passkey registration")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "auth did not return user id")
	}
	return userID, nil
}

func (g GRPCGateway) BeginPasskeyLogin(ctx context.Context) (publicauthapp.PasskeyChallenge, error) {
	resp, err := g.Client.BeginPasskeyLogin(ctx, &authv1.BeginPasskeyLoginRequest{})
	if err != nil {
		return publicauthapp.PasskeyChallenge{}, mapGRPCError(err, apperrors.KindInvalidInput, "failed to start passkey login")
	}
	sessionID := strings.TrimSpace(resp.GetSessionId())
	if sessionID == "" {
		return publicauthapp.PasskeyChallenge{}, apperrors.E(apperrors.KindUnknown, "auth did not return login session")
	}
	return publicauthapp.PasskeyChallenge{SessionID: sessionID, PublicKey: json.RawMessage(resp.GetCredentialRequestOptionsJson())}, nil
}

func (g GRPCGateway) FinishPasskeyLogin(ctx context.Context, sessionID string, credential json.RawMessage) (string, error) {
	resp, err := g.Client.FinishPasskeyLogin(ctx, &authv1.FinishPasskeyLoginRequest{
		SessionId:              sessionID,
		CredentialResponseJson: credential,
	})
	if err != nil {
		return "", mapGRPCError(err, apperrors.KindInvalidInput, "failed to finish passkey login")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "auth did not return user id")
	}
	return userID, nil
}

func (g GRPCGateway) CreateWebSession(ctx context.Context, userID string) (string, error) {
	resp, err := g.Client.CreateWebSession(ctx, &authv1.CreateWebSessionRequest{UserId: userID})
	if err != nil {
		return "", mapGRPCError(err, apperrors.KindUnknown, "failed to create web session")
	}
	sessionID := strings.TrimSpace(resp.GetSession().GetId())
	if sessionID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "auth did not return web session id")
	}
	return sessionID, nil
}

func (g GRPCGateway) HasValidWebSession(ctx context.Context, sessionID string) bool {
	resp, err := g.Client.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil || resp == nil || resp.GetSession() == nil {
		return false
	}
	return strings.TrimSpace(resp.GetSession().GetId()) != ""
}

func (g GRPCGateway) RevokeWebSession(ctx context.Context, sessionID string) error {
	_, err := g.Client.RevokeWebSession(ctx, &authv1.RevokeWebSessionRequest{SessionId: sessionID})
	return mapGRPCError(err, apperrors.KindUnknown, "failed to revoke web session")
}
