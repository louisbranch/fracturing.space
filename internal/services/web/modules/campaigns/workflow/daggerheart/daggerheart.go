// Package daggerheart implements the Daggerheart-specific character creation
// workflow behind the CharacterCreationWorkflow interface.
package daggerheart

import (
	"strings"
)

// SystemLabel is the case-insensitive system label used to resolve this workflow.
const SystemLabel = "Daggerheart"

// IsDaggerheartSystem returns true when the campaign system label matches Daggerheart.
func IsDaggerheartSystem(system string) bool {
	return strings.EqualFold(strings.TrimSpace(system), SystemLabel)
}

// Workflow implements campaigns.CharacterCreationWorkflow for Daggerheart.
type Workflow struct{}

// New returns a new Daggerheart workflow implementation.
func New() Workflow { return Workflow{} }
