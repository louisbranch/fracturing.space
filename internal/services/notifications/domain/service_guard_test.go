package domain

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- NewService defaults ---

func TestNewService_NilClockDefaultsToTimeNow(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), nil, sequentialIDGenerator("id-1"))
	if svc.clock == nil {
		t.Fatal("expected clock to be set")
	}
	// Verify it returns approximately current time (within 1s).
	got := svc.clock()
	if time.Since(got) > time.Second {
		t.Fatalf("default clock returned stale time: %v", got)
	}
}

func TestNewService_NilIDGeneratorDefaultsToBuiltin(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), nil)
	if svc.newID == nil {
		t.Fatal("expected newID to be set")
	}
	id, err := svc.newID()
	if err != nil {
		t.Fatalf("default id generator: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty id from default generator")
	}
}

// --- CreateIntent guard paths ---

func TestCreateIntent_NilService(t *testing.T) {
	t.Parallel()
	var svc *Service
	_, err := svc.CreateIntent(context.Background(), CreateIntentInput{RecipientUserID: "u1", MessageType: "test"})
	if !errors.Is(err, ErrStoreNotConfigured) {
		t.Fatalf("err = %v, want %v", err, ErrStoreNotConfigured)
	}
}

func TestCreateIntent_NilIDGenerator(t *testing.T) {
	t.Parallel()
	svc := &Service{store: newFakeStore(), clock: fixedClock(time.Now())}
	_, err := svc.CreateIntent(context.Background(), CreateIntentInput{RecipientUserID: "u1", MessageType: "test"})
	if !errors.Is(err, ErrIDGeneratorNotConfigured) {
		t.Fatalf("err = %v, want %v", err, ErrIDGeneratorNotConfigured)
	}
}

func TestCreateIntent_EmptyRecipient(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), sequentialIDGenerator("id-1"))
	_, err := svc.CreateIntent(context.Background(), CreateIntentInput{RecipientUserID: "  ", MessageType: "test"})
	if !errors.Is(err, ErrRecipientUserIDRequired) {
		t.Fatalf("err = %v, want %v", err, ErrRecipientUserIDRequired)
	}
}

func TestCreateIntent_EmptyMessageType(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), sequentialIDGenerator("id-1"))
	_, err := svc.CreateIntent(context.Background(), CreateIntentInput{RecipientUserID: "u1", MessageType: ""})
	if !errors.Is(err, ErrMessageTypeRequired) {
		t.Fatalf("err = %v, want %v", err, ErrMessageTypeRequired)
	}
}

// --- ListInbox guard and boundary paths ---

func TestListInbox_NilService(t *testing.T) {
	t.Parallel()
	var svc *Service
	_, err := svc.ListInbox(context.Background(), ListInboxInput{RecipientUserID: "u1"})
	if !errors.Is(err, ErrStoreNotConfigured) {
		t.Fatalf("err = %v, want %v", err, ErrStoreNotConfigured)
	}
}

func TestListInbox_EmptyRecipient(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), sequentialIDGenerator())
	_, err := svc.ListInbox(context.Background(), ListInboxInput{RecipientUserID: ""})
	if !errors.Is(err, ErrRecipientUserIDRequired) {
		t.Fatalf("err = %v, want %v", err, ErrRecipientUserIDRequired)
	}
}

func TestListInbox_DefaultPageSize(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	svc := NewService(newFakeStore(), fixedClock(now), sequentialIDGenerator("id-1"))

	// Invariant: zero page size should use defaultPageSize (50), not fail.
	_, err := svc.ListInbox(context.Background(), ListInboxInput{RecipientUserID: "u1", PageSize: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListInbox_NegativePageSize(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	svc := NewService(newFakeStore(), fixedClock(now), sequentialIDGenerator("id-1"))

	// Invariant: negative page size should fall back to default, not fail.
	_, err := svc.ListInbox(context.Background(), ListInboxInput{RecipientUserID: "u1", PageSize: -5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListInbox_OverMaxPageSize(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	svc := NewService(newFakeStore(), fixedClock(now), sequentialIDGenerator("id-1"))

	// Invariant: page size > maxPageSize (200) should be clamped.
	_, err := svc.ListInbox(context.Background(), ListInboxInput{RecipientUserID: "u1", PageSize: 999})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetNotification guard paths ---

func TestGetNotification_NilService(t *testing.T) {
	t.Parallel()
	var svc *Service
	_, err := svc.GetNotification(context.Background(), GetNotificationInput{RecipientUserID: "u1", NotificationID: "n1"})
	if !errors.Is(err, ErrStoreNotConfigured) {
		t.Fatalf("err = %v, want %v", err, ErrStoreNotConfigured)
	}
}

func TestGetNotification_EmptyRecipient(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), sequentialIDGenerator())
	_, err := svc.GetNotification(context.Background(), GetNotificationInput{RecipientUserID: "", NotificationID: "n1"})
	if !errors.Is(err, ErrRecipientUserIDRequired) {
		t.Fatalf("err = %v, want %v", err, ErrRecipientUserIDRequired)
	}
}

func TestGetNotification_EmptyNotificationID(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), sequentialIDGenerator())
	_, err := svc.GetNotification(context.Background(), GetNotificationInput{RecipientUserID: "u1", NotificationID: ""})
	if !errors.Is(err, ErrNotificationIDRequired) {
		t.Fatalf("err = %v, want %v", err, ErrNotificationIDRequired)
	}
}

// --- GetUnreadStatus guard paths ---

func TestGetUnreadStatus_NilService(t *testing.T) {
	t.Parallel()
	var svc *Service
	_, err := svc.GetUnreadStatus(context.Background(), GetUnreadStatusInput{RecipientUserID: "u1"})
	if !errors.Is(err, ErrStoreNotConfigured) {
		t.Fatalf("err = %v, want %v", err, ErrStoreNotConfigured)
	}
}

func TestGetUnreadStatus_EmptyRecipient(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), sequentialIDGenerator())
	_, err := svc.GetUnreadStatus(context.Background(), GetUnreadStatusInput{RecipientUserID: ""})
	if !errors.Is(err, ErrRecipientUserIDRequired) {
		t.Fatalf("err = %v, want %v", err, ErrRecipientUserIDRequired)
	}
}

func TestGetUnreadStatus_ZeroUnread(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), sequentialIDGenerator())
	status, err := svc.GetUnreadStatus(context.Background(), GetUnreadStatusInput{RecipientUserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.HasUnread {
		t.Fatal("expected HasUnread=false for empty inbox")
	}
	if status.UnreadCount != 0 {
		t.Fatalf("UnreadCount = %d, want 0", status.UnreadCount)
	}
}

// --- MarkRead guard paths ---

func TestMarkRead_NilService(t *testing.T) {
	t.Parallel()
	var svc *Service
	_, err := svc.MarkRead(context.Background(), MarkReadInput{RecipientUserID: "u1", NotificationID: "n1"})
	if !errors.Is(err, ErrStoreNotConfigured) {
		t.Fatalf("err = %v, want %v", err, ErrStoreNotConfigured)
	}
}

func TestMarkRead_EmptyRecipient(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), sequentialIDGenerator())
	_, err := svc.MarkRead(context.Background(), MarkReadInput{RecipientUserID: "", NotificationID: "n1"})
	if !errors.Is(err, ErrRecipientUserIDRequired) {
		t.Fatalf("err = %v, want %v", err, ErrRecipientUserIDRequired)
	}
}

func TestMarkRead_EmptyNotificationID(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeStore(), fixedClock(time.Now()), sequentialIDGenerator())
	_, err := svc.MarkRead(context.Background(), MarkReadInput{RecipientUserID: "u1", NotificationID: ""})
	if !errors.Is(err, ErrNotificationIDRequired) {
		t.Fatalf("err = %v, want %v", err, ErrNotificationIDRequired)
	}
}

// --- nowUTC nil clock fallback ---

func TestNowUTC_NilClockReturnsCurrentTime(t *testing.T) {
	t.Parallel()
	svc := &Service{store: newFakeStore()}
	got := svc.nowUTC()
	if time.Since(got) > time.Second {
		t.Fatalf("nowUTC returned stale time: %v", got)
	}
}
