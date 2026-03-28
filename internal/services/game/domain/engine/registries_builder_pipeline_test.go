package engine

import (
	"errors"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/coredomain"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

type pipelineModule struct {
	id      string
	version string
}

func (m pipelineModule) ID() string                                 { return m.id }
func (m pipelineModule) Version() string                            { return m.version }
func (m pipelineModule) RegisterCommands(_ *command.Registry) error { return nil }
func (m pipelineModule) RegisterEvents(_ *event.Registry) error     { return nil }
func (m pipelineModule) EmittableEventTypes() []event.Type          { return nil }
func (m pipelineModule) Decider() module.Decider                    { return nil }
func (m pipelineModule) Folder() module.Folder                      { return nil }
func (m pipelineModule) StateFactory() module.StateFactory          { return nil }

func TestRunRegistryValidationPipeline_StopsAtFirstError(t *testing.T) {
	bootstrap := newRegistryBootstrap(nil)
	errExpected := errors.New("validation exploded")
	calls := 0

	err := runRegistryValidationPipeline(
		bootstrap,
		func(registryBootstrap) error {
			calls++
			return nil
		},
		func(registryBootstrap) error {
			calls++
			return errExpected
		},
		func(registryBootstrap) error {
			calls++
			return nil
		},
	)
	if !errors.Is(err, errExpected) {
		t.Fatalf("runRegistryValidationPipeline() error = %v, want %v", err, errExpected)
	}
	if calls != 2 {
		t.Fatalf("step calls = %d, want 2", calls)
	}
}

func TestRunNamedRegistryValidationPipeline_WrapsStepNameAndStops(t *testing.T) {
	bootstrap := newRegistryBootstrap(nil)
	errExpected := errors.New("validation exploded")
	calls := 0

	err := runNamedRegistryValidationPipeline(
		bootstrap,
		namedRegistryValidationStep{
			name: "first",
			run: func(registryBootstrap) error {
				calls++
				return nil
			},
		},
		namedRegistryValidationStep{
			name: "second",
			run: func(registryBootstrap) error {
				calls++
				return errExpected
			},
		},
		namedRegistryValidationStep{
			name: "third",
			run: func(registryBootstrap) error {
				calls++
				return nil
			},
		},
	)
	if !errors.Is(err, errExpected) {
		t.Fatalf("runNamedRegistryValidationPipeline() error = %v, want %v", err, errExpected)
	}
	if !strings.Contains(err.Error(), "registry validation second") {
		t.Fatalf("runNamedRegistryValidationPipeline() error = %v, want step name context", err)
	}
	if calls != 2 {
		t.Fatalf("step calls = %d, want 2", calls)
	}
}

func TestRunNamedRegistryValidationPipeline_SkipsNilRun(t *testing.T) {
	bootstrap := newRegistryBootstrap(nil)
	calls := 0

	err := runNamedRegistryValidationPipeline(
		bootstrap,
		namedRegistryValidationStep{name: "nil-step"},
		namedRegistryValidationStep{
			name: "active-step",
			run: func(registryBootstrap) error {
				calls++
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("runNamedRegistryValidationPipeline() unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("step calls = %d, want 1", calls)
	}
}

func TestRunNamedSystemModuleRegistrationPipeline_WrapsStepNameAndStops(t *testing.T) {
	bootstrap := newRegistryBootstrap(nil)
	errExpected := errors.New("registration exploded")
	calls := 0
	mod := pipelineModule{id: "alpha", version: "v2"}

	err := runNamedSystemModuleRegistrationPipeline(
		bootstrap,
		mod,
		moduleRegistrationBaseline{},
		namedSystemModuleRegistrationStep{
			name: "first",
			run: func(registryBootstrap, module.Module, moduleRegistrationBaseline) error {
				calls++
				return nil
			},
		},
		namedSystemModuleRegistrationStep{
			name: "register events",
			run: func(registryBootstrap, module.Module, moduleRegistrationBaseline) error {
				calls++
				return errExpected
			},
		},
		namedSystemModuleRegistrationStep{
			name: "third",
			run: func(registryBootstrap, module.Module, moduleRegistrationBaseline) error {
				calls++
				return nil
			},
		},
	)
	if !errors.Is(err, errExpected) {
		t.Fatalf("runNamedSystemModuleRegistrationPipeline() error = %v, want %v", err, errExpected)
	}
	if !strings.Contains(err.Error(), "system module alpha@v2 register events") {
		t.Fatalf("runNamedSystemModuleRegistrationPipeline() error = %v, want module + step context", err)
	}
	if calls != 2 {
		t.Fatalf("step calls = %d, want 2", calls)
	}
}

func TestRunNamedSystemModuleRegistrationPipeline_SkipsNilRun(t *testing.T) {
	bootstrap := newRegistryBootstrap(nil)
	mod := pipelineModule{id: "alpha", version: "v2"}
	calls := 0

	err := runNamedSystemModuleRegistrationPipeline(
		bootstrap,
		mod,
		moduleRegistrationBaseline{},
		namedSystemModuleRegistrationStep{name: "nil-step"},
		namedSystemModuleRegistrationStep{
			name: "active-step",
			run: func(registryBootstrap, module.Module, moduleRegistrationBaseline) error {
				calls++
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("runNamedSystemModuleRegistrationPipeline() unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("step calls = %d, want 1", calls)
	}
}

func TestModuleVersionLabel_Fallbacks(t *testing.T) {
	if got := moduleVersionLabel(pipelineModule{id: "alpha", version: "v2"}); got != "alpha@v2" {
		t.Fatalf("moduleVersionLabel() = %q, want alpha@v2", got)
	}
	if got := moduleVersionLabel(pipelineModule{version: "v2"}); got != "<unknown>@v2" {
		t.Fatalf("moduleVersionLabel() missing id = %q, want <unknown>@v2", got)
	}
	if got := moduleVersionLabel(nil); got != "<nil>" {
		t.Fatalf("moduleVersionLabel() nil module = %q, want <nil>", got)
	}
}

func TestRegistryBootstrapRegisterCoreDomains_WrapsCommandRegistrationError(t *testing.T) {
	bootstrap := newRegistryBootstrap(nil)
	errExpected := errors.New("register commands failed")

	err := bootstrap.registerCoreDomains([]CoreDomain{
		{
			Contracts: coredomain.Contracts{
				DomainName: "test-domain",
				RegisterCommands: func(*command.Registry) error {
					return errExpected
				},
				RegisterEvents: func(*event.Registry) error { return nil },
			},
		},
	})
	if !errors.Is(err, errExpected) {
		t.Fatalf("registerCoreDomains() error = %v, want %v", err, errExpected)
	}
	if !strings.Contains(err.Error(), "register test-domain commands") {
		t.Fatalf("registerCoreDomains() error = %v, want wrapped domain context", err)
	}
}

func TestRegistryBootstrapRegisterCoreDomains_WrapsEventRegistrationError(t *testing.T) {
	bootstrap := newRegistryBootstrap(nil)
	errExpected := errors.New("register events failed")

	err := bootstrap.registerCoreDomains([]CoreDomain{
		{
			Contracts: coredomain.Contracts{
				DomainName:       "test-domain",
				RegisterCommands: func(*command.Registry) error { return nil },
				RegisterEvents: func(*event.Registry) error {
					return errExpected
				},
			},
		},
	})
	if !errors.Is(err, errExpected) {
		t.Fatalf("registerCoreDomains() error = %v, want %v", err, errExpected)
	}
	if !strings.Contains(err.Error(), "register test-domain events") {
		t.Fatalf("registerCoreDomains() error = %v, want wrapped domain context", err)
	}
}

func TestCollectFoldHandledTypes_IncludesCoreAndModuleTypes(t *testing.T) {
	coreType := event.Type("core.event")
	systemType := event.Type("sys.stub.event")
	foldHandled := collectFoldHandledTypes(
		[]CoreDomain{
			{
				Contracts: coredomain.Contracts{
					DomainName:       "core",
					RegisterCommands: func(*command.Registry) error { return nil },
					RegisterEvents:   func(*event.Registry) error { return nil },
					FoldHandledTypes: func() []event.Type { return []event.Type{coreType} },
				},
			},
		},
		[]module.Module{
			&fakeModuleWithFoldTypes{
				id:          "stub",
				version:     "v1",
				foldHandled: []event.Type{systemType},
			},
		},
	)

	if len(foldHandled) != 2 {
		t.Fatalf("collectFoldHandledTypes() len = %d, want 2 (%v)", len(foldHandled), foldHandled)
	}
	if foldHandled[0] != coreType || foldHandled[1] != systemType {
		t.Fatalf("collectFoldHandledTypes() = %v, want [%s %s]", foldHandled, coreType, systemType)
	}
}

func TestCollectProjectionHandledTypes_SkipsNilDomainFunctions(t *testing.T) {
	types := collectProjectionHandledTypes([]CoreDomain{
		{
			Contracts: coredomain.Contracts{
				DomainName:       "without-projection",
				RegisterCommands: func(*command.Registry) error { return nil },
				RegisterEvents:   func(*event.Registry) error { return nil },
			},
		},
		{
			Contracts: coredomain.Contracts{
				DomainName:             "with-projection",
				RegisterCommands:       func(*command.Registry) error { return nil },
				RegisterEvents:         func(*event.Registry) error { return nil },
				ProjectionHandledTypes: func() []event.Type { return []event.Type{"core.projected"} },
			},
		},
	})
	if len(types) != 1 {
		t.Fatalf("collectProjectionHandledTypes() len = %d, want 1", len(types))
	}
	if types[0] != event.Type("core.projected") {
		t.Fatalf("collectProjectionHandledTypes()[0] = %s, want core.projected", types[0])
	}
}

func TestRegistryBootstrapValidatePayloadValidators_ReportsNonAuditTypes(t *testing.T) {
	bootstrap := newRegistryBootstrap(nil)
	if err := bootstrap.eventRegistry.Register(event.Definition{
		Type:  event.Type("core.missing"),
		Owner: event.OwnerCore,
	}); err != nil {
		t.Fatalf("register missing-validator type: %v", err)
	}
	if err := bootstrap.eventRegistry.Register(event.Definition{
		Type:   event.Type("core.audit"),
		Owner:  event.OwnerCore,
		Intent: event.IntentAuditOnly,
	}); err != nil {
		t.Fatalf("register audit type: %v", err)
	}

	err := bootstrap.validatePayloadValidators()
	if err == nil {
		t.Fatal("expected payload validator error")
	}
	if !strings.Contains(err.Error(), "core.missing") {
		t.Fatalf("validatePayloadValidators() error = %v, want missing type listed", err)
	}
	if strings.Contains(err.Error(), "core.audit") {
		t.Fatalf("validatePayloadValidators() error = %v, did not expect audit-only type", err)
	}
}

func TestBuildRegistries_FailsWhenCoreDomainCommandRegistrationFails(t *testing.T) {
	errExpected := errors.New("core command registration failed")
	_, err := buildRegistries(
		[]CoreDomain{
			{
				Contracts: coredomain.Contracts{
					DomainName: "core",
					RegisterCommands: func(*command.Registry) error {
						return errExpected
					},
					RegisterEvents:      func(*event.Registry) error { return nil },
					EmittableEventTypes: func() []event.Type { return nil },
					FoldHandledTypes:    func() []event.Type { return nil },
				},
			},
		},
		nil,
	)
	if !errors.Is(err, errExpected) {
		t.Fatalf("buildRegistries() error = %v, want %v", err, errExpected)
	}
}

func TestBuildRegistries_FailsCoreEmittableValidationWhenDeclaredTypeIsNotRegistered(t *testing.T) {
	_, err := buildRegistries(
		[]CoreDomain{
			{
				Contracts: coredomain.Contracts{
					DomainName:       "participant-declared",
					RegisterCommands: func(*command.Registry) error { return nil },
					RegisterEvents: func(registry *event.Registry) error {
						return registry.Register(event.Definition{
							Type:            participant.EventTypeSeatReassigned,
							Owner:           event.OwnerCore,
							ValidatePayload: noopValidator,
						})
					},
					EmittableEventTypes: func() []event.Type { return []event.Type{participant.EventTypeJoined} },
					FoldHandledTypes:    func() []event.Type { return nil },
				},
			},
		},
		nil,
	)
	if err == nil {
		t.Fatal("expected core emittable validation error")
	}
	if !strings.Contains(err.Error(), "core emittable event types not in registry") {
		t.Fatalf("buildRegistries() error = %v, want core emittable validation error", err)
	}
}
