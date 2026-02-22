package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// ValidateAggregateFoldDispatch verifies that every core event type declared
// in CoreDomains().FoldHandledTypes is actually wired into the aggregate
// applier's fold dispatch sets.
func ValidateAggregateFoldDispatch(events *event.Registry) error {
	if events == nil {
		return fmt.Errorf("event registry is required for aggregate fold dispatch validation")
	}

	applier := &aggregate.Folder{}
	dispatched := make(map[event.Type]struct{})
	for _, t := range applier.FoldDispatchedTypes() {
		dispatched[t] = struct{}{}
	}

	declared := make(map[event.Type]struct{})
	for _, domain := range CoreDomains() {
		for _, t := range domain.FoldHandledTypes() {
			declared[t] = struct{}{}
		}
	}

	var missing []string
	for t := range declared {
		if _, ok := dispatched[t]; !ok {
			missing = append(missing, string(t))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("core fold types declared but not dispatched by aggregate applier: %s",
			strings.Join(missing, ", "))
	}
	return nil
}
