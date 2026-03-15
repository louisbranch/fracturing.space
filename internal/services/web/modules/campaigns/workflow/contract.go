// Package workflow defines transport-owned character creation workflow seams.
package workflow

import (
	"net/url"
	"strings"
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

// Installation describes one install-time workflow registration.
type Installation struct {
	ID                string
	Aliases           []string
	CharacterCreation CharacterCreation
}

// Registry maps normalized system labels to transport-owned workflow implementations.
type Registry map[string]CharacterCreation

// Install builds a workflow registry from install-time manifests instead of a
// hardcoded switch in the workflow service layer.
func Install(installs ...Installation) Registry {
	if len(installs) == 0 {
		return nil
	}
	registry := Registry{}
	for _, install := range installs {
		if install.CharacterCreation == nil {
			continue
		}
		if normalizedID := normalizeSystemLabel(install.ID); normalizedID != "" {
			registry[normalizedID] = install.CharacterCreation
		}
		for _, alias := range install.Aliases {
			if normalizedAlias := normalizeSystemLabel(alias); normalizedAlias != "" {
				registry[normalizedAlias] = install.CharacterCreation
			}
		}
	}
	if len(registry) == 0 {
		return nil
	}
	return registry
}

// Resolve returns the installed workflow implementation for one system label.
func (r Registry) Resolve(system string) CharacterCreation {
	if r == nil {
		return nil
	}
	return r[normalizeSystemLabel(system)]
}

// normalizeSystemLabel canonicalizes install-time workflow IDs and aliases so
// transport lookups stay case- and whitespace-insensitive.
func normalizeSystemLabel(system string) string {
	return strings.ToLower(strings.TrimSpace(system))
}
