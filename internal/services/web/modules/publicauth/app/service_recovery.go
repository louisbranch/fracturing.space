package app

import (
	"context"
	"encoding/json"
	"strings"
)

// recoveryService owns account recovery ceremonies.
type recoveryService struct {
	recovery RecoveryGateway
}

// NewRecoveryService wires recovery-only public auth flows behind input validation.
func NewRecoveryService(gateway RecoveryGateway) RecoveryService {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return recoveryService{recovery: gateway}
}

// RecoveryStart verifies the recovery code, then starts replacement passkey enrollment.
func (s recoveryService) RecoveryStart(ctx context.Context, username string, recoveryCode string) (RecoveryChallenge, error) {
	resolvedUsername, err := requireUsername(username)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	resolvedRecoveryCode, err := requireRecoveryCode(recoveryCode)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	recoverySessionID, err := s.recovery.BeginAccountRecovery(ctx, resolvedUsername, resolvedRecoveryCode)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	resolvedRecoverySessionID, err := requireGatewaySessionID(recoverySessionID)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	challenge, err := s.recovery.BeginRecoveryPasskeyRegistration(ctx, resolvedRecoverySessionID)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	resolvedPasskeySessionID, err := requireGatewaySessionID(challenge.SessionID)
	if err != nil {
		return RecoveryChallenge{}, err
	}
	return RecoveryChallenge{
		RecoverySessionID: resolvedRecoverySessionID,
		SessionID:         resolvedPasskeySessionID,
		PublicKey:         challenge.PublicKey,
	}, nil
}

// RecoveryFinish completes replacement passkey enrollment and returns the signed-in session.
func (s recoveryService) RecoveryFinish(ctx context.Context, recoverySessionID string, sessionID string, credential json.RawMessage, pendingID string) (PasskeyFinish, error) {
	resolvedRecoverySessionID, err := requireSessionID(recoverySessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	resolvedSessionID, err := requireSessionID(sessionID)
	if err != nil {
		return PasskeyFinish{}, err
	}
	if err := requireCredential(credential); err != nil {
		return PasskeyFinish{}, err
	}
	finished, err := s.recovery.FinishRecoveryPasskeyRegistration(ctx, resolvedRecoverySessionID, resolvedSessionID, credential, strings.TrimSpace(pendingID))
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
	finished.RecoveryCode, err = requireRecoveryCode(finished.RecoveryCode)
	if err != nil {
		return PasskeyFinish{}, err
	}
	return finished, nil
}
