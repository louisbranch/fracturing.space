package engine

import (
	"errors"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

type failingRegisterModule struct {
	id         string
	version    string
	commandErr error
	eventErr   error
}

func (m failingRegisterModule) ID() string      { return m.id }
func (m failingRegisterModule) Version() string { return m.version }
func (m failingRegisterModule) RegisterCommands(_ *command.Registry) error {
	return m.commandErr
}
func (m failingRegisterModule) RegisterEvents(_ *event.Registry) error {
	return m.eventErr
}
func (m failingRegisterModule) EmittableEventTypes() []event.Type { return nil }
func (m failingRegisterModule) Decider() module.Decider           { return nil }
func (m failingRegisterModule) Folder() module.Folder             { return nil }
func (m failingRegisterModule) StateFactory() module.StateFactory { return nil }

func TestBuildRegistries_RejectsNilModule(t *testing.T) {
	_, err := BuildRegistries(nil)
	if err == nil {
		t.Fatal("expected error for nil module")
	}
	if !strings.Contains(err.Error(), "system module is required") {
		t.Fatalf("expected nil module error, got %v", err)
	}
}

func TestBuildRegistries_PropagatesModuleCommandRegistrationError(t *testing.T) {
	commandErr := errors.New("register commands boom")
	_, err := BuildRegistries(failingRegisterModule{
		id:         "system-bad",
		version:    "v1",
		commandErr: commandErr,
	})
	if !errors.Is(err, commandErr) {
		t.Fatalf("expected command registration error, got %v", err)
	}
}

func TestBuildRegistries_PropagatesModuleEventRegistrationError(t *testing.T) {
	eventErr := errors.New("register events boom")
	_, err := BuildRegistries(failingRegisterModule{
		id:       "system-bad",
		version:  "v1",
		eventErr: eventErr,
	})
	if !errors.Is(err, eventErr) {
		t.Fatalf("expected event registration error, got %v", err)
	}
}

func TestValidateCoreEmittableEventTypes_FailsOnMissingRegistration(t *testing.T) {
	events := event.NewRegistry()
	err := validateCoreEmittableEventTypes(events)
	if err == nil {
		t.Fatal("expected missing core emittable event type error")
	}
	if !strings.Contains(err.Error(), "core emittable event types not in registry") {
		t.Fatalf("unexpected error: %v", err)
	}
}
