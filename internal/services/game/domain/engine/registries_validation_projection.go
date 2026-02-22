package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

// ValidateProjectionRegistries bundles the three projection coverage validators
// into a single call so that all startup paths (server, maintenance CLI, replay)
// run the same checks without scattering calls across bootstrap functions.
func ValidateProjectionRegistries(
	events *event.Registry,
	modules *module.Registry,
	adapters *bridge.AdapterRegistry,
	projectionHandledTypes []event.Type,
) error {
	if err := ValidateProjectionCoverage(events, projectionHandledTypes); err != nil {
		return fmt.Errorf("validate projection coverage: %w", err)
	}
	if err := ValidateNoProjectionHandlersForNonProjectionEvents(events, projectionHandledTypes); err != nil {
		return fmt.Errorf("validate projection intent guard: %w", err)
	}
	if err := ValidateNoStaleProjectionHandlers(events, projectionHandledTypes); err != nil {
		return fmt.Errorf("validate stale projection handlers: %w", err)
	}
	if err := ValidateAdapterEventCoverage(modules, adapters, events); err != nil {
		return fmt.Errorf("validate adapter event coverage: %w", err)
	}
	// Collect core domain projection declarations for alignment check.
	var coreProjectionDeclared []event.Type
	for _, domain := range CoreDomains() {
		if domain.ProjectionHandledTypes != nil {
			coreProjectionDeclared = append(coreProjectionDeclared, domain.ProjectionHandledTypes()...)
		}
	}
	if err := ValidateCoreProjectionDeclarations(events, coreProjectionDeclared); err != nil {
		return fmt.Errorf("validate core projection declarations: %w", err)
	}
	return nil
}

// ValidateProjectionCoverage verifies that every core IntentProjectionAndReplay
// event has a projection handler declared via ProjectionHandledTypes.
func ValidateProjectionCoverage(events *event.Registry, handledTypes []event.Type) error {
	if events == nil {
		return fmt.Errorf("event registry is required for projection coverage validation")
	}

	handled := make(map[event.Type]struct{})
	for _, t := range handledTypes {
		handled[t] = struct{}{}
	}

	var missing []string
	for _, def := range events.ListDefinitions() {
		if def.Owner != event.OwnerCore {
			continue
		}
		if def.Intent != event.IntentProjectionAndReplay {
			continue
		}
		// A type is covered if it is handled directly or if the registry
		// resolves it (via alias) to a handled canonical type.
		if _, ok := handled[def.Type]; ok {
			continue
		}
		if resolved := events.Resolve(def.Type); resolved != def.Type {
			if _, ok := handled[resolved]; ok {
				continue
			}
		}
		missing = append(missing, string(def.Type))
	}
	if len(missing) > 0 {
		return fmt.Errorf("core projection-and-replay events missing projection handlers: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateCoreProjectionDeclarations verifies that every core event type with
// IntentProjectionAndReplay is declared by some CoreDomain's
// ProjectionHandledTypes, and conversely that every declared type is
// registered with projection intent.
func ValidateCoreProjectionDeclarations(events *event.Registry, declared []event.Type) error {
	if events == nil {
		return fmt.Errorf("event registry is required for core projection declaration validation")
	}

	declaredSet := make(map[event.Type]struct{}, len(declared))
	for _, t := range declared {
		declaredSet[t] = struct{}{}
	}

	// Forward check: every core projection-and-replay event must be declared.
	registered := make(map[event.Type]struct{})
	var missing []string
	for _, def := range events.ListDefinitions() {
		if def.Owner != event.OwnerCore {
			continue
		}
		if def.Intent != event.IntentProjectionAndReplay {
			continue
		}
		registered[def.Type] = struct{}{}
		if _, ok := declaredSet[def.Type]; ok {
			continue
		}
		// Check if the registry resolves it via alias.
		if resolved := events.Resolve(def.Type); resolved != def.Type {
			if _, ok := declaredSet[resolved]; ok {
				continue
			}
		}
		missing = append(missing, string(def.Type))
	}
	if len(missing) > 0 {
		return fmt.Errorf("core projection events missing domain ProjectionHandledTypes declarations: %s",
			strings.Join(missing, ", "))
	}

	// Reverse check: every declared type must be a registered projection event.
	var stale []string
	for _, t := range declared {
		if _, ok := registered[t]; !ok {
			stale = append(stale, string(t))
		}
	}
	if len(stale) > 0 {
		return fmt.Errorf("stale core ProjectionHandledTypes declarations without registration: %s",
			strings.Join(stale, ", "))
	}
	return nil
}

// ValidateNoStaleProjectionHandlers verifies that every event type in the
// projection handler list is actually registered in the event registry (after
// alias resolution). A stale handler — one left behind after an event type is
// removed or renamed — would be dead code that silently misleads developers.
func ValidateNoStaleProjectionHandlers(events *event.Registry, projectionHandled []event.Type) error {
	if events == nil {
		return fmt.Errorf("event registry is required for stale projection handler check")
	}

	var stale []string
	for _, t := range projectionHandled {
		resolved := events.Resolve(t)
		if _, ok := events.Definition(resolved); !ok {
			stale = append(stale, string(t))
		}
	}
	if len(stale) > 0 {
		return fmt.Errorf("projection handlers for unregistered event types (stale): %s",
			strings.Join(stale, ", "))
	}
	return nil
}

// ValidateNoProjectionHandlersForNonProjectionEvents verifies that no
// projection handler exists for an event with IntentAuditOnly or
// IntentReplayOnly. Such handlers would be dead code — the projection
// applier skips non-projection events at runtime.
func ValidateNoProjectionHandlersForNonProjectionEvents(events *event.Registry, projectionHandled []event.Type) error {
	if events == nil {
		return fmt.Errorf("event registry is required for projection intent guard")
	}

	var dead []string
	for _, t := range projectionHandled {
		def, ok := events.Definition(t)
		if !ok {
			continue
		}
		if def.Intent == event.IntentAuditOnly || def.Intent == event.IntentReplayOnly {
			dead = append(dead, string(t))
		}
	}
	if len(dead) > 0 {
		return fmt.Errorf("projection handlers registered for non-projection events (dead code): %s",
			strings.Join(dead, ", "))
	}
	return nil
}
