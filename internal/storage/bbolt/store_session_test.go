package bbolt

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	sessiondomain "github.com/louisbranch/duality-engine/internal/session/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
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
	if err := store.PutSessionWithActivePointer(context.Background(), session); err != nil {
		t.Fatalf("put session with active pointer: %v", err)
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

func TestSessionStorePutSessionWithActivePointer(t *testing.T) {
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
	if err := store.PutSessionWithActivePointer(context.Background(), session); err != nil {
		t.Fatalf("put session with active pointer: %v", err)
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

func TestSessionStorePutSessionWithActivePointerConflict(t *testing.T) {
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
	if err := store.PutSessionWithActivePointer(context.Background(), session1); err != nil {
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

	err = store.PutSessionWithActivePointer(context.Background(), session2)
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

func TestSessionStorePutSessionWithActivePointerNonActiveStatus(t *testing.T) {
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
		Name:       "Paused Session",
		Status:     sessiondomain.SessionStatusPaused,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    nil,
	}

	err = store.PutSessionWithActivePointer(context.Background(), session)
	if err == nil {
		t.Fatal("expected error when setting non-active session as active")
	}
	if err.Error() != "session must be ACTIVE to set as active session" {
		t.Fatalf("expected 'session must be ACTIVE' error, got %v", err)
	}
}

func TestSessionStorePutSessionWithActivePointerEmptyCampaignID(t *testing.T) {
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

	if err := store.PutSessionWithActivePointer(context.Background(), session); err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionStorePutSessionWithActivePointerEmptySessionID(t *testing.T) {
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

	if err := store.PutSessionWithActivePointer(context.Background(), session); err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionStorePutSessionWithActivePointerCanceledContext(t *testing.T) {
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

	if err := store.PutSessionWithActivePointer(ctx, session); err == nil {
		t.Fatal("expected error")
	}
}
