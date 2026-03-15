package action

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestDecideAction_RejectsMissingRequestOrRollSeq(t *testing.T) {
	tests := []struct {
		name    string
		cmdType command.Type
		payload string
		code    string
	}{
		{name: "roll resolve missing request", cmdType: CommandTypeRollResolve, payload: `{"roll_seq":1}`, code: rejectionCodeRequestIDRequired},
		{name: "roll resolve missing roll seq", cmdType: CommandTypeRollResolve, payload: `{"request_id":"req"}`, code: rejectionCodeRollSeqRequired},
		{name: "outcome apply missing request", cmdType: CommandTypeOutcomeApply, payload: `{"roll_seq":1}`, code: rejectionCodeRequestIDRequired},
		{name: "outcome apply missing roll seq", cmdType: CommandTypeOutcomeApply, payload: `{"request_id":"req"}`, code: rejectionCodeRollSeqRequired},
		{name: "outcome reject missing request", cmdType: CommandTypeOutcomeReject, payload: `{"roll_seq":1}`, code: rejectionCodeRequestIDRequired},
		{name: "outcome reject missing roll seq", cmdType: CommandTypeOutcomeReject, payload: `{"request_id":"req"}`, code: rejectionCodeRollSeqRequired},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := Decide(State{}, command.Command{
				CampaignID:  "camp-1",
				Type:        tc.cmdType,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(tc.payload),
			}, time.Now)
			if len(decision.Events) != 0 {
				t.Fatalf("expected no events, got %d", len(decision.Events))
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
			}
			if decision.Rejections[0].Code != tc.code {
				t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, tc.code)
			}
		})
	}
}

func TestDecideAction_RejectsSystemOwnedByMetadata(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID: "camp-1",
		Type:       CommandTypeOutcomeApply,
		ActorType:  command.ActorTypeSystem,
		PayloadJSON: []byte(`{
			"request_id":"req-1",
			"roll_seq":1,
			"pre_effects":[{"type":"session.gate_opened","entity_type":"session","entity_id":"sess-1","system_id":"daggerheart"}]
		}`),
	}, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeOutcomeEffectSystemOwnedForbidden {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeOutcomeEffectSystemOwnedForbidden)
	}
}

func TestDecideActionOutcomeApply_EmitsPreThenOutcomeThenPost(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	decision := Decide(State{}, command.Command{
		CampaignID: "camp-1",
		Type:       CommandTypeOutcomeApply,
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
		PayloadJSON: []byte(`{
			"request_id":"req-1",
			"roll_seq":7,
			"pre_effects":[{"type":"session.gate_opened","entity_type":"session","entity_id":"sess-1","payload_json":{"gate_id":"g-1","gate_type":"test"}}],
			"post_effects":[{"type":"session.spotlight_set","entity_type":"session","entity_id":"sess-1","payload_json":{"participant_id":"p-1"}}]
		}`),
	}, func() time.Time { return now })

	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(decision.Events))
	}
	if decision.Events[0].Type != event.Type("session.gate_opened") {
		t.Fatalf("event[0] type = %s, want session.gate_opened", decision.Events[0].Type)
	}
	if decision.Events[1].Type != EventTypeOutcomeApplied {
		t.Fatalf("event[1] type = %s, want %s", decision.Events[1].Type, EventTypeOutcomeApplied)
	}
	if decision.Events[2].Type != event.Type("session.spotlight_set") {
		t.Fatalf("event[2] type = %s, want session.spotlight_set", decision.Events[2].Type)
	}
}

func TestDecideAction_RejectsEmptyNoteContent(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        CommandTypeNoteAdd,
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"content":"  ","character_id":"char-1"}`),
	}, time.Now)

	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeNoteContentRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeNoteContentRequired)
	}
}

func TestDecideAction_RejectsMalformedPayload(t *testing.T) {
	tests := []struct {
		name    string
		cmdType command.Type
	}{
		{name: "roll resolve", cmdType: CommandTypeRollResolve},
		{name: "outcome apply", cmdType: CommandTypeOutcomeApply},
		{name: "outcome reject", cmdType: CommandTypeOutcomeReject},
		{name: "note add", cmdType: CommandTypeNoteAdd},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := Decide(State{}, command.Command{
				CampaignID:  "camp-1",
				Type:        tc.cmdType,
				ActorType:   command.ActorTypeSystem,
				EntityID:    "ent-1",
				PayloadJSON: []byte(`{malformed`),
			}, time.Now)
			if len(decision.Events) != 0 {
				t.Fatalf("expected no events, got %d", len(decision.Events))
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
			}
			if decision.Rejections[0].Code != "PAYLOAD_DECODE_FAILED" {
				t.Fatalf("rejection code = %s, want PAYLOAD_DECODE_FAILED", decision.Rejections[0].Code)
			}
		})
	}
}
