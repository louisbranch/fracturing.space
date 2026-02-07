package bbolt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	sessiondomain "github.com/louisbranch/fracturing.space/internal/session/domain"
	"github.com/louisbranch/fracturing.space/internal/storage"
)

func TestSessionStorePutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "sess-123",
		CampaignID: "camp-456",
		Name:       "First Session",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	if err := store.PutSession(context.Background(), session); err != nil {
		t.Fatalf("put session: %v", err)
	}

	loaded, err := store.GetSession(context.Background(), "camp-456", "sess-123")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if loaded.ID != session.ID {
		t.Fatalf("expected id %q, got %q", session.ID, loaded.ID)
	}
	if loaded.CampaignID != session.CampaignID {
		t.Fatalf("expected campaign id %q, got %q", session.CampaignID, loaded.CampaignID)
	}
	if loaded.Name != session.Name {
		t.Fatalf("expected name %q, got %q", session.Name, loaded.Name)
	}
	if loaded.Status != session.Status {
		t.Fatalf("expected status %v, got %v", session.Status, loaded.Status)
	}
	if !loaded.StartedAt.Equal(now) {
		t.Fatalf("expected started_at %v, got %v", now, loaded.StartedAt)
	}
	if !loaded.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated_at %v, got %v", now, loaded.UpdatedAt)
	}
	if loaded.EndedAt != nil {
		t.Fatalf("expected nil ended_at, got %v", loaded.EndedAt)
	}
}

func TestSessionStoreGetNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetSession(context.Background(), "camp-456", "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestSessionStorePutEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.PutSession(context.Background(), sessiondomain.Session{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionStoreGetActiveSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "sess-123",
		CampaignID: "camp-456",
		Name:       "Active Session",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	// Store session and set as active
	if err := store.PutSession(context.Background(), session); err != nil {
		t.Fatalf("put session: %v", err)
	}

	// Retrieve active session
	active, err := store.GetActiveSession(context.Background(), "camp-456")
	if err != nil {
		t.Fatalf("get active session: %v", err)
	}
	if active.ID != session.ID {
		t.Fatalf("expected id %q, got %q", session.ID, active.ID)
	}
	if active.CampaignID != session.CampaignID {
		t.Fatalf("expected campaign id %q, got %q", session.CampaignID, active.CampaignID)
	}
	if active.Status != sessiondomain.SessionStatusActive {
		t.Fatalf("expected active status, got %v", active.Status)
	}
}

func TestSessionStoreGetActiveSessionNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.GetActiveSession(context.Background(), "camp-456")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestSessionStorePutSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "sess-123",
		CampaignID: "camp-456",
		Name:       "First Session",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	// Store session and set as active
	if err := store.PutSession(context.Background(), session); err != nil {
		t.Fatalf("put session: %v", err)
	}

	// Verify session is stored
	loaded, err := store.GetSession(context.Background(), "camp-456", "sess-123")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if loaded.ID != session.ID {
		t.Fatalf("expected id %q, got %q", session.ID, loaded.ID)
	}

	// Verify active session pointer
	active, err := store.GetActiveSession(context.Background(), "camp-456")
	if err != nil {
		t.Fatalf("get active session: %v", err)
	}
	if active.ID != session.ID {
		t.Fatalf("expected active session id %q, got %q", session.ID, active.ID)
	}
}

func TestSessionStorePutSessionConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	session1 := sessiondomain.Session{
		ID:         "sess-123",
		CampaignID: "camp-456",
		Name:       "First Session",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	// Store first session as active
	if err := store.PutSession(context.Background(), session1); err != nil {
		t.Fatalf("put first session: %v", err)
	}

	// Try to store second session as active - should fail
	session2 := sessiondomain.Session{
		ID:         "sess-456",
		CampaignID: "camp-456",
		Name:       "Second Session",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	err = store.PutSession(context.Background(), session2)
	if err == nil {
		t.Fatal("expected error when setting second active session")
	}
	if !errors.Is(err, storage.ErrActiveSessionExists) {
		t.Fatalf("expected ErrActiveSessionExists error, got %v", err)
	}

	// Verify first session is still active
	active, err := store.GetActiveSession(context.Background(), "camp-456")
	if err != nil {
		t.Fatalf("get active session: %v", err)
	}
	if active.ID != session1.ID {
		t.Fatalf("expected first session to still be active, got %q", active.ID)
	}
}

func TestSessionStorePutSessionNonActiveStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "sess-123",
		CampaignID: "camp-456",
		Name:       "Ended Session",
		Status:     sessiondomain.SessionStatusEnded,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	err = store.PutSession(context.Background(), session)
	if err == nil {
		t.Fatal("expected error when setting non-active session as active")
	}
	if err.Error() != "session must be ACTIVE to set as active session" {
		t.Fatalf("expected 'session must be ACTIVE' error, got %v", err)
	}
}

func TestSessionStoreEndSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	startedAt := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	endedAt := time.Date(2026, 1, 23, 13, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "sess-123",
		CampaignID: "camp-456",
		Name:       "Session",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  startedAt,
		UpdatedAt:  startedAt,
		EndedAt:    nil,
	}

	if err := store.PutSession(context.Background(), session); err != nil {
		t.Fatalf("put session: %v", err)
	}

	ended, endedNow, err := store.EndSession(context.Background(), "camp-456", "sess-123", endedAt)
	if err != nil {
		t.Fatalf("end session: %v", err)
	}
	if !endedNow {
		t.Fatal("expected endedNow true")
	}
	if ended.Status != sessiondomain.SessionStatusEnded {
		t.Fatalf("expected ended status, got %v", ended.Status)
	}
	if ended.EndedAt == nil || !ended.EndedAt.Equal(endedAt) {
		t.Fatalf("expected ended_at %v, got %v", endedAt, ended.EndedAt)
	}
	if !ended.UpdatedAt.Equal(endedAt) {
		t.Fatalf("expected updated_at %v, got %v", endedAt, ended.UpdatedAt)
	}

	_, err = store.GetActiveSession(context.Background(), "camp-456")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no active session, got %v", err)
	}
}

func TestSessionStoreEndSessionIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	endedAt := time.Date(2026, 1, 23, 13, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "sess-123",
		CampaignID: "camp-456",
		Name:       "Session",
		Status:     sessiondomain.SessionStatusEnded,
		StartedAt:  endedAt.Add(-time.Hour),
		UpdatedAt:  endedAt,
		EndedAt:    &endedAt,
	}

	payload, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	if err := store.db.Update(func(tx *bbolt.Tx) error {
		sessionBucket := tx.Bucket([]byte(sessionsBucket))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket is missing")
		}
		if err := sessionBucket.Put(sessionKey(session.CampaignID, session.ID), payload); err != nil {
			return err
		}
		activeBucket := tx.Bucket([]byte(campaignActiveSessionBucket))
		if activeBucket == nil {
			return fmt.Errorf("campaign active session bucket is missing")
		}
		return activeBucket.Put(activeSessionKey(session.CampaignID), []byte(session.ID))
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	ended, endedNow, err := store.EndSession(context.Background(), "camp-456", "sess-123", endedAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("end session: %v", err)
	}
	if endedNow {
		t.Fatal("expected endedNow false")
	}
	if ended.Status != sessiondomain.SessionStatusEnded {
		t.Fatalf("expected ended status, got %v", ended.Status)
	}
	if ended.EndedAt == nil || !ended.EndedAt.Equal(endedAt) {
		t.Fatalf("expected ended_at %v, got %v", endedAt, ended.EndedAt)
	}

	_, err = store.GetActiveSession(context.Background(), "camp-456")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no active session, got %v", err)
	}
}

func TestSessionStorePutSessionEmptyCampaignID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "sess-123",
		CampaignID: "",
		Name:       "Session",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	if err := store.PutSession(context.Background(), session); err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionStorePutSessionEmptySessionID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "",
		CampaignID: "camp-456",
		Name:       "Session",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	if err := store.PutSession(context.Background(), session); err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionStorePutSessionCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "sess-123",
		CampaignID: "camp-456",
		Name:       "Session",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	if err := store.PutSession(ctx, session); err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionStoreListSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Use different campaigns so each session can be ACTIVE (PutSession requires ACTIVE status)
	sessions := []sessiondomain.Session{
		{
			ID:         "sess-1",
			CampaignID: "camp-123",
			Name:       "Session One",
			Status:     sessiondomain.SessionStatusActive,
			StartedAt:  now,
			UpdatedAt:  now,
			EndedAt:    nil,
		},
		{
			ID:         "sess-2",
			CampaignID: "camp-456",
			Name:       "Session Two",
			Status:     sessiondomain.SessionStatusActive,
			StartedAt:  now,
			UpdatedAt:  now,
			EndedAt:    nil,
		},
		{
			ID:         "sess-3",
			CampaignID: "camp-789",
			Name:       "Session Three",
			Status:     sessiondomain.SessionStatusActive,
			StartedAt:  now,
			UpdatedAt:  now,
			EndedAt:    nil,
		},
	}

	for _, session := range sessions {
		if err := store.PutSession(context.Background(), session); err != nil {
			t.Fatalf("put session: %v", err)
		}
	}

	// Test listing sessions for camp-123
	page, err := store.ListSessions(context.Background(), "camp-123", 10, "")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(page.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(page.Sessions))
	}
	if page.Sessions[0].ID != "sess-1" {
		t.Fatalf("expected first id sess-1, got %q", page.Sessions[0].ID)
	}
	if page.Sessions[0].Status != sessiondomain.SessionStatusActive {
		t.Fatalf("expected first status Active, got %v", page.Sessions[0].Status)
	}
}

func TestSessionStoreListSessionsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	page, err := store.ListSessions(context.Background(), "camp-123", 10, "")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(page.Sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(page.Sessions))
	}
	if page.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %q", page.NextPageToken)
	}
}

func TestSessionStoreListSessionsPagination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Store first session as active (PutSession requires ACTIVE status and only one active per campaign)
	session1 := sessiondomain.Session{
		ID:         "sess-1",
		CampaignID: "camp-123",
		Name:       "Session One",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}
	if err := store.PutSession(context.Background(), session1); err != nil {
		t.Fatalf("put first session: %v", err)
	}

	// Manually insert additional sessions for pagination testing (without setting as active)
	sessions := []sessiondomain.Session{
		{
			ID:         "sess-2",
			CampaignID: "camp-123",
			Name:       "Session Two",
			Status:     sessiondomain.SessionStatusEnded,
			StartedAt:  now,
			UpdatedAt:  now,
			EndedAt:    nil,
		},
		{
			ID:         "sess-3",
			CampaignID: "camp-123",
			Name:       "Session Three",
			Status:     sessiondomain.SessionStatusEnded,
			StartedAt:  now,
			UpdatedAt:  now,
			EndedAt:    nil,
		},
	}

	// Manually insert sessions directly into the database for testing pagination
	for _, session := range sessions {
		payload, err := json.Marshal(session)
		if err != nil {
			t.Fatalf("marshal session: %v", err)
		}
		if err := store.db.Update(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(sessionsBucket))
			if bucket == nil {
				return fmt.Errorf("sessions bucket is missing")
			}
			key := sessionKey(session.CampaignID, session.ID)
			return bucket.Put(key, payload)
		}); err != nil {
			t.Fatalf("manually insert session: %v", err)
		}
	}

	page, err := store.ListSessions(context.Background(), "camp-123", 2, "")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(page.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(page.Sessions))
	}
	if page.Sessions[0].ID != "sess-1" {
		t.Fatalf("expected first id sess-1, got %q", page.Sessions[0].ID)
	}
	if page.Sessions[1].ID != "sess-2" {
		t.Fatalf("expected second id sess-2, got %q", page.Sessions[1].ID)
	}
	expectedToken := "camp-123/sess-2"
	if page.NextPageToken != expectedToken {
		t.Fatalf("expected next page token %q, got %q", expectedToken, page.NextPageToken)
	}

	page, err = store.ListSessions(context.Background(), "camp-123", 2, page.NextPageToken)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(page.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(page.Sessions))
	}
	if page.Sessions[0].ID != "sess-3" {
		t.Fatalf("expected id sess-3, got %q", page.Sessions[0].ID)
	}
	if page.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %q", page.NextPageToken)
	}
}

func TestSessionStoreListSessionsPrefixFiltering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	// Store sessions in different campaigns so each can be ACTIVE
	sessions := []sessiondomain.Session{
		{
			ID:         "sess-1",
			CampaignID: "camp-123",
			Name:       "Session One",
			Status:     sessiondomain.SessionStatusActive,
			StartedAt:  now,
			UpdatedAt:  now,
			EndedAt:    nil,
		},
		{
			ID:         "sess-2",
			CampaignID: "camp-456",
			Name:       "Session Two",
			Status:     sessiondomain.SessionStatusActive,
			StartedAt:  now,
			UpdatedAt:  now,
			EndedAt:    nil,
		},
		{
			ID:         "sess-3",
			CampaignID: "camp-123",
			Name:       "Session Three",
			Status:     sessiondomain.SessionStatusActive,
			StartedAt:  now,
			UpdatedAt:  now,
			EndedAt:    nil,
		},
	}

	// Store first session in camp-123
	if err := store.PutSession(context.Background(), sessions[0]); err != nil {
		t.Fatalf("put first session: %v", err)
	}
	// Store session in camp-456
	if err := store.PutSession(context.Background(), sessions[1]); err != nil {
		t.Fatalf("put second session: %v", err)
	}
	// Manually insert third session in camp-123 (can't use PutSession as camp-123 already has active session)
	payload, err := json.Marshal(sessions[2])
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	if err := store.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(sessionsBucket))
		if bucket == nil {
			return fmt.Errorf("sessions bucket is missing")
		}
		key := sessionKey(sessions[2].CampaignID, sessions[2].ID)
		return bucket.Put(key, payload)
	}); err != nil {
		t.Fatalf("manually insert session: %v", err)
	}

	page, err := store.ListSessions(context.Background(), "camp-123", 10, "")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(page.Sessions) != 2 {
		t.Fatalf("expected 2 sessions for camp-123, got %d", len(page.Sessions))
	}
	foundSess1 := false
	foundSess3 := false
	for _, s := range page.Sessions {
		if s.ID == "sess-1" && s.Name == "Session One" {
			foundSess1 = true
		}
		if s.ID == "sess-3" && s.Name == "Session Three" {
			foundSess3 = true
		}
		if s.CampaignID != "camp-123" {
			t.Fatalf("expected campaign id camp-123, got %q", s.CampaignID)
		}
	}
	if !foundSess1 {
		t.Fatal("expected to find sess-1")
	}
	if !foundSess3 {
		t.Fatal("expected to find sess-3")
	}
}

func TestSessionStoreListSessionsEmptyPageToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	session := sessiondomain.Session{
		ID:         "sess-1",
		CampaignID: "camp-123",
		Name:       "Session One",
		Status:     sessiondomain.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	if err := store.PutSession(context.Background(), session); err != nil {
		t.Fatalf("put session: %v", err)
	}

	page, err := store.ListSessions(context.Background(), "camp-123", 10, "")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(page.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(page.Sessions))
	}
	if page.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %q", page.NextPageToken)
	}
}

func TestSessionStoreListSessionsInvalidPageSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.ListSessions(context.Background(), "camp-123", 0, "")
	if err == nil {
		t.Fatal("expected error for zero page size")
	}
	if err.Error() != "page size must be greater than zero" {
		t.Fatalf("expected 'page size must be greater than zero' error, got %v", err)
	}

	_, err = store.ListSessions(context.Background(), "camp-123", -1, "")
	if err == nil {
		t.Fatal("expected error for negative page size")
	}
}

func TestSessionStoreListSessionsEmptyCampaignID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.ListSessions(context.Background(), "", 10, "")
	if err == nil {
		t.Fatal("expected error for empty campaign ID")
	}
	if err.Error() != "campaign id is required" {
		t.Fatalf("expected 'campaign id is required' error, got %v", err)
	}

	_, err = store.ListSessions(context.Background(), "  ", 10, "")
	if err == nil {
		t.Fatal("expected error for whitespace campaign ID")
	}
}

func TestSessionStoreListSessionsCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.ListSessions(ctx, "camp-123", 10, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionEventAppendAndList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Date(2026, 1, 25, 12, 0, 0, 0, time.UTC)
	event := sessiondomain.SessionEvent{
		SessionID:   "sess-123",
		Timestamp:   now,
		Type:        sessiondomain.SessionEventTypeNoteAdded,
		RequestID:   "req-1",
		PayloadJSON: []byte(`{"text":"note"}`),
	}

	stored, err := store.AppendSessionEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("append session event: %v", err)
	}
	if stored.Seq != 1 {
		t.Fatalf("expected seq 1, got %d", stored.Seq)
	}

	stored2, err := store.AppendSessionEvent(context.Background(), sessiondomain.SessionEvent{
		SessionID:   "sess-123",
		Timestamp:   now,
		Type:        sessiondomain.SessionEventTypeNoteAdded,
		RequestID:   "req-2",
		PayloadJSON: []byte(`{"text":"note2"}`),
	})
	if err != nil {
		t.Fatalf("append session event: %v", err)
	}
	if stored2.Seq != 2 {
		t.Fatalf("expected seq 2, got %d", stored2.Seq)
	}

	events, err := store.ListSessionEvents(context.Background(), "sess-123", 0, 10)
	if err != nil {
		t.Fatalf("list session events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Seq != 1 || events[1].Seq != 2 {
		t.Fatalf("expected seq order 1,2 got %d,%d", events[0].Seq, events[1].Seq)
	}

	filtered, err := store.ListSessionEvents(context.Background(), "sess-123", 1, 10)
	if err != nil {
		t.Fatalf("list session events after seq: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 event after seq, got %d", len(filtered))
	}
	if filtered[0].Seq != 2 {
		t.Fatalf("expected seq 2 after filter, got %d", filtered[0].Seq)
	}
}

func TestSessionEventAppendEmptySessionID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duality.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.AppendSessionEvent(context.Background(), sessiondomain.SessionEvent{
		SessionID: " ",
		Type:      sessiondomain.SessionEventTypeNoteAdded,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
