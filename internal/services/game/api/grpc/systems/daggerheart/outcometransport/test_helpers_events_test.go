package outcometransport

import (
	"context"
	"encoding/json"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/test/mock/gamefakes"
)

type rollEventConfig struct {
	campaignID string
	sessionID  string
	requestID  string
	outcome    string
	metadata   workflowtransport.RollSystemMetadata
}

type eventPayload struct {
	RequestID  string         `json:"request_id,omitempty"`
	RollSeq    uint64         `json:"roll_seq,omitempty"`
	Outcome    string         `json:"outcome,omitempty"`
	SystemData map[string]any `json:"system_data,omitempty"`
}

func appendRollEvent(t testingT, store *gamefakes.EventStore, config rollEventConfig) event.Event {
	t.Helper()
	campaignID := config.campaignID
	if campaignID == "" {
		campaignID = "camp-1"
	}
	sessionID := config.sessionID
	if sessionID == "" {
		sessionID = "sess-1"
	}
	requestID := config.requestID
	if requestID == "" {
		requestID = "req-1"
	}
	metadata := config.metadata
	if metadata.CharacterID == "" {
		metadata.CharacterID = "char-1"
	}
	if metadata.RollKind == "" {
		metadata.RollKind = pb.RollKind_ROLL_KIND_ACTION.String()
	}
	if metadata.HopeFear == nil {
		metadata.HopeFear = workflowtransport.BoolPtr(true)
	}
	outcome := config.outcome
	if outcome == "" {
		outcome = pb.Outcome_SUCCESS_WITH_HOPE.String()
	}

	payloadJSON, err := json.Marshal(eventPayload{
		RequestID:  requestID,
		RollSeq:    1,
		Outcome:    outcome,
		SystemData: metadata.MapValue(),
	})
	if err != nil {
		t.Fatalf("marshal roll payload: %v", err)
	}
	evt, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(campaignID),
		Timestamp:   testTimestamp,
		Type:        eventTypeActionRollResolved,
		SessionID:   ids.SessionID(sessionID),
		RequestID:   requestID,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    requestID,
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}
	return evt
}
