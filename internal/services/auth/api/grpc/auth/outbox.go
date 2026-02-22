package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

const signupCompletedEventType = "auth.signup_completed"

type signupOutboxEnqueuer interface {
	EnqueueIntegrationOutboxEvent(ctx context.Context, event storage.IntegrationOutboxEvent) error
}

func signupCompletedDedupeKey(userID string) string {
	return "signup_completed:user:" + userID + ":v1"
}

func (s *AuthService) signupCompletedOutboxEvent(baseUser user.User, signupMethod string) (storage.IntegrationOutboxEvent, error) {
	now := time.Now().UTC()
	if s != nil && s.clock != nil {
		now = s.clock().UTC()
	}

	newID := id.NewID
	if s != nil && s.idGenerator != nil {
		newID = s.idGenerator
	}
	eventID, err := newID()
	if err != nil {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("generate signup completed event id: %w", err)
	}

	payload, err := json.Marshal(map[string]string{
		"user_id":        baseUser.ID,
		"email":          baseUser.Email,
		"signup_method":  signupMethod,
		"signup_at":      now.Format(time.RFC3339Nano),
		"notification_v": "v1",
	})
	if err != nil {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("marshal signup completed payload: %w", err)
	}

	return storage.IntegrationOutboxEvent{
		ID:            eventID,
		EventType:     signupCompletedEventType,
		PayloadJSON:   string(payload),
		DedupeKey:     signupCompletedDedupeKey(baseUser.ID),
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (s *AuthService) persistUserWithSignupCompletedOutbox(ctx context.Context, baseUser user.User, signupMethod string) error {
	if s == nil || s.store == nil {
		return nil
	}
	event, err := s.signupCompletedOutboxEvent(baseUser, signupMethod)
	if err != nil {
		return err
	}

	if txStore, ok := s.store.(storage.UserOutboxTransactionalStore); ok {
		return txStore.PutUserWithIntegrationOutboxEvent(ctx, baseUser, event)
	}

	if err := s.store.PutUser(ctx, baseUser); err != nil {
		return fmt.Errorf("put user: %w", err)
	}
	outboxStore, ok := s.store.(signupOutboxEnqueuer)
	if !ok {
		return nil
	}
	if err := outboxStore.EnqueueIntegrationOutboxEvent(ctx, event); err != nil {
		return fmt.Errorf("enqueue signup completed outbox: %w", err)
	}
	return nil
}

func (s *AuthService) enqueueSignupCompletedOutbox(ctx context.Context, baseUser user.User, signupMethod string) error {
	if s == nil || s.store == nil {
		return nil
	}
	outboxStore, ok := s.store.(signupOutboxEnqueuer)
	if !ok {
		return nil
	}
	event, err := s.signupCompletedOutboxEvent(baseUser, signupMethod)
	if err != nil {
		return err
	}
	return outboxStore.EnqueueIntegrationOutboxEvent(ctx, event)
}
