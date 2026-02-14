package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCampaignGetNotFound(t *testing.T) {
	store := openTestStore(t)

	_, err := store.Get(context.Background(), "no-such-campaign")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestParticipantGetAndDelete(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-part", now)
	p := seedParticipant(t, store, "camp-part", "part-1", "user-1", now)

	got, err := store.GetParticipant(context.Background(), "camp-part", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if got.ID != p.ID || got.CampaignID != p.CampaignID {
		t.Fatalf("expected identity to match")
	}
	if got.UserID != p.UserID || got.DisplayName != p.DisplayName {
		t.Fatalf("expected user id/display name to match")
	}
	if got.Role != p.Role || got.Controller != p.Controller || got.CampaignAccess != p.CampaignAccess {
		t.Fatalf("expected role/controller/access to match")
	}

	if err := store.DeleteParticipant(context.Background(), "camp-part", "part-1"); err != nil {
		t.Fatalf("delete participant: %v", err)
	}

	_, err = store.GetParticipant(context.Background(), "camp-part", "part-1")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestParticipantGetNotFound(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-part-nf", now)

	_, err := store.GetParticipant(context.Background(), "camp-part-nf", "no-part")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListParticipantsByCampaign(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-lp", now)

	for _, id := range []string{"part-a", "part-b", "part-c"} {
		seedParticipant(t, store, "camp-lp", id, "user-"+id, now)
	}

	all, err := store.ListParticipantsByCampaign(context.Background(), "camp-lp")
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(all))
	}

	// Empty campaign returns empty
	empty, err := store.ListParticipantsByCampaign(context.Background(), "camp-empty")
	if err != nil {
		t.Fatalf("list participants empty: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 participants for empty campaign, got %d", len(empty))
	}
}

func TestSessionGetAndListSessions(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-sess", now)

	s := seedSession(t, store, "camp-sess", "sess-1", now)

	got, err := store.GetSession(context.Background(), "camp-sess", "sess-1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.ID != s.ID || got.CampaignID != s.CampaignID {
		t.Fatalf("expected identity to match")
	}
	if got.Name != s.Name || got.Status != session.SessionStatusActive {
		t.Fatalf("expected name/status to match")
	}
	if !got.StartedAt.Equal(s.StartedAt) {
		t.Fatalf("expected started at to match")
	}

	// End the active session before creating the next one
	if _, _, err := store.EndSession(context.Background(), "camp-sess", "sess-1", now.Add(time.Hour)); err != nil {
		t.Fatalf("end session 1: %v", err)
	}

	// Create two more sessions (ending each before the next)
	sess2 := session.Session{
		ID: "sess-2", CampaignID: "camp-sess", Name: "Session 2",
		Status: session.SessionStatusActive, StartedAt: now.Add(2 * time.Hour), UpdatedAt: now.Add(2 * time.Hour),
	}
	if err := store.PutSession(context.Background(), sess2); err != nil {
		t.Fatalf("put session 2: %v", err)
	}
	if _, _, err := store.EndSession(context.Background(), "camp-sess", "sess-2", now.Add(3*time.Hour)); err != nil {
		t.Fatalf("end session 2: %v", err)
	}

	sess3 := session.Session{
		ID: "sess-3", CampaignID: "camp-sess", Name: "Session 3",
		Status: session.SessionStatusActive, StartedAt: now.Add(4 * time.Hour), UpdatedAt: now.Add(4 * time.Hour),
	}
	if err := store.PutSession(context.Background(), sess3); err != nil {
		t.Fatalf("put session 3: %v", err)
	}
	if _, _, err := store.EndSession(context.Background(), "camp-sess", "sess-3", now.Add(5*time.Hour)); err != nil {
		t.Fatalf("end session 3: %v", err)
	}

	// List with paging
	page, err := store.ListSessions(context.Background(), "camp-sess", 2, "")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(page.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(page.Sessions))
	}
	if page.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	second, err := store.ListSessions(context.Background(), "camp-sess", 2, page.NextPageToken)
	if err != nil {
		t.Fatalf("list sessions page 2: %v", err)
	}
	if len(second.Sessions) != 1 {
		t.Fatalf("expected 1 session on page 2, got %d", len(second.Sessions))
	}
}

func TestSessionGetNotFound(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-sess-nf", now)

	_, err := store.GetSession(context.Background(), "camp-sess-nf", "no-sess")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCampaignForkMetadata(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-fork", now)

	expected := storage.ForkMetadata{
		ParentCampaignID: "camp-parent",
		ForkEventSeq:     42,
		OriginCampaignID: "camp-origin",
	}

	if err := store.SetCampaignForkMetadata(context.Background(), "camp-fork", expected); err != nil {
		t.Fatalf("set fork metadata: %v", err)
	}

	got, err := store.GetCampaignForkMetadata(context.Background(), "camp-fork")
	if err != nil {
		t.Fatalf("get fork metadata: %v", err)
	}
	if got.ParentCampaignID != expected.ParentCampaignID {
		t.Fatalf("expected parent campaign id %q, got %q", expected.ParentCampaignID, got.ParentCampaignID)
	}
	if got.ForkEventSeq != expected.ForkEventSeq {
		t.Fatalf("expected fork event seq %d, got %d", expected.ForkEventSeq, got.ForkEventSeq)
	}
	if got.OriginCampaignID != expected.OriginCampaignID {
		t.Fatalf("expected origin campaign id %q, got %q", expected.OriginCampaignID, got.OriginCampaignID)
	}

	// Campaign without fork metadata returns zero-value (not ErrNotFound)
	seedCampaign(t, store, "camp-no-fork", now)
	zeroMeta, err := store.GetCampaignForkMetadata(context.Background(), "camp-no-fork")
	if err != nil {
		t.Fatalf("expected no error for campaign without fork metadata, got %v", err)
	}
	if zeroMeta.ParentCampaignID != "" || zeroMeta.ForkEventSeq != 0 || zeroMeta.OriginCampaignID != "" {
		t.Fatalf("expected zero-value fork metadata, got %+v", zeroMeta)
	}

	// Non-existent campaign returns ErrNotFound
	_, err = store.GetCampaignForkMetadata(context.Background(), "camp-does-not-exist")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-existent campaign, got %v", err)
	}
}

func TestGetGameStatistics(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-stats-1", now)
	seedCampaign(t, store, "camp-stats-2", now)
	seedParticipant(t, store, "camp-stats-1", "part-1", "user-1", now)
	seedParticipant(t, store, "camp-stats-1", "part-2", "user-2", now)
	seedCharacter(t, store, "camp-stats-1", "char-1", "Aria", character.CharacterKindPC, now)
	seedSession(t, store, "camp-stats-1", "sess-1", now)

	stats, err := store.GetGameStatistics(context.Background(), nil)
	if err != nil {
		t.Fatalf("get game statistics: %v", err)
	}
	if stats.CampaignCount != 2 {
		t.Fatalf("expected 2 campaigns, got %d", stats.CampaignCount)
	}
	if stats.ParticipantCount != 2 {
		t.Fatalf("expected 2 participants, got %d", stats.ParticipantCount)
	}
	if stats.CharacterCount != 1 {
		t.Fatalf("expected 1 character, got %d", stats.CharacterCount)
	}
	if stats.SessionCount != 1 {
		t.Fatalf("expected 1 session, got %d", stats.SessionCount)
	}

	// With future "since" → zero counts
	future := now.Add(24 * time.Hour)
	futureStats, err := store.GetGameStatistics(context.Background(), &future)
	if err != nil {
		t.Fatalf("get game statistics with since: %v", err)
	}
	if futureStats.CampaignCount != 0 || futureStats.SessionCount != 0 {
		t.Fatalf("expected zero counts with future since, got campaigns=%d sessions=%d",
			futureStats.CampaignCount, futureStats.SessionCount)
	}
}

func TestAppendTelemetryEvent(t *testing.T) {
	store := openTestEventsStore(t)
	now := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)

	// With Attributes map
	err := store.AppendTelemetryEvent(context.Background(), storage.TelemetryEvent{
		Timestamp:  now,
		EventName:  "test.event",
		Severity:   "info",
		CampaignID: "camp-tel",
		SessionID:  "sess-1",
		ActorType:  "system",
		RequestID:  "req-1",
		Attributes: map[string]any{"key": "value"},
	})
	if err != nil {
		t.Fatalf("append telemetry event with attributes: %v", err)
	}

	// With AttributesJSON
	err = store.AppendTelemetryEvent(context.Background(), storage.TelemetryEvent{
		Timestamp:      now,
		EventName:      "test.event2",
		Severity:       "warn",
		AttributesJSON: []byte(`{"key":"value2"}`),
	})
	if err != nil {
		t.Fatalf("append telemetry event with json: %v", err)
	}

	// Required field validation: missing event name
	err = store.AppendTelemetryEvent(context.Background(), storage.TelemetryEvent{
		Timestamp: now,
		Severity:  "info",
	})
	if err == nil {
		t.Fatal("expected error for missing event name")
	}

	// Required field validation: missing severity
	err = store.AppendTelemetryEvent(context.Background(), storage.TelemetryEvent{
		Timestamp: now,
		EventName: "test.event3",
	})
	if err == nil {
		t.Fatal("expected error for missing severity")
	}
}

func TestSnapshotLifecycle(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-snaps", now)

	snap1 := storage.Snapshot{
		CampaignID:          "camp-snaps",
		SessionID:           "sess-1",
		EventSeq:            5,
		CharacterStatesJSON: []byte(`[{"hp":10}]`),
		GMStateJSON:         []byte(`{"fear":3}`),
		SystemStateJSON:     []byte(`{}`),
		CreatedAt:           now,
	}
	if err := store.PutSnapshot(context.Background(), snap1); err != nil {
		t.Fatalf("put snapshot 1: %v", err)
	}

	snap2 := storage.Snapshot{
		CampaignID:          "camp-snaps",
		SessionID:           "sess-2",
		EventSeq:            10,
		CharacterStatesJSON: []byte(`[{"hp":8}]`),
		GMStateJSON:         []byte(`{"fear":5}`),
		SystemStateJSON:     []byte(`{}`),
		CreatedAt:           now.Add(time.Hour),
	}
	if err := store.PutSnapshot(context.Background(), snap2); err != nil {
		t.Fatalf("put snapshot 2: %v", err)
	}

	// GetSnapshot by session
	got, err := store.GetSnapshot(context.Background(), "camp-snaps", "sess-1")
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if got.EventSeq != 5 {
		t.Fatalf("expected event seq 5, got %d", got.EventSeq)
	}
	if string(got.CharacterStatesJSON) != `[{"hp":10}]` {
		t.Fatalf("expected character states to match")
	}
	if string(got.GMStateJSON) != `{"fear":3}` {
		t.Fatalf("expected gm state to match")
	}

	// GetLatestSnapshot
	latest, err := store.GetLatestSnapshot(context.Background(), "camp-snaps")
	if err != nil {
		t.Fatalf("get latest snapshot: %v", err)
	}
	if latest.EventSeq != 10 {
		t.Fatalf("expected latest event seq 10, got %d", latest.EventSeq)
	}

	// ListSnapshots
	list, err := store.ListSnapshots(context.Background(), "camp-snaps", 10)
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(list))
	}
	// Ordered by event seq descending
	if list[0].EventSeq != 10 || list[1].EventSeq != 5 {
		t.Fatalf("expected descending order, got seq %d, %d", list[0].EventSeq, list[1].EventSeq)
	}

	// Not-found path
	_, err = store.GetSnapshot(context.Background(), "camp-snaps", "no-sess")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	_, err = store.GetLatestSnapshot(context.Background(), "no-camp")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for latest, got %v", err)
	}
}
