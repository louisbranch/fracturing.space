package domain

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCreateIntent_IdempotentByDedupeKey(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 20, 25, 0, 0, time.UTC)
	store := newFakeStore()
	svc := NewService(store, fixedClock(now), sequentialIDGenerator("notif-1", "notif-2"))

	first, err := svc.CreateIntent(context.Background(), CreateIntentInput{
		RecipientUserID: "user-1",
		Topic:           "campaign.invite",
		PayloadJSON:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "campaign.invite:inv-1",
		Source:          "game",
	})
	if err != nil {
		t.Fatalf("create first intent: %v", err)
	}

	second, err := svc.CreateIntent(context.Background(), CreateIntentInput{
		RecipientUserID: "user-1",
		Topic:           "campaign.invite",
		PayloadJSON:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "campaign.invite:inv-1",
		Source:          "game",
	})
	if err != nil {
		t.Fatalf("create second intent: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("expected dedupe create to return existing notification id %q, got %q", first.ID, second.ID)
	}
	if got := store.notificationCount(); got != 1 {
		t.Fatalf("expected one persisted notification, got %d", got)
	}
}

func TestListInbox_FiltersRecipientAndPaginatesNewestFirst(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 2, 21, 20, 30, 0, 0, time.UTC)
	store := newFakeStore()
	svc := NewService(store, fixedClock(base), sequentialIDGenerator("notif-1", "notif-2", "notif-3", "notif-4"))

	createAt := func(at time.Time, recipient string, dedupe string) {
		t.Helper()
		svc.clock = fixedClock(at)
		if _, err := svc.CreateIntent(context.Background(), CreateIntentInput{
			RecipientUserID: recipient,
			Topic:           "session.update",
			PayloadJSON:     `{"status":"active"}`,
			DedupeKey:       dedupe,
			Source:          "game",
		}); err != nil {
			t.Fatalf("create intent at %s: %v", at, err)
		}
	}

	createAt(base.Add(1*time.Minute), "user-1", "a")
	createAt(base.Add(2*time.Minute), "user-2", "x")
	createAt(base.Add(3*time.Minute), "user-1", "b")
	createAt(base.Add(4*time.Minute), "user-1", "c")

	pageOne, err := svc.ListInbox(context.Background(), ListInboxInput{
		RecipientUserID: "user-1",
		PageSize:        2,
	})
	if err != nil {
		t.Fatalf("list page one: %v", err)
	}
	if got := len(pageOne.Notifications); got != 2 {
		t.Fatalf("page one notifications = %d, want 2", got)
	}
	if pageOne.Notifications[0].DedupeKey != "c" || pageOne.Notifications[1].DedupeKey != "b" {
		t.Fatalf("unexpected page one order: %+v", pageOne.Notifications)
	}
	if pageOne.NextPageToken == "" {
		t.Fatal("expected non-empty next page token")
	}

	pageTwo, err := svc.ListInbox(context.Background(), ListInboxInput{
		RecipientUserID: "user-1",
		PageSize:        2,
		PageToken:       pageOne.NextPageToken,
	})
	if err != nil {
		t.Fatalf("list page two: %v", err)
	}
	if got := len(pageTwo.Notifications); got != 1 {
		t.Fatalf("page two notifications = %d, want 1", got)
	}
	if pageTwo.Notifications[0].DedupeKey != "a" {
		t.Fatalf("unexpected page two notification dedupe key: %q", pageTwo.Notifications[0].DedupeKey)
	}
}

func TestMarkRead_PersistsReadTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 20, 45, 0, 0, time.UTC)
	store := newFakeStore()
	svc := NewService(store, fixedClock(now), sequentialIDGenerator("notif-1"))

	created, err := svc.CreateIntent(context.Background(), CreateIntentInput{
		RecipientUserID: "user-1",
		Topic:           "campaign.invite",
		PayloadJSON:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "campaign.invite:inv-1",
		Source:          "game",
	})
	if err != nil {
		t.Fatalf("create intent: %v", err)
	}

	readAt := now.Add(5 * time.Minute)
	svc.clock = fixedClock(readAt)
	read, err := svc.MarkRead(context.Background(), MarkReadInput{
		RecipientUserID: "user-1",
		NotificationID:  created.ID,
	})
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if read.ReadAt == nil {
		t.Fatal("expected read_at to be set")
	}
	if !read.ReadAt.Equal(readAt) {
		t.Fatalf("read_at = %v, want %v", read.ReadAt, readAt)
	}
}

func TestGetUnreadStatus_ReportsUnreadCount(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 20, 47, 0, 0, time.UTC)
	store := newFakeStore()
	svc := NewService(store, fixedClock(now), sequentialIDGenerator("notif-1", "notif-2"))

	first, err := svc.CreateIntent(context.Background(), CreateIntentInput{
		RecipientUserID: "user-1",
		Topic:           "campaign.invite",
		DedupeKey:       "invite:1",
	})
	if err != nil {
		t.Fatalf("create first intent: %v", err)
	}
	if _, err := svc.CreateIntent(context.Background(), CreateIntentInput{
		RecipientUserID: "user-1",
		Topic:           "session.update",
		DedupeKey:       "session:1",
	}); err != nil {
		t.Fatalf("create second intent: %v", err)
	}
	if _, err := svc.MarkRead(context.Background(), MarkReadInput{
		RecipientUserID: "user-1",
		NotificationID:  first.ID,
	}); err != nil {
		t.Fatalf("mark first read: %v", err)
	}

	status, err := svc.GetUnreadStatus(context.Background(), GetUnreadStatusInput{
		RecipientUserID: "user-1",
	})
	if err != nil {
		t.Fatalf("get unread status: %v", err)
	}
	if !status.HasUnread {
		t.Fatalf("HasUnread = false, want true")
	}
	if status.UnreadCount != 1 {
		t.Fatalf("UnreadCount = %d, want 1", status.UnreadCount)
	}
}

func TestCreateIntent_ConcurrentDedupeReturnsSingleNotification(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 20, 50, 0, 0, time.UTC)
	store := newConcurrentConflictStore()
	svc := NewService(store, fixedClock(now), lockedSequentialIDGenerator("notif-1", "notif-2"))

	type createResult struct {
		notification Notification
		err          error
	}
	results := make(chan createResult, 2)
	input := CreateIntentInput{
		RecipientUserID: "user-1",
		Topic:           "campaign.invite",
		PayloadJSON:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "campaign.invite:inv-1",
		Source:          "game",
	}

	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			notification, err := svc.CreateIntent(context.Background(), input)
			results <- createResult{notification: notification, err: err}
		}()
	}
	wg.Wait()
	close(results)

	var ids []string
	for result := range results {
		if result.err != nil {
			t.Fatalf("expected idempotent create under race, got error: %v", result.err)
		}
		ids = append(ids, result.notification.ID)
	}
	if len(ids) != 2 {
		t.Fatalf("results = %d, want 2", len(ids))
	}
	if ids[0] != ids[1] {
		t.Fatalf("expected same notification id from concurrent dedupe calls, got %q and %q", ids[0], ids[1])
	}
	if got := store.notificationCount(); got != 1 {
		t.Fatalf("expected one persisted notification, got %d", got)
	}
}

func fixedClock(at time.Time) func() time.Time {
	return func() time.Time { return at }
}

func sequentialIDGenerator(ids ...string) func() (string, error) {
	queue := append([]string(nil), ids...)
	index := 0
	return func() (string, error) {
		if index >= len(queue) {
			return "", ErrIDGeneratorExhausted
		}
		value := queue[index]
		index++
		return value, nil
	}
}

func lockedSequentialIDGenerator(ids ...string) func() (string, error) {
	queue := append([]string(nil), ids...)
	index := 0
	var mu sync.Mutex
	return func() (string, error) {
		mu.Lock()
		defer mu.Unlock()
		if index >= len(queue) {
			return "", ErrIDGeneratorExhausted
		}
		value := queue[index]
		index++
		return value, nil
	}
}

type fakeStore struct {
	notifications map[string]Notification
	dedupeIndex   map[string]string
}

type concurrentConflictStore struct {
	mu            sync.Mutex
	releaseGets   chan struct{}
	getCalls      int
	notifications map[string]Notification
	dedupeIndex   map[string]string
}

func newConcurrentConflictStore() *concurrentConflictStore {
	return &concurrentConflictStore{
		releaseGets:   make(chan struct{}),
		notifications: make(map[string]Notification),
		dedupeIndex:   make(map[string]string),
	}
}

func (s *concurrentConflictStore) notificationCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.notifications)
}

func (s *concurrentConflictStore) GetNotificationByRecipientAndDedupeKey(_ context.Context, recipientUserID string, dedupeKey string) (Notification, error) {
	s.mu.Lock()
	s.getCalls++
	callIndex := s.getCalls
	if s.getCalls == 2 {
		close(s.releaseGets)
	}
	release := s.releaseGets
	s.mu.Unlock()

	<-release
	if callIndex <= 2 {
		return Notification{}, ErrNotFound
	}

	key := dedupeKeyIndexKey(recipientUserID, dedupeKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	notificationID, ok := s.dedupeIndex[key]
	if !ok {
		return Notification{}, ErrNotFound
	}
	notification, ok := s.notifications[notificationID]
	if !ok {
		return Notification{}, ErrNotFound
	}
	return notification, nil
}

func (s *concurrentConflictStore) PutNotification(_ context.Context, notification Notification) error {
	key := dedupeKeyIndexKey(notification.RecipientUserID, notification.DedupeKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existingID, ok := s.dedupeIndex[key]; ok && existingID != notification.ID {
		return ErrConflict
	}
	s.notifications[notification.ID] = notification
	if strings.TrimSpace(notification.DedupeKey) != "" {
		s.dedupeIndex[key] = notification.ID
	}
	return nil
}

func (s *concurrentConflictStore) ListNotificationsByRecipient(_ context.Context, recipientUserID string, pageSize int, pageToken string) (NotificationPage, error) {
	base := newFakeStore()
	s.mu.Lock()
	base.notifications = make(map[string]Notification, len(s.notifications))
	for id, notification := range s.notifications {
		base.notifications[id] = notification
	}
	base.dedupeIndex = make(map[string]string, len(s.dedupeIndex))
	for key, value := range s.dedupeIndex {
		base.dedupeIndex[key] = value
	}
	s.mu.Unlock()
	return base.ListNotificationsByRecipient(context.Background(), recipientUserID, pageSize, pageToken)
}

func (s *concurrentConflictStore) CountUnreadNotificationsByRecipient(_ context.Context, recipientUserID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	unreadCount := 0
	for _, notification := range s.notifications {
		if notification.RecipientUserID != recipientUserID {
			continue
		}
		if notification.ReadAt == nil {
			unreadCount++
		}
	}
	return unreadCount, nil
}

func (s *concurrentConflictStore) MarkNotificationRead(_ context.Context, recipientUserID string, notificationID string, readAt time.Time) (Notification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notification, ok := s.notifications[notificationID]
	if !ok || notification.RecipientUserID != recipientUserID {
		return Notification{}, ErrNotFound
	}
	readValue := readAt.UTC()
	notification.ReadAt = &readValue
	notification.UpdatedAt = readValue
	s.notifications[notificationID] = notification
	return notification, nil
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		notifications: make(map[string]Notification),
		dedupeIndex:   make(map[string]string),
	}
}

func (s *fakeStore) notificationCount() int {
	return len(s.notifications)
}

func (s *fakeStore) GetNotificationByRecipientAndDedupeKey(_ context.Context, recipientUserID string, dedupeKey string) (Notification, error) {
	key := dedupeKeyIndexKey(recipientUserID, dedupeKey)
	notificationID, ok := s.dedupeIndex[key]
	if !ok {
		return Notification{}, ErrNotFound
	}
	notification, ok := s.notifications[notificationID]
	if !ok {
		return Notification{}, ErrNotFound
	}
	return notification, nil
}

func (s *fakeStore) PutNotification(_ context.Context, notification Notification) error {
	if strings.TrimSpace(notification.ID) == "" {
		return errors.New("notification id is required")
	}
	s.notifications[notification.ID] = notification
	if strings.TrimSpace(notification.DedupeKey) != "" {
		s.dedupeIndex[dedupeKeyIndexKey(notification.RecipientUserID, notification.DedupeKey)] = notification.ID
	}
	return nil
}

func (s *fakeStore) ListNotificationsByRecipient(_ context.Context, recipientUserID string, pageSize int, pageToken string) (NotificationPage, error) {
	filtered := make([]Notification, 0, len(s.notifications))
	for _, notification := range s.notifications {
		if notification.RecipientUserID != recipientUserID {
			continue
		}
		filtered = append(filtered, notification)
	}
	sort.Slice(filtered, func(i int, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	start := 0
	if pageToken != "" {
		for idx := range filtered {
			if filtered[idx].ID == pageToken {
				start = idx + 1
				break
			}
		}
	}
	if start >= len(filtered) {
		return NotificationPage{}, nil
	}

	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	page := NotificationPage{
		Notifications: append([]Notification(nil), filtered[start:end]...),
	}
	if end < len(filtered) {
		page.NextPageToken = filtered[end-1].ID
	}
	return page, nil
}

func (s *fakeStore) CountUnreadNotificationsByRecipient(_ context.Context, recipientUserID string) (int, error) {
	unreadCount := 0
	for _, notification := range s.notifications {
		if notification.RecipientUserID != recipientUserID {
			continue
		}
		if notification.ReadAt == nil {
			unreadCount++
		}
	}
	return unreadCount, nil
}

func (s *fakeStore) MarkNotificationRead(_ context.Context, recipientUserID string, notificationID string, readAt time.Time) (Notification, error) {
	notification, ok := s.notifications[notificationID]
	if !ok {
		return Notification{}, ErrNotFound
	}
	if notification.RecipientUserID != recipientUserID {
		return Notification{}, ErrNotFound
	}
	value := readAt.UTC()
	notification.ReadAt = &value
	notification.UpdatedAt = value
	s.notifications[notification.ID] = notification
	return notification, nil
}

func dedupeKeyIndexKey(recipientUserID string, dedupeKey string) string {
	return strings.TrimSpace(recipientUserID) + "|" + strings.TrimSpace(dedupeKey)
}
