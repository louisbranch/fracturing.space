package engine

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

// ValidateActiveSessionPolicyCoverage ensures every core command type has an
// active-session policy classification.
func ValidateActiveSessionPolicyCoverage(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	definitions := registry.ListDefinitions()
	missing := make([]string, 0)
	for _, definition := range definitions {
		if definition.Owner != command.OwnerCore {
			continue
		}
		if definition.ActiveSession.Classification != "" {
			continue
		}
		missing = append(missing, string(definition.Type))
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("active-session policy missing core command types: %s", strings.Join(missing, ", "))
}
