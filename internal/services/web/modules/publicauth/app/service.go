package app

import (
	"context"
	"encoding/json"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type service struct {
	auth Gateway
}

// NewService constructs a publicauth service with fail-closed gateway defaults.
func NewService(gateway Gateway) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{auth: gateway}
}

func (service) HealthBody() string {
	return "ok"
}

func (s service) PasskeyLoginStart(ctx context.Context) (PasskeyChallenge, error) {
	return s.auth.BeginPasskeyLogin(ctx)
}

func (s service) PasskeyLoginFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyFinish, error) {
	if strings.TrimSpace(sessionID) == "" {
		return PasskeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "session_id is required")
	}
	if len(credential) == 0 {
		return PasskeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "credential is required")
	}
	userID, err := s.auth.FinishPasskeyLogin(ctx, sessionID, credential)
	if err != nil {
		return PasskeyFinish{}, err
	}
	webSessionID, err := s.auth.CreateWebSession(ctx, userID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	return PasskeyFinish{SessionID: webSessionID, UserID: userID}, nil
}

func (s service) PasskeyRegisterStart(ctx context.Context, email string) (PasskeyRegisterResult, error) {
	if strings.TrimSpace(email) == "" {
		return PasskeyRegisterResult{}, apperrors.E(apperrors.KindInvalidInput, "email is required")
	}
	userID, err := s.auth.CreateUser(ctx, strings.TrimSpace(email))
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	challenge, err := s.auth.BeginPasskeyRegistration(ctx, userID)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	return PasskeyRegisterResult{SessionID: challenge.SessionID, UserID: userID, PublicKey: challenge.PublicKey}, nil
}

func (s service) PasskeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyFinish, error) {
	if strings.TrimSpace(sessionID) == "" {
		return PasskeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "session_id is required")
	}
	if len(credential) == 0 {
		return PasskeyFinish{}, apperrors.E(apperrors.KindInvalidInput, "credential is required")
	}
	userID, err := s.auth.FinishPasskeyRegistration(ctx, sessionID, credential)
	if err != nil {
		return PasskeyFinish{}, err
	}
	return PasskeyFinish{UserID: userID}, nil
}

func (s service) HasValidWebSession(ctx context.Context, sessionID string) bool {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false
	}
	return s.auth.HasValidWebSession(ctx, sessionID)
}

func (s service) RevokeWebSession(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	return s.auth.RevokeWebSession(ctx, sessionID)
}
