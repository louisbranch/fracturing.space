package projection

import (
	"sort"
	"testing"
)

// TestRegisteredHandlerTypes_MatchesProjectionHandledTypes verifies that the
// handler registry map contains exactly the same event types as the manual
// ProjectionHandledTypes() list. This bridge test ensures the two remain in
// sync before ProjectionHandledTypes() is refactored to delegate to the map.
func TestRegisteredHandlerTypes_MatchesProjectionHandledTypes(t *testing.T) {
	registered := registeredHandlerTypes()
	if len(registered) == 0 {
		t.Fatal("registeredHandlerTypes() returned empty list")
	}

	manual := ProjectionHandledTypes()
	if len(manual) == 0 {
		t.Fatal("ProjectionHandledTypes() returned empty list")
	}

	// Sort both for comparison.
	sortedManual := make([]string, len(manual))
	for i, et := range manual {
		sortedManual[i] = string(et)
	}
	sort.Strings(sortedManual)

	sortedRegistry := make([]string, len(registered))
	for i, et := range registered {
		sortedRegistry[i] = string(et)
	}
	sort.Strings(sortedRegistry)

	if len(sortedManual) != len(sortedRegistry) {
		t.Fatalf("ProjectionHandledTypes() has %d entries, registeredHandlerTypes() has %d",
			len(sortedManual), len(sortedRegistry))
	}

	for i := range sortedManual {
		if sortedManual[i] != sortedRegistry[i] {
			t.Errorf("mismatch at index %d: manual=%s registry=%s", i, sortedManual[i], sortedRegistry[i])
		}
	}
}

// TestHandlerRegistry_AllEntriesHaveApply verifies that every entry in the
// handler registry has a non-nil apply function.
func TestHandlerRegistry_AllEntriesHaveApply(t *testing.T) {
	for et, h := range handlers {
		if h.apply == nil {
			t.Errorf("handler for %s has nil apply function", et)
		}
	}
}
