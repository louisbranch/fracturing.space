package storage

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound indicates a requested notification or delivery record is missing.
	ErrNotFound = errors.New("record not found")
	// ErrConflict indicates a requested write conflicts with uniqueness constraints.
	ErrConflict = errors.New("record conflict")
)

// DeliveryChannel identifies one notification channel type.
type DeliveryChannel string

const (
	// DeliveryChannelInApp represents inbox/internal delivery.
	DeliveryChannelInApp DeliveryChannel = "in_app"
	// DeliveryChannelEmail represents email delivery.
	DeliveryChannelEmail DeliveryChannel = "email"
)

// DeliveryStatus identifies one delivery lifecycle state.
type DeliveryStatus string

const (
	// DeliveryStatusPending means the delivery is queued for processing.
	DeliveryStatusPending DeliveryStatus = "pending"
	// DeliveryStatusFailed means the delivery attempt failed and can be retried.
	DeliveryStatusFailed DeliveryStatus = "failed"
	// DeliveryStatusDelivered means the channel delivery was completed.
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	// DeliveryStatusSkipped means the channel was intentionally skipped.
	DeliveryStatusSkipped DeliveryStatus = "skipped"
)

// NotificationRecord stores one user notification inbox item.
type NotificationRecord struct {
	ID              string
	RecipientUserID string
	MessageType     string
	PayloadJSON     string
	DedupeKey       string
	Source          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ReadAt          *time.Time
}

// NotificationPage stores a paged inbox listing result.
type NotificationPage struct {
	Notifications []NotificationRecord
	NextPageToken string
}

// DeliveryRecord stores one channel-delivery attempt state.
type DeliveryRecord struct {
	NotificationID string
	Channel        DeliveryChannel
	Status         DeliveryStatus
	AttemptCount   int
	NextAttemptAt  time.Time
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeliveredAt    *time.Time
}

// NotificationStore persists notification inbox state.
type NotificationStore interface {
	PutNotification(ctx context.Context, record NotificationRecord) error
	GetNotificationByRecipientAndDedupeKey(ctx context.Context, recipientUserID string, dedupeKey string) (NotificationRecord, error)
	ListNotificationsByRecipient(ctx context.Context, recipientUserID string, pageSize int, pageToken string) (NotificationPage, error)
	CountUnreadNotificationsByRecipient(ctx context.Context, recipientUserID string) (int, error)
	MarkNotificationRead(ctx context.Context, recipientUserID string, notificationID string, readAt time.Time) (NotificationRecord, error)
}

// DeliveryStore persists channel delivery attempt state.
type DeliveryStore interface {
	PutDelivery(ctx context.Context, record DeliveryRecord) error
	ListPendingDeliveries(ctx context.Context, channel DeliveryChannel, limit int, now time.Time) ([]DeliveryRecord, error)
	MarkDeliveryRetry(ctx context.Context, notificationID string, channel DeliveryChannel, attemptCount int, nextAttemptAt time.Time, lastError string) error
	MarkDeliverySucceeded(ctx context.Context, notificationID string, channel DeliveryChannel, deliveredAt time.Time) error
}

// NotificationBootstrapStore atomically persists a notification with initial channel deliveries.
type NotificationBootstrapStore interface {
	PutNotificationWithDeliveries(ctx context.Context, notification NotificationRecord, deliveries []DeliveryRecord) error
}
