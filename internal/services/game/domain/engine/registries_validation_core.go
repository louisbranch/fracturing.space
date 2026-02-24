package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// validateCoreEmittableEventTypes ensures every event type a core domain
// decider declares as emittable is registered in the event registry.
func validateCoreEmittableEventTypes(events *event.Registry) error {
	var missing []string
	for _, domain := range CoreDomains() {
		for _, t := range domain.EmittableEventTypes() {
			if _, ok := events.Definition(t); !ok {
				missing = append(missing, string(t))
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("core emittable event types not in registry: %s",
			strings.Join(missing, ", "))
	}
	return nil
}

// ValidateFoldCoverage verifies that every core event with IntentProjectionAndReplay
// or IntentReplayOnly has a fold handler declared via FoldHandledTypes in the domain
// packages.
//
// This is a startup-time safety check: if a developer adds a new event that affects
// aggregate state and forgets the fold case, the server refuses to start.
func ValidateFoldCoverage(events *event.Registry) error {
	if events == nil {
		return fmt.Errorf("event registry is required for fold coverage validation")
	}

	handled := make(map[event.Type]struct{})
	for _, domain := range CoreDomains() {
		for _, t := range domain.FoldHandledTypes() {
			handled[t] = struct{}{}
		}
	}

	var missing []string
	for _, def := range events.ListDefinitions() {
		if def.Owner != event.OwnerCore {
			continue
		}
		if def.Intent != event.IntentProjectionAndReplay && def.Intent != event.IntentReplayOnly {
			continue
		}
		if _, ok := handled[def.Type]; !ok {
			missing = append(missing, string(def.Type))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("core replay events missing fold handlers: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateNoFoldHandlersForAuditOnlyEvents verifies that no fold handler
// exists for an event with IntentAuditOnly. Such a handler would be dead
// code — the aggregate folder skips audit-only events at runtime, so a
// handler would never execute.
func ValidateNoFoldHandlersForAuditOnlyEvents(events *event.Registry, foldHandled []event.Type) error {
	if events == nil {
		return fmt.Errorf("event registry is required for audit-only fold guard")
	}

	var dead []string
	for _, t := range foldHandled {
		def, ok := events.Definition(t)
		if !ok {
			continue
		}
		if def.Intent == event.IntentAuditOnly {
			dead = append(dead, string(t))
		}
	}
	if len(dead) > 0 {
		return fmt.Errorf("fold handlers registered for audit-only events (dead code): %s",
			strings.Join(dead, ", "))
	}
	return nil
}

// ValidateEntityKeyedAddressing verifies addressing consistency within each
// core domain. A domain is entity-keyed when ANY of its registered
// FoldHandledTypes have AddressingPolicyEntityTarget. Once identified as
// entity-keyed, ALL fold types in that domain must have the same policy.
func ValidateEntityKeyedAddressing(events *event.Registry) error {
	if events == nil {
		return fmt.Errorf("event registry is required for entity-keyed addressing validation")
	}

	var missing []string
	for _, domain := range CoreDomains() {
		types := domain.FoldHandledTypes()

		// Check if any registered fold type uses entity addressing.
		hasEntityAddressing := false
		for _, t := range types {
			def, ok := events.Definition(t)
			if !ok {
				continue
			}
			if def.Addressing == event.AddressingPolicyEntityTarget {
				hasEntityAddressing = true
				break
			}
		}
		if !hasEntityAddressing {
			continue
		}

		// Domain is entity-keyed — every registered fold type must use entity addressing.
		for _, t := range types {
			def, ok := events.Definition(t)
			if !ok {
				continue
			}
			if def.Addressing != event.AddressingPolicyEntityTarget {
				missing = append(missing, string(t))
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("entity-keyed fold types missing AddressingPolicyEntityTarget: %s",
			strings.Join(missing, ", "))
	}
	return nil
}

// ValidateAliasFoldCoverage verifies that every alias target type has a fold
// handler in the core domain dispatch table. An alias that resolves to a type
// with no fold handler would silently ignore legacy events after alias
// resolution (A1), creating silent state divergence.
func ValidateAliasFoldCoverage(events *event.Registry) error {
	if events == nil {
		return fmt.Errorf("event registry is required for alias fold coverage validation")
	}

	aliases := events.ListAliases()
	if len(aliases) == 0 {
		return nil
	}

	// Build set of all fold-handled types from core domains.
	handled := make(map[event.Type]struct{})
	for _, domain := range CoreDomains() {
		for _, t := range domain.FoldHandledTypes() {
			handled[t] = struct{}{}
		}
	}

	var missing []string
	for _, canonical := range aliases {
		def, ok := events.Definition(canonical)
		if !ok {
			continue
		}
		// Only check types that should be folded (skip audit-only).
		if def.Intent == event.IntentAuditOnly {
			continue
		}
		if _, ok := handled[canonical]; !ok {
			missing = append(missing, string(canonical))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("alias target types missing fold handlers: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateCoreDeciderCommandCoverage verifies that every core-owned command
// type in the command registry is claimed by some CoreDomain's
// DeciderHandledCommands, and conversely that every declared handler has a
// matching registration. This is the core-domain counterpart of
// ValidateDeciderCommandCoverage for system modules.
func ValidateCoreDeciderCommandCoverage(commands *command.Registry) error {
	if commands == nil {
		return fmt.Errorf("command registry is required for core decider coverage validation")
	}

	// Collect all types each core domain declares its decider handles.
	declared := make(map[command.Type]struct{})
	for _, domain := range CoreDomains() {
		if domain.DeciderHandledCommands == nil {
			continue
		}
		for _, t := range domain.DeciderHandledCommands() {
			declared[t] = struct{}{}
		}
	}

	// Forward check: every registered core command must be in declared set.
	registered := make(map[command.Type]struct{})
	var missing []string
	for _, def := range commands.ListDefinitions() {
		if def.Owner != command.OwnerCore {
			continue
		}
		registered[def.Type] = struct{}{}
		if _, ok := declared[def.Type]; !ok {
			missing = append(missing, string(def.Type))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("core commands missing decider handlers: %s", strings.Join(missing, ", "))
	}

	// Reverse check: every declared handler must have a registration.
	var stale []string
	for t := range declared {
		if _, ok := registered[t]; !ok {
			stale = append(stale, string(t))
		}
	}
	if len(stale) > 0 {
		return fmt.Errorf("stale core decider handler declarations without registration: %s",
			strings.Join(stale, ", "))
	}
	return nil
}
