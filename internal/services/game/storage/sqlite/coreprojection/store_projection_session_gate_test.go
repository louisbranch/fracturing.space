package coreprojection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestPutAndGetSessionGate_RoundTrip(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 10, 14, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-gate-rt", now)

	gate := storage.SessionGate{
		CampaignID:         "camp-gate-rt",
		SessionID:          "sess-1",
		GateID:             "gate-rt-1",
		GateType:           "decision",
		Status:             session.GateStatusOpen,
		Reason:             "Consent needed",
		CreatedAt:          now,
		CreatedByActorType: "system",
		CreatedByActorID:   "sys-1",
		Metadata: map[string]any{
			"eligible_participant_ids": []string{"p1", "p2"},
			"response_authority":       "participant",
			"options":                  []string{"proceed", "abort"},
		},
		Progress: &session.GateProgress{
			WorkflowType:           "decision",
			ResponseAuthority:      session.GateResponseAuthorityParticipant,
			EligibleParticipantIDs: []string{"p1", "p2"},
			Options:                []string{"proceed", "abort"},
			EligibleCount:          2,
			PendingCount:           2,
			PendingParticipantIDs:  []string{"p1", "p2"},
			Responses: []session.GateProgressResponse{
				{
					ParticipantID: "p1",
					Decision:      "proceed",
					Response:      map[string]any{"note": "sounds good"},
					RecordedAt:    now.Format(time.RFC3339Nano),
					ActorType:     "participant",
					ActorID:       "p1",
				},
			},
			RespondedCount: 1,
		},
		Resolution: map[string]any{"decision": "pending"},
	}

	if err := store.PutSessionGate(ctx, gate); err != nil {
		t.Fatalf("put session gate: %v", err)
	}

	got, err := store.GetSessionGate(ctx, gate.CampaignID, gate.SessionID, gate.GateID)
	if err != nil {
		t.Fatalf("get session gate: %v", err)
	}

	if got.CampaignID != gate.CampaignID {
		t.Fatalf("campaign_id = %q, want %q", got.CampaignID, gate.CampaignID)
	}
	if got.SessionID != gate.SessionID {
		t.Fatalf("session_id = %q, want %q", got.SessionID, gate.SessionID)
	}
	if got.GateID != gate.GateID {
		t.Fatalf("gate_id = %q, want %q", got.GateID, gate.GateID)
	}
	if got.GateType != gate.GateType {
		t.Fatalf("gate_type = %q, want %q", got.GateType, gate.GateType)
	}
	if got.Status != gate.Status {
		t.Fatalf("status = %q, want %q", got.Status, gate.Status)
	}
	if got.Reason != gate.Reason {
		t.Fatalf("reason = %q, want %q", got.Reason, gate.Reason)
	}
	if !got.CreatedAt.Equal(gate.CreatedAt) {
		t.Fatalf("created_at = %v, want %v", got.CreatedAt, gate.CreatedAt)
	}
	if got.CreatedByActorType != gate.CreatedByActorType {
		t.Fatalf("created_by_actor_type = %q, want %q", got.CreatedByActorType, gate.CreatedByActorType)
	}
	if got.CreatedByActorID != gate.CreatedByActorID {
		t.Fatalf("created_by_actor_id = %q, want %q", got.CreatedByActorID, gate.CreatedByActorID)
	}

	// Metadata round-trips via structured storage; verify eligible participants and options.
	eligible, ok := got.Metadata["eligible_participant_ids"].([]any)
	if !ok || len(eligible) != 2 {
		t.Fatalf("expected 2 eligible participants in metadata, got %#v", got.Metadata["eligible_participant_ids"])
	}
	if eligible[0] != "p1" || eligible[1] != "p2" {
		t.Fatalf("eligible participant ids mismatch: %v", eligible)
	}
	options, ok := got.Metadata["options"].([]any)
	if !ok || len(options) != 2 {
		t.Fatalf("expected 2 options in metadata, got %#v", got.Metadata["options"])
	}
	if options[0] != "proceed" || options[1] != "abort" {
		t.Fatalf("options mismatch: %v", options)
	}

	// Progress round-trip.
	if got.Progress == nil {
		t.Fatal("expected non-nil progress")
	}
	if got.Progress.EligibleCount != 2 {
		t.Fatalf("progress eligible_count = %d, want 2", got.Progress.EligibleCount)
	}
	if got.Progress.PendingCount != 1 {
		t.Fatalf("progress pending_count = %d, want 1", got.Progress.PendingCount)
	}
	if got.Progress.RespondedCount != 1 {
		t.Fatalf("progress responded_count = %d, want 1", got.Progress.RespondedCount)
	}
	if len(got.Progress.Responses) != 1 {
		t.Fatalf("progress responses length = %d, want 1", len(got.Progress.Responses))
	}
	if got.Progress.Responses[0].Decision != "proceed" {
		t.Fatalf("response decision = %q, want %q", got.Progress.Responses[0].Decision, "proceed")
	}

	// Resolution round-trip.
	if got.Resolution["decision"] != "pending" {
		t.Fatalf("resolution decision = %v, want %q", got.Resolution["decision"], "pending")
	}
}

func TestGetOpenSessionGate_NoGate(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	_, err := store.GetOpenSessionGate(ctx, "no-camp", "no-sess")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetOpenSessionGate_OneOpen(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 10, 15, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-open-gate", now)

	gate := storage.SessionGate{
		CampaignID:         "camp-open-gate",
		SessionID:          "sess-1",
		GateID:             "gate-open-1",
		GateType:           "decision",
		Status:             session.GateStatusOpen,
		Reason:             "Consent",
		CreatedAt:          now,
		CreatedByActorType: "system",
		Metadata:           map[string]any{"response_authority": "participant"},
	}
	if err := store.PutSessionGate(ctx, gate); err != nil {
		t.Fatalf("put session gate: %v", err)
	}

	open, err := store.GetOpenSessionGate(ctx, gate.CampaignID, gate.SessionID)
	if err != nil {
		t.Fatalf("get open session gate: %v", err)
	}
	if open.GateID != gate.GateID {
		t.Fatalf("open gate_id = %q, want %q", open.GateID, gate.GateID)
	}
	if open.Status != session.GateStatusOpen {
		t.Fatalf("open gate status = %q, want %q", open.Status, session.GateStatusOpen)
	}
}

func TestPutSessionGate_UpdateStatus(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 10, 16, 0, 0, 0, time.UTC)
	resolvedAt := now.Add(time.Hour)

	seedCampaign(t, store, "camp-gate-upd", now)

	// Insert an open gate.
	gate := storage.SessionGate{
		CampaignID:         "camp-gate-upd",
		SessionID:          "sess-1",
		GateID:             "gate-upd-1",
		GateType:           "decision",
		Status:             session.GateStatusOpen,
		Reason:             "Needs consensus",
		CreatedAt:          now,
		CreatedByActorType: "system",
		Metadata:           map[string]any{"response_authority": "participant"},
		Resolution:         map[string]any{},
	}
	if err := store.PutSessionGate(ctx, gate); err != nil {
		t.Fatalf("put open gate: %v", err)
	}

	// Update to resolved.
	gate.Status = session.GateStatusResolved
	gate.ResolvedAt = &resolvedAt
	gate.ResolvedByActorType = "participant"
	gate.ResolvedByActorID = "part-1"
	gate.Resolution = map[string]any{"decision": "proceed"}

	if err := store.PutSessionGate(ctx, gate); err != nil {
		t.Fatalf("put resolved gate: %v", err)
	}

	got, err := store.GetSessionGate(ctx, gate.CampaignID, gate.SessionID, gate.GateID)
	if err != nil {
		t.Fatalf("get resolved gate: %v", err)
	}
	if got.Status != session.GateStatusResolved {
		t.Fatalf("status = %q, want %q", got.Status, session.GateStatusResolved)
	}
	if got.ResolvedAt == nil || !got.ResolvedAt.Equal(resolvedAt.UTC()) {
		t.Fatalf("resolved_at = %v, want %v", got.ResolvedAt, resolvedAt.UTC())
	}
	if got.ResolvedByActorType != "participant" {
		t.Fatalf("resolved_by_actor_type = %q, want %q", got.ResolvedByActorType, "participant")
	}
	if got.ResolvedByActorID != "part-1" {
		t.Fatalf("resolved_by_actor_id = %q, want %q", got.ResolvedByActorID, "part-1")
	}
	if got.Resolution["decision"] != "proceed" {
		t.Fatalf("resolution decision = %v, want %q", got.Resolution["decision"], "proceed")
	}

	// Open gate should no longer be found.
	_, err = store.GetOpenSessionGate(ctx, gate.CampaignID, gate.SessionID)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for open gate after resolve, got %v", err)
	}
}
