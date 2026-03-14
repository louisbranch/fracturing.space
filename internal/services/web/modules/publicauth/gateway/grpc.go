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
	CheckUsernameAvailability(context.Context, *authv1.CheckUsernameAvailabilityRequest, ...grpc.CallOption) (*authv1.CheckUsernameAvailabilityResponse, error)
	FinishAccountRegistration(context.Context, *authv1.FinishAccountRegistrationRequest, ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error)
	BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error)
	FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error)
	BeginAccountRecovery(context.Context, *authv1.BeginAccountRecoveryRequest, ...grpc.CallOption) (*authv1.BeginAccountRecoveryResponse, error)
	BeginRecoveryPasskeyRegistration(context.Context, *authv1.BeginRecoveryPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error)
	FinishRecoveryPasskeyRegistration(context.Context, *authv1.FinishRecoveryPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error)
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

// CheckUsernameAvailability returns advisory signup validation state.
func (g GRPCGateway) CheckUsernameAvailability(ctx context.Context, username string) (publicauthapp.UsernameAvailability, error) {
	resp, err := g.Client.CheckUsernameAvailability(ctx, &authv1.CheckUsernameAvailabilityRequest{Username: username})
	if err != nil {
		return publicauthapp.UsernameAvailability{}, mapGRPCError(err, apperrors.KindUnavailable, "Unable to validate the username.")
	}
	return publicauthapp.UsernameAvailability{
		CanonicalUsername: strings.TrimSpace(resp.GetCanonicalUsername()),
		State:             mapUsernameAvailabilityState(resp.GetState()),
	}, nil
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
func (g GRPCGateway) FinishPasskeyLogin(ctx context.Context, sessionID string, credential json.RawMessage, pendingID string) (string, error) {
	resp, err := g.Client.FinishPasskeyLogin(ctx, &authv1.FinishPasskeyLoginRequest{
		SessionId:              sessionID,
		CredentialResponseJson: credential,
		PendingId:              strings.TrimSpace(pendingID),
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

// BeginAccountRecovery verifies a recovery code and returns a recovery session ID.
func (g GRPCGateway) BeginAccountRecovery(ctx context.Context, username string, recoveryCode string) (string, error) {
	resp, err := g.Client.BeginAccountRecovery(ctx, &authv1.BeginAccountRecoveryRequest{
		Username:     username,
		RecoveryCode: recoveryCode,
	})
	if err != nil {
		return "", mapGRPCError(err, apperrors.KindInvalidInput, "Failed to start account recovery.")
	}
	recoverySessionID := strings.TrimSpace(resp.GetRecoverySessionId())
	if recoverySessionID == "" {
		return "", apperrors.E(apperrors.KindUnknown, "Auth did not return a recovery session ID.")
	}
	return recoverySessionID, nil
}

// BeginRecoveryPasskeyRegistration starts replacement passkey enrollment.
func (g GRPCGateway) BeginRecoveryPasskeyRegistration(ctx context.Context, recoverySessionID string) (publicauthapp.PasskeyChallenge, error) {
	resp, err := g.Client.BeginRecoveryPasskeyRegistration(ctx, &authv1.BeginRecoveryPasskeyRegistrationRequest{
		RecoverySessionId: recoverySessionID,
	})
	if err != nil {
		return publicauthapp.PasskeyChallenge{}, mapGRPCError(err, apperrors.KindInvalidInput, "Failed to begin recovery passkey registration.")
	}
	sessionID := strings.TrimSpace(resp.GetSessionId())
	if sessionID == "" {
		return publicauthapp.PasskeyChallenge{}, apperrors.E(apperrors.KindUnknown, "Auth did not return a recovery passkey session.")
	}
	return publicauthapp.PasskeyChallenge{SessionID: sessionID, PublicKey: json.RawMessage(resp.GetCredentialCreationOptionsJson())}, nil
}

// FinishRecoveryPasskeyRegistration completes recovery and returns the signed-in session.
func (g GRPCGateway) FinishRecoveryPasskeyRegistration(ctx context.Context, recoverySessionID string, sessionID string, credential json.RawMessage, pendingID string) (publicauthapp.PasskeyFinish, error) {
	resp, err := g.Client.FinishRecoveryPasskeyRegistration(ctx, &authv1.FinishRecoveryPasskeyRegistrationRequest{
		RecoverySessionId:      recoverySessionID,
		SessionId:              sessionID,
		CredentialResponseJson: credential,
		PendingId:              strings.TrimSpace(pendingID),
	})
	if err != nil {
		return publicauthapp.PasskeyFinish{}, mapGRPCError(err, apperrors.KindInvalidInput, "Failed to finish account recovery.")
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

// mapUsernameAvailabilityState maps auth transport enum values to web app state.
func mapUsernameAvailabilityState(state authv1.UsernameAvailabilityState) publicauthapp.UsernameAvailabilityState {
	switch state {
	case authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_AVAILABLE:
		return publicauthapp.UsernameAvailabilityStateAvailable
	case authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_UNAVAILABLE:
		return publicauthapp.UsernameAvailabilityStateUnavailable
	default:
		return publicauthapp.UsernameAvailabilityStateInvalid
	}
}
