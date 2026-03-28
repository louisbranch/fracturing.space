package coreprojection

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestPutSessionGateRejectsInvalidResponseTimestamp(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-gate-write-error", now)
	seedSession(t, store, "camp-gate-write-error", "sess-1", now)

	err := store.PutSessionGate(context.Background(), storage.SessionGate{
		CampaignID:         "camp-gate-write-error",
		SessionID:          "sess-1",
		GateID:             "gate-1",
		GateType:           "decision",
		Status:             session.GateStatusOpen,
		Reason:             "Need a vote",
		CreatedAt:          now,
		CreatedByActorType: "system",
		Metadata: map[string]any{
			"eligible_participant_ids": []string{"p1"},
		},
		Progress: &session.GateProgress{
			Responses: []session.GateProgressResponse{{
				ParticipantID: "p1",
				Decision:      "north",
				RecordedAt:    "not-a-time",
				ActorType:     "participant",
				ActorID:       "p1",
			}},
		},
	})
	if err == nil {
		t.Fatal("expected invalid response timestamp error")
	}
	if !strings.Contains(err.Error(), `encode session gate response timestamp for participant "p1"`) {
		t.Fatalf("expected timestamp encoding error, got %v", err)
	}
}

func TestGetSessionGateRejectsCorruptStoredPayloads(t *testing.T) {
	tests := []struct {
		name    string
		corrupt func(t *testing.T, store *Store)
		want    string
	}{
		{
			name: "metadata extras",
			corrupt: func(t *testing.T, store *Store) {
				t.Helper()
				if _, err := store.sqlDB.ExecContext(context.Background(), `
					UPDATE session_gates
					SET metadata_extra_json = ?
					WHERE campaign_id = ? AND session_id = ? AND gate_id = ?
				`, []byte("{"), "camp-gate-read-error", "sess-1", "gate-1"); err != nil {
					t.Fatalf("corrupt metadata extras: %v", err)
				}
			},
			want: "load session gate details: decode session gate metadata extras",
		},
		{
			name: "resolution extras",
			corrupt: func(t *testing.T, store *Store) {
				t.Helper()
				if _, err := store.sqlDB.ExecContext(context.Background(), `
					UPDATE session_gates
					SET resolution_extra_json = ?
					WHERE campaign_id = ? AND session_id = ? AND gate_id = ?
				`, []byte("{"), "camp-gate-read-error", "sess-1", "gate-1"); err != nil {
					t.Fatalf("corrupt resolution extras: %v", err)
				}
			},
			want: "load session gate details: decode session gate resolution extras",
		},
		{
			name: "response payload",
			corrupt: func(t *testing.T, store *Store) {
				t.Helper()
				if _, err := store.sqlDB.ExecContext(context.Background(), `
					UPDATE session_gate_responses
					SET response_json = ?
					WHERE campaign_id = ? AND session_id = ? AND gate_id = ? AND participant_id = ?
				`, []byte("{"), "camp-gate-read-error", "sess-1", "gate-1", "p1"); err != nil {
					t.Fatalf("corrupt response payload: %v", err)
				}
			},
			want: "load session gate details: decode session gate response payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openTestStore(t)
			seedCorruptibleSessionGate(t, store, "camp-gate-read-error", "sess-1", "gate-1")
			tt.corrupt(t, store)

			_, err := store.GetSessionGate(context.Background(), "camp-gate-read-error", "sess-1", "gate-1")
			if err == nil {
				t.Fatal("expected corrupt session gate read to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestGetOpenSessionGateRejectsCorruptStoredPayloads(t *testing.T) {
	store := openTestStore(t)
	seedCorruptibleSessionGate(t, store, "camp-gate-open-read-error", "sess-1", "gate-1")

	if _, err := store.sqlDB.ExecContext(context.Background(), `
		UPDATE session_gates
		SET metadata_extra_json = ?
		WHERE campaign_id = ? AND session_id = ? AND gate_id = ?
	`, []byte("{"), "camp-gate-open-read-error", "sess-1", "gate-1"); err != nil {
		t.Fatalf("corrupt open gate metadata extras: %v", err)
	}

	_, err := store.GetOpenSessionGate(context.Background(), "camp-gate-open-read-error", "sess-1")
	if err == nil {
		t.Fatal("expected corrupt open session gate read to fail")
	}
	if !strings.Contains(err.Error(), "load open session gate details: decode session gate metadata extras") {
		t.Fatalf("expected wrapped open-gate decode error, got %v", err)
	}
}

func seedCorruptibleSessionGate(t *testing.T, store *Store, campaignID, sessionID, gateID string) {
	t.Helper()

	now := time.Date(2026, 2, 5, 14, 0, 0, 0, time.UTC)
	seedCampaign(t, store, campaignID, now)
	seedSession(t, store, campaignID, sessionID, now)

	err := store.PutSessionGate(context.Background(), storage.SessionGate{
		CampaignID:         campaignID,
		SessionID:          sessionID,
		GateID:             gateID,
		GateType:           "decision",
		Status:             session.GateStatusOpen,
		Reason:             "Need a vote",
		CreatedAt:          now,
		CreatedByActorType: "system",
		Metadata: map[string]any{
			"eligible_participant_ids": []string{"p1", "p2"},
			"topic":                    "bridge",
		},
		Progress: &session.GateProgress{
			Responses: []session.GateProgressResponse{{
				ParticipantID: "p1",
				Decision:      "north",
				Response:      map[string]any{"note": "advance"},
				RecordedAt:    now.Format(time.RFC3339Nano),
				ActorType:     "participant",
				ActorID:       "p1",
			}},
		},
		Resolution: map[string]any{
			"decision": "pending",
			"note":     "waiting on more votes",
		},
	})
	if err != nil {
		t.Fatalf("seed session gate: %v", err)
	}
}
