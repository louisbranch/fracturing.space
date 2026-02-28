package domain

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

var (
	// ErrNotFound indicates a notification record was not found.
	ErrNotFound = errors.New("notification not found")
	// ErrConflict indicates a write conflicted with existing uniqueness constraints.
	ErrConflict = errors.New("notification conflict")
	// ErrStoreNotConfigured indicates the service is missing persistence wiring.
	ErrStoreNotConfigured = errors.New("notification store is not configured")
	// ErrRecipientUserIDRequired indicates recipient identity is required.
	ErrRecipientUserIDRequired = errors.New("recipient user id is required")
	// ErrMessageTypeRequired indicates a message type is required.
	ErrMessageTypeRequired = errors.New("notification message type is required")
	// ErrNotificationIDRequired indicates notification ID is required.
	ErrNotificationIDRequired = errors.New("notification id is required")
	// ErrIDGeneratorNotConfigured indicates an ID generator is required.
	ErrIDGeneratorNotConfigured = errors.New("notification id generator is not configured")
	// ErrIDGeneratorExhausted indicates a fixed test ID sequence was exhausted.
	ErrIDGeneratorExhausted = errors.New("notification id generator exhausted")
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// Notification captures one user-targeted notification item.
type Notification struct {
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

// NotificationPage is a paged recipient inbox view.
type NotificationPage struct {
	Notifications []Notification
	NextPageToken string
}

// UnreadStatus captures recipient unread-inbox status.
type UnreadStatus struct {
	HasUnread   bool
	UnreadCount int
}

// CreateIntentInput describes one producer notification request.
type CreateIntentInput struct {
	RecipientUserID string
	MessageType     string
	PayloadJSON     string
	DedupeKey       string
	Source          string
}

// ListInboxInput configures recipient inbox listing.
type ListInboxInput struct {
	RecipientUserID string
	PageSize        int
	PageToken       string
}

// GetUnreadStatusInput identifies a recipient unread-status lookup.
type GetUnreadStatusInput struct {
	RecipientUserID string
}

// MarkReadInput identifies one recipient notification to acknowledge.
type MarkReadInput struct {
	RecipientUserID string
	NotificationID  string
}

// GetNotificationInput identifies one recipient notification lookup.
type GetNotificationInput struct {
	RecipientUserID string
	NotificationID  string
}

// Store is the domain persistence boundary for notification lifecycle behavior.
type Store interface {
	GetNotificationByRecipientAndDedupeKey(ctx context.Context, recipientUserID string, dedupeKey string) (Notification, error)
	GetNotificationByRecipientAndID(ctx context.Context, recipientUserID string, notificationID string) (Notification, error)
	PutNotification(ctx context.Context, notification Notification) error
	ListNotificationsByRecipient(ctx context.Context, recipientUserID string, pageSize int, pageToken string) (NotificationPage, error)
	CountUnreadNotificationsByRecipient(ctx context.Context, recipientUserID string) (int, error)
	MarkNotificationRead(ctx context.Context, recipientUserID string, notificationID string, readAt time.Time) (Notification, error)
}

// Service orchestrates recipient inbox lifecycle behavior.
type Service struct {
	store Store
	clock func() time.Time
	newID func() (string, error)
}

// NewService constructs notification domain use-cases.
func NewService(store Store, clock func() time.Time, newID func() (string, error)) *Service {
	if clock == nil {
		clock = time.Now
	}
	if newID == nil {
		newID = id.NewID
	}
	return &Service{
		store: store,
		clock: clock,
		newID: newID,
	}
}

// CreateIntent stores one notification item and de-duplicates by recipient+dedupe key.
func (s *Service) CreateIntent(ctx context.Context, input CreateIntentInput) (Notification, error) {
	if s == nil || s.store == nil {
		return Notification{}, ErrStoreNotConfigured
	}
	if s.newID == nil {
		return Notification{}, ErrIDGeneratorNotConfigured
	}
	recipientUserID := strings.TrimSpace(input.RecipientUserID)
	if recipientUserID == "" {
		return Notification{}, ErrRecipientUserIDRequired
	}
	messageType := NormalizeMessageType(input.MessageType)
	if messageType == "" {
		return Notification{}, ErrMessageTypeRequired
	}
	dedupeKey := strings.TrimSpace(input.DedupeKey)
	if dedupeKey != "" {
		existing, err := s.store.GetNotificationByRecipientAndDedupeKey(ctx, recipientUserID, dedupeKey)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return Notification{}, err
		}
	}

	notificationID, err := s.newID()
	if err != nil {
		return Notification{}, err
	}
	now := s.nowUTC()
	notification := Notification{
		ID:              notificationID,
		RecipientUserID: recipientUserID,
		MessageType:     messageType,
		PayloadJSON:     strings.TrimSpace(input.PayloadJSON),
		DedupeKey:       dedupeKey,
		Source:          strings.TrimSpace(input.Source),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.store.PutNotification(ctx, notification); err != nil {
		if dedupeKey != "" && errors.Is(err, ErrConflict) {
			existing, lookupErr := s.store.GetNotificationByRecipientAndDedupeKey(ctx, recipientUserID, dedupeKey)
			if lookupErr == nil {
				return existing, nil
			}
			if errors.Is(lookupErr, ErrNotFound) {
				return Notification{}, err
			}
			return Notification{}, lookupErr
		}
		return Notification{}, err
	}
	return notification, nil
}

// ListInbox lists recipient inbox notifications newest first.
func (s *Service) ListInbox(ctx context.Context, input ListInboxInput) (NotificationPage, error) {
	if s == nil || s.store == nil {
		return NotificationPage{}, ErrStoreNotConfigured
	}
	recipientUserID := strings.TrimSpace(input.RecipientUserID)
	if recipientUserID == "" {
		return NotificationPage{}, ErrRecipientUserIDRequired
	}
	pageSize := input.PageSize
	switch {
	case pageSize <= 0:
		pageSize = defaultPageSize
	case pageSize > maxPageSize:
		pageSize = maxPageSize
	}
	return s.store.ListNotificationsByRecipient(ctx, recipientUserID, pageSize, strings.TrimSpace(input.PageToken))
}

// GetNotification returns one recipient notification visible in the inbox.
func (s *Service) GetNotification(ctx context.Context, input GetNotificationInput) (Notification, error) {
	if s == nil || s.store == nil {
		return Notification{}, ErrStoreNotConfigured
	}
	recipientUserID := strings.TrimSpace(input.RecipientUserID)
	if recipientUserID == "" {
		return Notification{}, ErrRecipientUserIDRequired
	}
	notificationID := strings.TrimSpace(input.NotificationID)
	if notificationID == "" {
		return Notification{}, ErrNotificationIDRequired
	}
	return s.store.GetNotificationByRecipientAndID(ctx, recipientUserID, notificationID)
}

// GetUnreadStatus returns unread-inbox status for one recipient.
func (s *Service) GetUnreadStatus(ctx context.Context, input GetUnreadStatusInput) (UnreadStatus, error) {
	if s == nil || s.store == nil {
		return UnreadStatus{}, ErrStoreNotConfigured
	}
	recipientUserID := strings.TrimSpace(input.RecipientUserID)
	if recipientUserID == "" {
		return UnreadStatus{}, ErrRecipientUserIDRequired
	}
	unreadCount, err := s.store.CountUnreadNotificationsByRecipient(ctx, recipientUserID)
	if err != nil {
		return UnreadStatus{}, err
	}
	if unreadCount < 0 {
		unreadCount = 0
	}
	return UnreadStatus{
		HasUnread:   unreadCount > 0,
		UnreadCount: unreadCount,
	}, nil
}

// MarkRead marks one recipient notification as read.
func (s *Service) MarkRead(ctx context.Context, input MarkReadInput) (Notification, error) {
	if s == nil || s.store == nil {
		return Notification{}, ErrStoreNotConfigured
	}
	recipientUserID := strings.TrimSpace(input.RecipientUserID)
	if recipientUserID == "" {
		return Notification{}, ErrRecipientUserIDRequired
	}
	notificationID := strings.TrimSpace(input.NotificationID)
	if notificationID == "" {
		return Notification{}, ErrNotificationIDRequired
	}
	return s.store.MarkNotificationRead(ctx, recipientUserID, notificationID, s.nowUTC())
}

func (s *Service) nowUTC() time.Time {
	if s.clock == nil {
		return time.Now().UTC()
	}
	return s.clock().UTC()
}
