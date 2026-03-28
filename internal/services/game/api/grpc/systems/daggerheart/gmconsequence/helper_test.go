package gmconsequence

import (
	"context"
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// fakeGateStore satisfies SessionGateStore for tests.
type fakeGateStore struct {
	gate storage.SessionGate
	err  error
}

func (f *fakeGateStore) GetOpenSessionGate(_ context.Context, _, _ string) (storage.SessionGate, error) {
	return f.gate, f.err
}

// fakeSpotlightStore satisfies SessionSpotlightStore for tests.
type fakeSpotlightStore struct {
	spotlight storage.SessionSpotlight
	err       error
}

func (f *fakeSpotlightStore) GetSessionSpotlight(_ context.Context, _, _ string) (storage.SessionSpotlight, error) {
	return f.spotlight, f.err
}

// recordedCommand captures one ExecuteCoreCommand invocation.
type recordedCommand struct {
	input CoreCommandInput
}

// fakeExecutor records calls and optionally returns an error.
type fakeExecutor struct {
	calls []recordedCommand
	err   error
}

func (f *fakeExecutor) execute(_ context.Context, in CoreCommandInput) error {
	f.calls = append(f.calls, recordedCommand{input: in})
	return f.err
}

func TestOpen_NoGateNeeded_NoSpotlightNeeded(t *testing.T) {
	// Gate is already open and spotlight is already GM with no character —
	// no repair commands should be executed.
	exec := &fakeExecutor{}
	deps := Dependencies{
		SessionGate: &fakeGateStore{
			gate: storage.SessionGate{GateID: "existing-gate"},
			err:  nil,
		},
		SessionSpotlight: &fakeSpotlightStore{
			spotlight: storage.SessionSpotlight{
				SpotlightType: session.SpotlightTypeGM,
				CharacterID:   "",
			},
			err: nil,
		},
		ExecuteCoreCommand: exec.execute,
	}

	err := Open(context.Background(), deps, "camp-1", "sess-1", "scene-1", "req-1", "inv-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.calls) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(exec.calls))
	}
}

func TestOpen_GateNeeded(t *testing.T) {
	// No gate is currently open — helper should open one.
	// Spotlight is already GM with no character — no spotlight command needed.
	exec := &fakeExecutor{}
	deps := Dependencies{
		SessionGate: &fakeGateStore{err: storage.ErrNotFound},
		SessionSpotlight: &fakeSpotlightStore{
			spotlight: storage.SessionSpotlight{
				SpotlightType: session.SpotlightTypeGM,
				CharacterID:   "",
			},
			err: nil,
		},
		ExecuteCoreCommand: exec.execute,
	}

	err := Open(context.Background(), deps, "camp-1", "sess-1", "scene-1", "req-1", "inv-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 command, got %d", len(exec.calls))
	}
	if exec.calls[0].input.EntityType != "session_gate" {
		t.Fatalf("expected session_gate entity, got %q", exec.calls[0].input.EntityType)
	}
}

func TestOpen_SpotlightNeeded(t *testing.T) {
	// Gate is already open — no gate command needed.
	// Spotlight is on a character — helper should set GM spotlight.
	exec := &fakeExecutor{}
	deps := Dependencies{
		SessionGate: &fakeGateStore{
			gate: storage.SessionGate{GateID: "existing-gate"},
			err:  nil,
		},
		SessionSpotlight: &fakeSpotlightStore{
			spotlight: storage.SessionSpotlight{
				SpotlightType: session.SpotlightTypeCharacter,
				CharacterID:   "char-1",
			},
			err: nil,
		},
		ExecuteCoreCommand: exec.execute,
	}

	err := Open(context.Background(), deps, "camp-1", "sess-1", "scene-1", "req-1", "inv-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 command, got %d", len(exec.calls))
	}
	if exec.calls[0].input.EntityType != "session_spotlight" {
		t.Fatalf("expected session_spotlight entity, got %q", exec.calls[0].input.EntityType)
	}
}

func TestOpen_BothGateAndSpotlightNeeded(t *testing.T) {
	// No gate is open and spotlight needs clearing — both commands should execute.
	exec := &fakeExecutor{}
	deps := Dependencies{
		SessionGate: &fakeGateStore{err: storage.ErrNotFound},
		SessionSpotlight: &fakeSpotlightStore{
			spotlight: storage.SessionSpotlight{
				SpotlightType: session.SpotlightTypeCharacter,
				CharacterID:   "char-1",
			},
			err: nil,
		},
		ExecuteCoreCommand: exec.execute,
	}

	err := Open(context.Background(), deps, "camp-1", "sess-1", "scene-1", "req-1", "inv-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.calls) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(exec.calls))
	}
	if exec.calls[0].input.EntityType != "session_gate" {
		t.Fatalf("first command entity = %q, want session_gate", exec.calls[0].input.EntityType)
	}
	if exec.calls[1].input.EntityType != "session_spotlight" {
		t.Fatalf("second command entity = %q, want session_spotlight", exec.calls[1].input.EntityType)
	}
}

func TestOpen_ExecuteCoreCommandError(t *testing.T) {
	// Gate needs opening but the executor returns an error — must propagate.
	execErr := errors.New("command executor boom")
	exec := &fakeExecutor{err: execErr}
	deps := Dependencies{
		SessionGate:        &fakeGateStore{err: storage.ErrNotFound},
		SessionSpotlight:   &fakeSpotlightStore{err: storage.ErrNotFound},
		ExecuteCoreCommand: exec.execute,
	}

	err := Open(context.Background(), deps, "camp-1", "sess-1", "scene-1", "req-1", "inv-1", nil)
	if err == nil {
		t.Fatal("expected error to propagate")
	}
	// Invariant: the executor error must surface without being swallowed.
	if !errors.Is(err, execErr) {
		t.Fatalf("expected executor error in chain, got %v", err)
	}
}
