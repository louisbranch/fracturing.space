// Package workflow defines transport-owned character creation workflow seams.
package workflow

import (
	"net/url"
)

// CharacterCreation combines system-specific domain assembly with transport
// parsing for one game-system workflow.
type CharacterCreation interface {
	BuildView(
		progress Progress,
		catalog Catalog,
		profile Profile,
	) CharacterCreationView
	ParseStepInput(form url.Values, nextStep int32) (*StepInput, error)
}

// Registry maps canonical game-system identifiers to transport-owned workflow
// implementations.
type Registry = map[GameSystem]CharacterCreation
