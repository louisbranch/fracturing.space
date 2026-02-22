package module

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type testPayload struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func TestDecideFunc_AcceptsValidPayload(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"char-1"}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFunc(cmd, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(p *testPayload, _ func() time.Time) *command.Rejection { return nil },
		now)

	if len(decision.Rejections) > 0 {
		t.Fatalf("expected accept, got rejection: %v", decision.Rejections[0].Message)
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	if decision.Events[0].Type != event.Type("sys.test.done") {
		t.Fatalf("event type = %s, want sys.test.done", decision.Events[0].Type)
	}
	if decision.Events[0].EntityID != "char-1" {
		t.Fatalf("entity id = %s, want char-1", decision.Events[0].EntityID)
	}
}

func TestDecideFunc_UsesCommandEntityID(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"from-payload"}`),
		EntityID:    "from-cmd",
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFunc(cmd, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(p *testPayload, _ func() time.Time) *command.Rejection { return nil },
		now)

	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	if decision.Events[0].EntityID != "from-cmd" {
		t.Fatalf("entity id = %s, want from-cmd (command takes priority)", decision.Events[0].EntityID)
	}
}

func TestDecideFunc_RejectsOnUnmarshalError(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{bad json`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFunc(cmd, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(p *testPayload, _ func() time.Time) *command.Rejection {
			t.Fatal("validate should not be called on unmarshal error")
			return nil
		}, now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
}

func TestDecideFunc_RejectsOnValidateRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"char-1"}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFunc(cmd, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(p *testPayload, _ func() time.Time) *command.Rejection {
			return &command.Rejection{Code: "TEST_REJECT", Message: "rejected"}
		}, now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "TEST_REJECT" {
		t.Fatalf("rejection code = %s, want TEST_REJECT", decision.Rejections[0].Code)
	}
}

func TestDecideFunc_DefaultsNowToTimeNow(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"char-1"}`),
	}

	before := time.Now().UTC()
	decision := DecideFunc(cmd, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(p *testPayload, _ func() time.Time) *command.Rejection { return nil },
		nil)
	after := time.Now().UTC()

	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	ts := decision.Events[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Fatalf("event timestamp %v not in expected range [%v, %v]", ts, before, after)
	}
}

type decideTestState struct {
	Counter int
}

func TestDecideFuncWithState_AcceptsWithState(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"char-1"}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncWithState(cmd, decideTestState{Counter: 5}, true, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(s decideTestState, hasState bool, p *testPayload, nowFn func() time.Time) *command.Rejection {
			if !hasState {
				t.Fatal("expected hasState to be true")
			}
			if s.Counter != 5 {
				t.Fatalf("state counter = %d, want 5", s.Counter)
			}
			return nil
		}, now)

	if len(decision.Rejections) > 0 {
		t.Fatalf("expected accept, got rejection: %v", decision.Rejections[0].Message)
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
}

func TestDecideFuncWithState_PassesFalseHasState(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"char-1"}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	var zeroState decideTestState
	decision := DecideFuncWithState(cmd, zeroState, false, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(s decideTestState, hasState bool, p *testPayload, nowFn func() time.Time) *command.Rejection {
			if hasState {
				t.Fatal("expected hasState to be false")
			}
			return nil
		}, now)

	if len(decision.Rejections) > 0 {
		t.Fatalf("expected accept, got rejection: %v", decision.Rejections[0].Message)
	}
}

func TestDecideFuncWithState_RejectsOnValidation(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"char-1"}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncWithState(cmd, decideTestState{Counter: 3}, true, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(s decideTestState, _ bool, p *testPayload, _ func() time.Time) *command.Rejection {
			return &command.Rejection{Code: "STATE_CHECK", Message: "state mismatch"}
		}, now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "STATE_CHECK" {
		t.Fatalf("rejection code = %s, want STATE_CHECK", decision.Rejections[0].Code)
	}
}

func TestDecideFuncWithState_RejectsOnUnmarshalError(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{bad json`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncWithState(cmd, decideTestState{}, false, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(_ decideTestState, _ bool, _ *testPayload, _ func() time.Time) *command.Rejection {
			t.Fatal("validate should not be called on unmarshal error")
			return nil
		}, now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
}

func TestDecideFunc_ValidateCanMutatePayload(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":" alice ","id":"char-1"}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFunc(cmd, event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(p *testPayload, _ func() time.Time) *command.Rejection {
			p.Name = "trimmed"
			return nil
		}, now)

	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	if string(decision.Events[0].PayloadJSON) != `{"name":"trimmed","id":"char-1"}` {
		t.Fatalf("payload = %s, want trimmed name", string(decision.Events[0].PayloadJSON))
	}
}
