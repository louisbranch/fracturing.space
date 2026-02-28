package domain

import (
	"encoding/json"
	"fmt"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
)

// signupCompletedEventPayload captures the durable fields consumed by worker handlers.
type signupCompletedEventPayload struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	SignupMethod string `json:"signup_method"`
}

// decodeSignupCompletedPayload centralizes payload parsing so all signup handlers
// enforce the same required fields and permanent-error semantics.
func decodeSignupCompletedPayload(event *authv1.IntegrationOutboxEvent) (signupCompletedEventPayload, error) {
	if event == nil {
		return signupCompletedEventPayload{}, fmt.Errorf("event is required")
	}

	var payload signupCompletedEventPayload
	if err := json.Unmarshal([]byte(event.GetPayloadJson()), &payload); err != nil {
		return signupCompletedEventPayload{}, fmt.Errorf("decode signup payload: %w", err)
	}
	payload.UserID = strings.TrimSpace(payload.UserID)
	if payload.UserID == "" {
		return signupCompletedEventPayload{}, fmt.Errorf("user_id is required in signup payload")
	}
	payload.Email = strings.TrimSpace(payload.Email)
	payload.SignupMethod = strings.TrimSpace(payload.SignupMethod)
	return payload, nil
}
