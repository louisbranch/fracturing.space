package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestDecideStart_NormalizesCharacterControllers(t *testing.T) {
	t.Parallel()

	now := func() time.Time { return time.Date(2026, 3, 27, 13, 0, 0, 0, time.UTC) }
	decision := decideStart(State{}, command.Command{
		Type: CommandTypeStart,
		PayloadJSON: mustMarshalSessionJSON(t, StartPayload{
			SessionID:   ids.SessionID(" sess-1 "),
			SessionName: " Session One ",
			CharacterControllers: []CharacterControllerAssignment{
				{CharacterID: ids.CharacterID(" char-1 "), ParticipantID: ids.ParticipantID(" part-1 ")},
				{CharacterID: ids.CharacterID(" char-2 "), ParticipantID: ids.ParticipantID(" part-2 ")},
			},
		}),
	}, now)

	if len(decision.Rejections) != 0 {
		t.Fatalf("unexpected rejections: %+v", decision.Rejections)
	}
	if len(decision.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(decision.Events))
	}
	if decision.Events[0].SessionID != ids.SessionID("sess-1") {
		t.Fatalf("SessionID = %q, want %q", decision.Events[0].SessionID, ids.SessionID("sess-1"))
	}

	var got StartPayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &got); err != nil {
		t.Fatalf("json.Unmarshal(event payload): %v", err)
	}
	if got.SessionID != ids.SessionID("sess-1") || got.SessionName != "Session One" {
		t.Fatalf("payload = %#v", got)
	}
	if len(got.CharacterControllers) != 2 {
		t.Fatalf("controllers = %d, want 2", len(got.CharacterControllers))
	}
	if got.CharacterControllers[0].CharacterID != ids.CharacterID("char-1") || got.CharacterControllers[0].ParticipantID != ids.ParticipantID("part-1") {
		t.Fatalf("first controller = %#v", got.CharacterControllers[0])
	}
}

func TestFoldStarted_IgnoresIncompleteCharacterControllers(t *testing.T) {
	t.Parallel()

	state, err := foldStarted(State{}, event.Event{
		Type: EventTypeStarted,
		PayloadJSON: mustMarshalSessionJSON(t, StartPayload{
			SessionID:   ids.SessionID("sess-1"),
			SessionName: "Session One",
			CharacterControllers: []CharacterControllerAssignment{
				{CharacterID: ids.CharacterID("char-1"), ParticipantID: ids.ParticipantID("part-1")},
				{CharacterID: ids.CharacterID("char-2")},
				{ParticipantID: ids.ParticipantID("part-3")},
			},
		}),
	})
	if err != nil {
		t.Fatalf("foldStarted() error = %v", err)
	}

	if !state.Started || state.Ended {
		t.Fatalf("state = %#v, want started and not ended", state)
	}
	if len(state.CharacterControllers) != 1 {
		t.Fatalf("len(CharacterControllers) = %d, want 1", len(state.CharacterControllers))
	}
	if got := state.CharacterControllers[ids.CharacterID("char-1")]; got != ids.ParticipantID("part-1") {
		t.Fatalf("CharacterControllers[char-1] = %q, want %q", got, ids.ParticipantID("part-1"))
	}
}

func mustMarshalSessionJSON(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", value, err)
	}
	return data
}
