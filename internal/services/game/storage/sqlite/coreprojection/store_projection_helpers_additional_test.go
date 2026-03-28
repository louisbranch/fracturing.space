package coreprojection

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func TestProjectionApplyHelperValidation(t *testing.T) {
	validEvent := event.Event{CampaignID: ids.CampaignID("camp-1"), Seq: 1}
	apply := func(context.Context, event.Event, storage.ProjectionApplyTxStore) error { return nil }

	t.Run("validate request", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		tests := []struct {
			name string
			ctx  context.Context
			s    *Store
			evt  event.Event
			fn   func(context.Context, event.Event, storage.ProjectionApplyTxStore) error
			want string
		}{
			{name: "canceled context", ctx: canceledCtx, s: openTestStore(t), evt: validEvent, fn: apply, want: context.Canceled.Error()},
			{name: "nil store", ctx: context.Background(), s: nil, evt: validEvent, fn: apply, want: "storage is not configured"},
			{name: "nil callback", ctx: context.Background(), s: openTestStore(t), evt: validEvent, fn: nil, want: "projection apply callback is required"},
			{name: "missing campaign", ctx: context.Background(), s: openTestStore(t), evt: event.Event{Seq: 1}, fn: apply, want: "campaign id is required"},
			{name: "zero sequence", ctx: context.Background(), s: openTestStore(t), evt: event.Event{CampaignID: ids.CampaignID("camp-1")}, fn: apply, want: "event sequence must be greater than zero"},
		}

		for _, tt := range tests {
			err := validateProjectionApplyExactlyOnceRequest(tt.ctx, tt.s, tt.evt, tt.fn)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("%s: validateProjectionApplyExactlyOnceRequest() error = %v, want %q", tt.name, err, tt.want)
			}
		}

		if err := validateProjectionApplyExactlyOnceRequest(context.Background(), openTestStore(t), validEvent, apply); err != nil {
			t.Fatalf("validateProjectionApplyExactlyOnceRequest(valid) error = %v", err)
		}
	})

}

func TestProjectionStoresAndTxStoreBinding(t *testing.T) {
	if got := (*Store)(nil).ProjectionStores(); got.Daggerheart != nil {
		t.Fatalf("nil store projection stores = %+v", got)
	}

	store := openTestStore(t)
	if got := store.ProjectionStores(); got.Daggerheart == nil {
		t.Fatal("ProjectionStores().Daggerheart = nil, want bound store")
	}

	tx, err := store.sqlDB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	cloned := store.txStore(tx)
	if cloned == store || cloned.tx != tx || cloned.q == store.q {
		t.Fatalf("txStore clone = %#v", cloned)
	}
	if got := cloned.ProjectionStores(); got.Daggerheart == nil {
		t.Fatal("txStore ProjectionStores().Daggerheart = nil, want bound store")
	}
}

func TestSessionGateHelperConversions(t *testing.T) {
	t.Run("responses from progress returns a copy", func(t *testing.T) {
		progress := &session.GateProgress{
			Responses: []session.GateProgressResponse{{ParticipantID: "part-1", Decision: "ready"}},
		}
		got := sessionGateResponsesFromProgress(progress)
		if len(got) != 1 || got[0].ParticipantID != "part-1" {
			t.Fatalf("sessionGateResponsesFromProgress() = %+v", got)
		}
		got[0].ParticipantID = "mutated"
		if progress.Responses[0].ParticipantID != "part-1" {
			t.Fatalf("progress responses mutated = %+v", progress.Responses)
		}
		if sessionGateResponsesFromProgress(nil) != nil {
			t.Fatal("sessionGateResponsesFromProgress(nil) = non-nil, want nil")
		}
	})

	t.Run("responses to progress rows decodes payloads", func(t *testing.T) {
		now := time.Date(2026, 3, 27, 13, 0, 0, 0, time.UTC)
		rows := []db.ListSessionGateResponsesRow{
			{
				ParticipantID: "part-1",
				Decision:      "ready",
				ResponseJson:  []byte(`{"vote":"yes"}`),
				RecordedAt:    sql.NullInt64{Int64: sqliteutil.ToMillis(now), Valid: true},
				ActorType:     "participant",
				ActorID:       "part-1",
			},
			{
				ParticipantID: "part-2",
				Decision:      "wait",
				ActorType:     "participant",
				ActorID:       "part-2",
			},
		}

		got, err := sessionGateResponsesToProgressRows(rows)
		if err != nil {
			t.Fatalf("sessionGateResponsesToProgressRows() error = %v", err)
		}
		if len(got) != 2 || got[0].Response["vote"] != "yes" || got[0].RecordedAt != now.UTC().Format(time.RFC3339Nano) {
			t.Fatalf("responses = %+v", got)
		}
		if got[1].Response != nil || got[1].RecordedAt != "" {
			t.Fatalf("second response = %+v", got[1])
		}

		_, err = sessionGateResponsesToProgressRows([]db.ListSessionGateResponsesRow{{
			ParticipantID: "part-1",
			ResponseJson:  []byte("{"),
		}})
		if err == nil {
			t.Fatal("expected invalid response json error")
		}
	})

	t.Run("recorded at parsing and optional json helpers", func(t *testing.T) {
		now := time.Date(2026, 3, 27, 13, 5, 0, 0, time.UTC)

		got, err := sessionGateRecordedAtToNullMillis(now.Format(time.RFC3339Nano))
		if err != nil {
			t.Fatalf("sessionGateRecordedAtToNullMillis(valid) error = %v", err)
		}
		if !got.Valid || sqliteutil.FromMillis(got.Int64) != now {
			t.Fatalf("recorded at millis = %+v", got)
		}

		got, err = sessionGateRecordedAtToNullMillis(" ")
		if err != nil || got.Valid {
			t.Fatalf("sessionGateRecordedAtToNullMillis(blank) = (%+v, %v)", got, err)
		}

		if _, err := sessionGateRecordedAtToNullMillis("not-a-time"); err == nil {
			t.Fatal("expected invalid recorded_at parse error")
		}

		data, err := marshalOptionalJSONBytes(map[string]any{"step": "ready"}, "marshal metadata")
		if err != nil {
			t.Fatalf("marshalOptionalJSONBytes() error = %v", err)
		}
		values, err := decodeOptionalJSONObjectBytes(data, "decode metadata")
		if err != nil {
			t.Fatalf("decodeOptionalJSONObjectBytes() error = %v", err)
		}
		if values["step"] != "ready" {
			t.Fatalf("decoded values = %+v", values)
		}

		if _, err := decodeOptionalJSONObjectBytes([]byte("{"), "decode metadata"); err == nil {
			t.Fatal("expected invalid optional json error")
		}
	})
}
