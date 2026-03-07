package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// testTimestamp is a shared timestamp for deterministic test events.
var testTimestamp = time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

// rollEventBuilder constructs a roll-resolved event with a RollResolvePayload
// and appends it to a fake event store. It provides sensible defaults so tests
// only need to set the fields that matter.
type rollEventBuilder struct {
	t          *testing.T
	campaignID string
	sessionID  string
	requestID  string

	results    map[string]any
	outcome    string
	systemData map[string]any
}

// newRollEvent creates a builder with sensible defaults: campaign "camp-1",
// session "sess-1", character "char-1", action roll kind, SUCCESS_WITH_HOPE.
func newRollEvent(t *testing.T, requestID string) *rollEventBuilder {
	t.Helper()
	return &rollEventBuilder{
		t:          t,
		campaignID: "camp-1",
		sessionID:  "sess-1",
		requestID:  requestID,
		results:    map[string]any{"d20": 20},
		outcome:    pb.Outcome_SUCCESS_WITH_HOPE.String(),
		systemData: map[string]any{
			sdKeyCharacterID: "char-1",
			sdKeyRollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			sdKeyHopeFear:    true,
		},
	}
}

func (b *rollEventBuilder) withCampaign(id string) *rollEventBuilder {
	b.campaignID = id
	return b
}

func (b *rollEventBuilder) withSession(id string) *rollEventBuilder {
	b.sessionID = id
	return b
}

func (b *rollEventBuilder) withCharacter(id string) *rollEventBuilder {
	b.systemData[sdKeyCharacterID] = id
	return b
}

func (b *rollEventBuilder) withRollKind(kind pb.RollKind) *rollEventBuilder {
	b.systemData[sdKeyRollKind] = kind.String()
	return b
}

func (b *rollEventBuilder) withOutcome(outcome string) *rollEventBuilder {
	b.outcome = outcome
	b.systemData[sdKeyOutcome] = outcome
	return b
}

func (b *rollEventBuilder) withHopeFear(v bool) *rollEventBuilder {
	b.systemData[sdKeyHopeFear] = v
	return b
}

func (b *rollEventBuilder) withCrit(v bool) *rollEventBuilder {
	b.systemData[sdKeyCrit] = v
	return b
}

func (b *rollEventBuilder) withCritNegates(v bool) *rollEventBuilder {
	b.systemData[sdKeyCritNegates] = v
	return b
}

func (b *rollEventBuilder) withSystemData(key string, value any) *rollEventBuilder {
	b.systemData[key] = value
	return b
}

func (b *rollEventBuilder) withResults(results map[string]any) *rollEventBuilder {
	b.results = results
	return b
}

// appendTo marshals the payload, creates the event, and appends it to the
// store. It returns the stored event (with assigned seq).
func (b *rollEventBuilder) appendTo(store *fakeEventStore) event.Event {
	b.t.Helper()
	payload := action.RollResolvePayload{
		RequestID:  b.requestID,
		RollSeq:    1,
		Results:    b.results,
		Outcome:    b.outcome,
		SystemData: b.systemData,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		b.t.Fatalf("encode roll payload: %v", err)
	}
	evt, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  b.campaignID,
		Timestamp:   testTimestamp,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   b.sessionID,
		RequestID:   b.requestID,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    b.requestID,
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		b.t.Fatalf("append roll event: %v", err)
	}
	return evt
}

// testSessionCtx creates a context with campaign and session metadata plus a
// request ID — the common setup for session outcome tests.
func testSessionCtx(campaignID, sessionID, requestID string) context.Context {
	ctx := withCampaignSessionMetadata(context.Background(), campaignID, sessionID)
	return grpcmeta.WithRequestID(ctx, requestID)
}
