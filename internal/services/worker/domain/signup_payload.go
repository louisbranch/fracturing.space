package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// signupCompletedEventPayload captures the durable fields consumed by worker handlers.
type signupCompletedEventPayload struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	SignupMethod string `json:"signup_method"`
}

// decodeSignupCompletedPayload centralizes payload parsing so all signup handlers
// enforce the same required fields and permanent-error semantics.
func decodeSignupCompletedPayload(event OutboxEvent) (signupCompletedEventPayload, error) {
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
	payload.Username = strings.TrimSpace(payload.Username)
	payload.SignupMethod = strings.TrimSpace(payload.SignupMethod)
	return payload, nil
}
