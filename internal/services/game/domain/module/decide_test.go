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

type testOutputPayload struct {
	FullName string `json:"full_name"`
	ID       string `json:"id"`
}

func TestDecideFuncTransform_AcceptsAndTransforms(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"char-1"}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncTransform(cmd, decideTestState{Counter: 5}, true,
		event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(s decideTestState, hasState bool, p *testPayload, nowFn func() time.Time) *command.Rejection {
			if !hasState {
				t.Fatal("expected hasState to be true")
			}
			if s.Counter != 5 {
				t.Fatalf("state counter = %d, want 5", s.Counter)
			}
			return nil
		},
		func(s decideTestState, hasState bool, p testPayload) testOutputPayload {
			return testOutputPayload{FullName: "transformed:" + p.Name, ID: p.ID}
		},
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
	want := `{"full_name":"transformed:alice","id":"char-1"}`
	if string(decision.Events[0].PayloadJSON) != want {
		t.Fatalf("payload = %s, want %s", string(decision.Events[0].PayloadJSON), want)
	}
}

func TestDecideFuncTransform_RejectsOnUnmarshalError(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{bad json`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncTransform(cmd, decideTestState{}, false,
		event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(_ decideTestState, _ bool, _ *testPayload, _ func() time.Time) *command.Rejection {
			t.Fatal("validate should not be called on unmarshal error")
			return nil
		},
		func(_ decideTestState, _ bool, _ testPayload) testOutputPayload {
			t.Fatal("transform should not be called on unmarshal error")
			return testOutputPayload{}
		},
		now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "PAYLOAD_DECODE_FAILED" {
		t.Fatalf("rejection code = %s, want PAYLOAD_DECODE_FAILED", decision.Rejections[0].Code)
	}
}

func TestDecideFuncTransform_RejectsOnValidation(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"char-1"}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncTransform(cmd, decideTestState{Counter: 3}, true,
		event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(_ decideTestState, _ bool, _ *testPayload, _ func() time.Time) *command.Rejection {
			return &command.Rejection{Code: "TRANSFORM_REJECT", Message: "bad input"}
		},
		func(_ decideTestState, _ bool, _ testPayload) testOutputPayload {
			t.Fatal("transform should not be called after rejection")
			return testOutputPayload{}
		},
		now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "TRANSFORM_REJECT" {
		t.Fatalf("rejection code = %s, want TRANSFORM_REJECT", decision.Rejections[0].Code)
	}
}

func TestDecideFuncTransform_UsesCommandEntityID(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":"alice","id":"from-payload"}`),
		EntityID:    "from-cmd",
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncTransform(cmd, decideTestState{}, false,
		event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		nil,
		func(_ decideTestState, _ bool, p testPayload) testOutputPayload {
			return testOutputPayload{FullName: p.Name, ID: p.ID}
		},
		now)

	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	if decision.Events[0].EntityID != "from-cmd" {
		t.Fatalf("entity id = %s, want from-cmd (command takes priority)", decision.Events[0].EntityID)
	}
}

func TestDecideFuncTransform_ValidateCanMutatePayloadBeforeTransform(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.do"),
		PayloadJSON: []byte(`{"name":" alice ","id":"char-1"}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncTransform(cmd, decideTestState{}, false,
		event.Type("sys.test.done"), "character",
		func(p *testPayload) string { return p.ID },
		func(_ decideTestState, _ bool, p *testPayload, _ func() time.Time) *command.Rejection {
			p.Name = "trimmed"
			return nil
		},
		func(_ decideTestState, _ bool, p testPayload) testOutputPayload {
			return testOutputPayload{FullName: p.Name, ID: p.ID}
		},
		now)

	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	want := `{"full_name":"trimmed","id":"char-1"}`
	if string(decision.Events[0].PayloadJSON) != want {
		t.Fatalf("payload = %s, want %s", string(decision.Events[0].PayloadJSON), want)
	}
}

type multiPayload struct {
	Targets []string `json:"targets"`
}

func TestDecideFuncMulti_EmitsMultipleEvents(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.multi"),
		PayloadJSON: []byte(`{"targets":["t1","t2","t3"]}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncMulti(cmd, decideTestState{Counter: 1}, true,
		func(s decideTestState, hasState bool, p *multiPayload, nowFn func() time.Time) *command.Rejection {
			if !hasState {
				t.Fatal("expected hasState to be true")
			}
			return nil
		},
		func(s decideTestState, hasState bool, p multiPayload, nowFn func() time.Time) ([]EventSpec, error) {
			specs := make([]EventSpec, 0, len(p.Targets))
			for _, target := range p.Targets {
				specs = append(specs, EventSpec{
					Type:       event.Type("sys.test.target_hit"),
					EntityType: "character",
					EntityID:   target,
					Payload:    testPayload{Name: "hit", ID: target},
				})
			}
			return specs, nil
		},
		now)

	if len(decision.Rejections) > 0 {
		t.Fatalf("expected accept, got rejection: %v", decision.Rejections[0].Message)
	}
	if len(decision.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(decision.Events))
	}
	for i, target := range []string{"t1", "t2", "t3"} {
		if decision.Events[i].EntityID != target {
			t.Errorf("event[%d].EntityID = %s, want %s", i, decision.Events[i].EntityID, target)
		}
		if decision.Events[i].Type != event.Type("sys.test.target_hit") {
			t.Errorf("event[%d].Type = %s, want sys.test.target_hit", i, decision.Events[i].Type)
		}
	}
}

func TestDecideFuncMulti_RejectsOnUnmarshalError(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.multi"),
		PayloadJSON: []byte(`{bad json`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncMulti(cmd, decideTestState{}, false,
		nil,
		func(_ decideTestState, _ bool, _ multiPayload, _ func() time.Time) ([]EventSpec, error) {
			t.Fatal("expand should not be called on unmarshal error")
			return nil, nil
		},
		now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "PAYLOAD_DECODE_FAILED" {
		t.Fatalf("rejection code = %s, want PAYLOAD_DECODE_FAILED", decision.Rejections[0].Code)
	}
}

func TestDecideFuncMulti_RejectsOnValidation(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.multi"),
		PayloadJSON: []byte(`{"targets":["t1"]}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncMulti(cmd, decideTestState{}, false,
		func(_ decideTestState, _ bool, _ *multiPayload, _ func() time.Time) *command.Rejection {
			return &command.Rejection{Code: "MULTI_REJECT", Message: "nope"}
		},
		func(_ decideTestState, _ bool, _ multiPayload, _ func() time.Time) ([]EventSpec, error) {
			t.Fatal("expand should not be called after rejection")
			return nil, nil
		},
		now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "MULTI_REJECT" {
		t.Fatalf("rejection code = %s, want MULTI_REJECT", decision.Rejections[0].Code)
	}
}

func TestDecideFuncMulti_RejectsOnEmptySpecs(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("sys.test.multi"),
		PayloadJSON: []byte(`{"targets":[]}`),
	}
	now := func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	decision := DecideFuncMulti(cmd, decideTestState{}, false,
		nil,
		func(_ decideTestState, _ bool, _ multiPayload, _ func() time.Time) ([]EventSpec, error) {
			return nil, nil
		},
		now)

	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "NO_EVENTS" {
		t.Fatalf("rejection code = %s, want NO_EVENTS", decision.Rejections[0].Code)
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
