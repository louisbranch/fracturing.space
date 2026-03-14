package gateway

import (
	"context"
	"encoding/json"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

// AuthClient performs passkey and user bootstrap operations.
type AuthClient interface {
	BeginAccountRegistration(context.Context, *authv1.BeginAccountRegistrationRequest, ...grpc.CallOption) (*authv1.BeginAccountRegistrationResponse, error)
	FinishAccountRegistration(context.Context, *authv1.FinishAccountRegistrationRequest, ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error)
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

// newGRPCGateway builds package wiring for this web seam.
func newGRPCGateway(client AuthClient) GRPCGateway {
	return GRPCGateway{Client: client}
}

// mapGRPCError maps values across transport and domain boundaries.
func mapGRPCError(err error, fallbackKind apperrors.Kind, fallbackMessage string) error {
	return mapGRPCErrorWithKey(err, fallbackKind, "", fallbackMessage)
}

// mapGRPCErrorWithKey maps values across transport and domain boundaries.
func mapGRPCErrorWithKey(err error, fallbackKind apperrors.Kind, fallbackKey string, fallbackMessage string) error {
	return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
		FallbackKind:    fallbackKind,
		FallbackKey:     fallbackKey,
		FallbackMessage: fallbackMessage,
	})
}

// BeginAccountRegistration starts username-backed registration.
func (g GRPCGateway) BeginAccountRegistration(ctx context.Context, username string) (publicauthapp.PasskeyChallenge, error) {
	resp, err := g.Client.BeginAccountRegistration(ctx, &authv1.BeginAccountRegistrationRequest{Username: username})
	if err != nil {
		return publicauthapp.PasskeyChallenge{}, mapGRPCErrorWithKey(err, apperrors.KindInvalidInput, "error.http.failed_to_create_user", "Failed to create user.")
	}
	sessionID := strings.TrimSpace(resp.GetSessionId())
	if sessionID == "" {
		return publicauthapp.PasskeyChallenge{}, apperrors.E(apperrors.KindUnknown, "Auth did not return a registration session.")
	}
	return publicauthapp.PasskeyChallenge{SessionID: sessionID, PublicKey: json.RawMessage(resp.GetCredentialCreationOptionsJson())}, nil
}

// FinishAccountRegistration centralizes this web behavior in one helper seam.
func (g GRPCGateway) FinishAccountRegistration(ctx context.Context, sessionID string, credential json.RawMessage) (publicauthapp.PasskeyFinish, error) {
	resp, err := g.Client.FinishAccountRegistration(ctx, &authv1.FinishAccountRegistrationRequest{
		SessionId:              sessionID,
		CredentialResponseJson: credential,
	})
	if err != nil {
		return publicauthapp.PasskeyFinish{}, mapGRPCError(err, apperrors.KindInvalidInput, "Failed to finish passkey registration.")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return publicauthapp.PasskeyFinish{}, apperrors.E(apperrors.KindUnknown, "Auth did not return a user ID.")
	}
	webSessionID := strings.TrimSpace(resp.GetSession().GetId())
	if webSessionID == "" {
		return publicauthapp.PasskeyFinish{}, apperrors.E(apperrors.KindUnknown, "Auth did not return a web session ID.")
	}
	return publicauthapp.PasskeyFinish{
		SessionID:    webSessionID,
		UserID:       userID,
		RecoveryCode: strings.TrimSpace(resp.GetRecoveryCode()),
	}, nil
}

// BeginPasskeyLogin centralizes this web behavior in one helper seam.
func (g GRPCGateway) BeginPasskeyLogin(ctx context.Context, username string) (publicauthapp.PasskeyChallenge, error) {
	resp, err := g.Client.BeginPasskeyLogin(ctx, &authv1.BeginPasskeyLoginRequest{Username: username})
	if err != nil {
		return publicauthapp.PasskeyChallenge{}, mapGRPCError(err, apperrors.KindInvalidInput, "Failed to start passkey login.")
	}
	sessionID := strings.TrimSpace(resp.GetSessionId())
	if sessionID == "" {
		return publicauthapp.PasskeyChallenge{}, apperrors.E(apperrors.KindUnknown, "Auth did not return a login session.")
	}
	return publicauthapp.PasskeyChallenge{SessionID: sessionID, PublicKey: json.RawMessage(resp.GetCredentialRequestOptionsJson())}, nil
}

// FinishPasskeyLogin centralizes this web behavior in one helper seam.
func (g GRPCGateway) FinishPasskeyLogin(ctx context.Context, sessionID string, credential json.RawMessage) (string, error) {
	resp, err := g.Client.FinishPasskeyLogin(ctx, &authv1.FinishPasskeyLoginRequest{
		SessionId:              sessionID,
		CredentialResponseJson: credential,
	})
	if err != nil {
		return "", mapGRPCError(err, apperrors.KindInvalidInput, "Failed to finish passkey login.")
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "Auth did not return a user ID.")
	}
	return userID, nil
}

// CreateWebSession executes package-scoped creation behavior for this flow.
func (g GRPCGateway) CreateWebSession(ctx context.Context, userID string) (string, error) {
	resp, err := g.Client.CreateWebSession(ctx, &authv1.CreateWebSessionRequest{UserId: userID})
	if err != nil {
		return "", mapGRPCError(err, apperrors.KindUnknown, "Failed to create a web session.")
	}
	sessionID := strings.TrimSpace(resp.GetSession().GetId())
	if sessionID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "Auth did not return a web session ID.")
	}
	return sessionID, nil
}

// HasValidWebSession reports whether this package condition is satisfied.
func (g GRPCGateway) HasValidWebSession(ctx context.Context, sessionID string) bool {
	resp, err := g.Client.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil || resp == nil || resp.GetSession() == nil {
		return false
	}
	return strings.TrimSpace(resp.GetSession().GetId()) != ""
}

// RevokeWebSession applies this package workflow transition.
func (g GRPCGateway) RevokeWebSession(ctx context.Context, sessionID string) error {
	_, err := g.Client.RevokeWebSession(ctx, &authv1.RevokeWebSessionRequest{SessionId: sessionID})
	return mapGRPCError(err, apperrors.KindUnknown, "Failed to revoke the web session.")
}
