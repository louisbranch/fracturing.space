package app

import (
	"context"
	"encoding/json"
	"strings"
)

// passkeyService owns username, login, and signup ceremonies.
type passkeyService struct {
	passkeys PasskeyGateway
}

// NewPasskeyService wires passkey-only public auth flows behind input validation.
func NewPasskeyService(gateway PasskeyGateway) PasskeyService {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return passkeyService{passkeys: gateway}
}

// CheckUsernameAvailability returns advisory live validation state for signup.
func (s passkeyService) CheckUsernameAvailability(ctx context.Context, username string) (UsernameAvailability, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return UsernameAvailability{State: UsernameAvailabilityStateInvalid}, nil
	}
	return s.passkeys.CheckUsernameAvailability(ctx, username)
}

// PasskeyLoginStart normalizes the username before asking auth to begin login.
func (s passkeyService) PasskeyLoginStart(ctx context.Context, username string) (PasskeyChallenge, error) {
	resolvedUsername, err := requireUsername(username)
	if err != nil {
		return PasskeyChallenge{}, err
	}
	return s.passkeys.BeginPasskeyLogin(ctx, resolvedUsername)
}

// PasskeyLoginFinish validates the ceremony response, then creates a web session.
func (s passkeyService) PasskeyLoginFinish(ctx context.Context, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error) {
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	if err := requireCredential(credential); err != nil {
		return PasskeyFinish{}, err
	}
	userID, err := s.passkeys.FinishPasskeyLogin(ctx, resolvedSessionID, credential, strings.TrimSpace(pendingID))
	if err != nil {
		return PasskeyFinish{}, err
	}
	resolvedUserID, err := requireGatewayUserID(userID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	webSessionID, err := s.passkeys.CreateWebSession(ctx, resolvedUserID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	resolvedWebSessionID, err := requireGatewaySessionID(webSessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	return PasskeyFinish{SessionID: resolvedWebSessionID, UserID: resolvedUserID}, nil
}

// PasskeyRegisterStart validates the requested username before starting signup.
func (s passkeyService) PasskeyRegisterStart(ctx context.Context, username string) (PasskeyRegisterResult, error) {
	resolvedUsername, err := requireUsername(username)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	challenge, err := s.passkeys.BeginAccountRegistration(ctx, resolvedUsername)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	resolvedChallengeSessionID, err := requireGatewaySessionID(challenge.SessionID)
	if err != nil {
		return PasskeyRegisterResult{}, err
	}
	return PasskeyRegisterResult{SessionID: resolvedChallengeSessionID, PublicKey: challenge.PublicKey}, nil
}

// PasskeyRegisterFinish stages signup and normalizes the recovery-code reveal.
func (s passkeyService) PasskeyRegisterFinish(ctx context.Context, sessionID string, credential json.RawMessage) (PasskeyRegistrationReveal, error) {
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyRegistrationReveal{}, err
	}
	if err := requireCredential(credential); err != nil {
		return PasskeyRegistrationReveal{}, err
	}
	finished, err := s.passkeys.FinishAccountRegistration(ctx, resolvedSessionID, credential)
	if err != nil {
		return PasskeyRegistrationReveal{}, err
	}
	finished.RecoveryCode, err = requireRecoveryCode(finished.RecoveryCode)
	if err != nil {
		return PasskeyRegistrationReveal{}, err
	}
	return finished, nil
}

// PasskeyRegisterAcknowledge activates one staged signup and returns the signed-in session.
func (s passkeyService) PasskeyRegisterAcknowledge(ctx context.Context, sessionID string, pendingID string) (PasskeyFinish, error) {
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished, err := s.passkeys.AcknowledgeAccountRegistration(ctx, resolvedSessionID, strings.TrimSpace(pendingID))
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished.UserID, err = requireGatewayUserID(finished.UserID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	finished.SessionID, err = requireGatewaySessionID(finished.SessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	return finished, nil
}
