package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

type signupOutboxStore interface {
	EnqueueIntegrationOutboxEvent(ctx context.Context, event storage.IntegrationOutboxEvent) error
}

func signupCompletedEvent(clock func() time.Time, idGenerator func() (string, error), created user.User, signupMethod string) (storage.IntegrationOutboxEvent, error) {
	now := time.Now().UTC()
	if clock != nil {
		now = clock().UTC()
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	eventID, err := idGenerator()
	if err != nil {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("generate signup completed event id: %w", err)
	}
	payload, err := json.Marshal(map[string]string{
		"user_id":        created.ID,
		"email":          created.Email,
		"signup_method":  signupMethod,
		"signup_at":      now.Format(time.RFC3339Nano),
		"notification_v": "v1",
	})
	if err != nil {
		return storage.IntegrationOutboxEvent{}, fmt.Errorf("marshal signup completed payload: %w", err)
	}

	return storage.IntegrationOutboxEvent{
		ID:            eventID,
		EventType:     "auth.signup_completed",
		PayloadJSON:   string(payload),
		DedupeKey:     "signup_completed:user:" + created.ID + ":v1",
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func persistUserWithSignupCompleted(ctx context.Context, userStore UserStore, clock func() time.Time, idGenerator func() (string, error), created user.User, signupMethod string) error {
	if userStore == nil {
		return nil
	}

	event, err := signupCompletedEvent(clock, idGenerator, created, signupMethod)
	if err != nil {
		return err
	}

	if txStore, ok := userStore.(storage.UserOutboxTransactionalStore); ok {
		return txStore.PutUserWithIntegrationOutboxEvent(ctx, created, event)
	}

	if err := userStore.PutUser(ctx, created); err != nil {
		return err
	}

	outboxStore, ok := userStore.(signupOutboxStore)
	if !ok {
		return nil
	}
	return outboxStore.EnqueueIntegrationOutboxEvent(ctx, event)
}
