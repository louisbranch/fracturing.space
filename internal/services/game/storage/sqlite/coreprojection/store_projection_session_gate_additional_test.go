package coreprojection

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestSessionGateUpdateReplacesChildRowsAndRoundTripsProgress(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 4, 12, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-gate-update", now)
	seedSession(t, store, "camp-gate-update", "sess-1", now)

	initial := storage.SessionGate{
		CampaignID:         "camp-gate-update",
		SessionID:          "sess-1",
		GateID:             "gate-1",
		GateType:           "decision",
		Status:             session.GateStatusOpen,
		Reason:             "Choose a route",
		CreatedAt:          now,
		CreatedByActorType: "system",
		Metadata: map[string]any{
			"eligible_participant_ids": []string{"p2", "p1"},
			"response_authority":       session.GateResponseAuthorityParticipant,
			"topic":                    "bridge",
		},
		Progress: &session.GateProgress{
			Responses: []session.GateProgressResponse{
				{
					ParticipantID: "p2",
					Decision:      "north",
					Response:      map[string]any{"note": "go high"},
					RecordedAt:    "2026-02-04T08:00:00-05:00",
					ActorType:     "participant",
					ActorID:       "p2",
				},
				{
					ParticipantID: "p1",
					Decision:      "south",
					Response:      map[string]any{"note": "go low"},
					RecordedAt:    "2026-02-04T13:05:00Z",
					ActorType:     "participant",
					ActorID:       "p1",
				},
			},
		},
		Resolution: map[string]any{"decision": "pending"},
	}

	tx, err := store.sqlDB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	if err := store.txStore(tx).PutSessionGate(context.Background(), initial); err != nil {
		t.Fatalf("put initial session gate in tx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	resolvedAt := now.Add(2 * time.Hour)
	updated := storage.SessionGate{
		CampaignID:          initial.CampaignID,
		SessionID:           initial.SessionID,
		GateID:              initial.GateID,
		GateType:            initial.GateType,
		Status:              session.GateStatusResolved,
		Reason:              "Route chosen",
		CreatedAt:           initial.CreatedAt,
		CreatedByActorType:  initial.CreatedByActorType,
		ResolvedAt:          &resolvedAt,
		ResolvedByActorType: "participant",
		ResolvedByActorID:   "p3",
		Metadata: map[string]any{
			"eligible_participant_ids": []string{"p3", "p1"},
			"response_authority":       session.GateResponseAuthorityParticipant,
			"topic":                    "tower",
		},
		Progress: &session.GateProgress{
			Responses: []session.GateProgressResponse{
				{
					ParticipantID: "p3",
					Decision:      "wait",
					Response:      map[string]any{"note": "hold position"},
					RecordedAt:    "2026-02-04T10:00:00-05:00",
					ActorType:     "participant",
					ActorID:       "p3",
				},
				{
					ParticipantID: "p1",
					Decision:      "north",
					Response:      map[string]any{"note": "advance"},
					RecordedAt:    "2026-02-04T15:02:00Z",
					ActorType:     "participant",
					ActorID:       "p1",
				},
			},
		},
		Resolution: map[string]any{
			"decision": "north",
			"note":     "majority chose the tower path",
		},
	}

	if err := store.PutSessionGate(context.Background(), updated); err != nil {
		t.Fatalf("put updated session gate: %v", err)
	}

	got, err := store.GetSessionGate(context.Background(), updated.CampaignID, updated.SessionID, updated.GateID)
	if err != nil {
		t.Fatalf("get session gate: %v", err)
	}

	if got.Status != session.GateStatusResolved {
		t.Fatalf("status = %q, want resolved", got.Status)
	}
	if got.Reason != updated.Reason {
		t.Fatalf("reason = %q, want %q", got.Reason, updated.Reason)
	}
	if got.ResolvedAt == nil || !got.ResolvedAt.Equal(resolvedAt.UTC()) {
		t.Fatalf("resolved_at = %v, want %v", got.ResolvedAt, resolvedAt.UTC())
	}
	if got.ResolvedByActorType != "participant" || got.ResolvedByActorID != "p3" {
		t.Fatalf("resolved by = %q/%q", got.ResolvedByActorType, got.ResolvedByActorID)
	}

	eligible, ok := got.Metadata["eligible_participant_ids"].([]any)
	if !ok || len(eligible) != 2 || eligible[0] != "p1" || eligible[1] != "p3" {
		t.Fatalf("eligible participant ids = %#v", got.Metadata["eligible_participant_ids"])
	}
	if got.Metadata["topic"] != "tower" {
		t.Fatalf("metadata topic = %#v", got.Metadata["topic"])
	}

	if got.Progress == nil {
		t.Fatal("expected progress to round-trip")
	}
	if got.Progress.EligibleCount != 2 || got.Progress.RespondedCount != 2 || got.Progress.PendingCount != 0 || !got.Progress.AllResponded {
		t.Fatalf("progress counts = %#v", got.Progress)
	}
	if len(got.Progress.Responses) != 2 {
		t.Fatalf("progress responses len = %d, want 2", len(got.Progress.Responses))
	}
	if got.Progress.Responses[0].ParticipantID != "p1" || got.Progress.Responses[1].ParticipantID != "p3" {
		t.Fatalf("progress response order = %#v", got.Progress.Responses)
	}
	if got.Progress.Responses[0].Response["note"] != "advance" {
		t.Fatalf("p1 response payload = %#v", got.Progress.Responses[0].Response)
	}
	if got.Progress.Responses[1].RecordedAt != "2026-02-04T15:00:00Z" {
		t.Fatalf("p3 recorded_at = %q, want UTC-normalized value", got.Progress.Responses[1].RecordedAt)
	}
	if _, ok := got.Progress.DecisionCounts["south"]; ok {
		t.Fatalf("stale decision counts leaked from replaced response rows: %#v", got.Progress.DecisionCounts)
	}
	if got.Progress.DecisionCounts["north"] != 1 || got.Progress.DecisionCounts["wait"] != 1 {
		t.Fatalf("decision counts = %#v", got.Progress.DecisionCounts)
	}

	if got.Resolution["decision"] != "north" || got.Resolution["note"] != "majority chose the tower path" {
		t.Fatalf("resolution = %#v", got.Resolution)
	}

	openGate, err := store.GetOpenSessionGate(context.Background(), updated.CampaignID, updated.SessionID)
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no open session gate after resolved update, got gate=%#v err=%v", openGate, err)
	}
}

func TestGetOpenSessionGateRejectsCorruptMultipleOpenRows(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 4, 15, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-gate-corrupt", now)
	seedSession(t, store, "camp-gate-corrupt", "sess-1", now)

	if err := store.PutSessionGate(context.Background(), storage.SessionGate{
		CampaignID:         "camp-gate-corrupt",
		SessionID:          "sess-1",
		GateID:             "gate-1",
		GateType:           "decision",
		Status:             session.GateStatusOpen,
		Reason:             "First open gate",
		CreatedAt:          now,
		CreatedByActorType: "system",
	}); err != nil {
		t.Fatalf("put first open gate: %v", err)
	}

	if _, err := store.sqlDB.ExecContext(context.Background(), `DROP INDEX idx_session_gates_open`); err != nil {
		t.Fatalf("drop open-gate unique index: %v", err)
	}
	if _, err := store.sqlDB.ExecContext(context.Background(), `
		INSERT INTO session_gates (
			campaign_id, session_id, gate_id, gate_type, status, reason,
			created_at, created_by_actor_type, created_by_actor_id,
			response_authority, resolution_decision
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"camp-gate-corrupt",
		"sess-1",
		"gate-2",
		"decision",
		"open",
		"Second open gate",
		now.Add(time.Minute).UnixMilli(),
		"system",
		"",
		"",
		"",
	); err != nil {
		t.Fatalf("insert second open gate: %v", err)
	}

	_, err := store.GetOpenSessionGate(context.Background(), "camp-gate-corrupt", "sess-1")
	if err == nil {
		t.Fatal("expected multiple-open gate error")
	}
	if !strings.Contains(err.Error(), "multiple open session gates") {
		t.Fatalf("expected multiple-open error, got %v", err)
	}
}
