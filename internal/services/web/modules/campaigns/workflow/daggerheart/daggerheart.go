// Package daggerheart implements the Daggerheart-specific character-creation
// workflow behind the workflow.CharacterCreation contract.
package daggerheart

import (
	"strings"

	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
)

// SystemLabel is the case-insensitive system label used to resolve this workflow.
const SystemLabel = "Daggerheart"

// IsDaggerheartSystem returns true when the campaign system label matches Daggerheart.
func IsDaggerheartSystem(system string) bool {
	return strings.EqualFold(strings.TrimSpace(system), SystemLabel)
}

// Workflow implements workflow.CharacterCreation for Daggerheart.
type Workflow struct {
	AssetBaseURL string
}

// New returns a new Daggerheart workflow implementation.
func New(assetBaseURL string) Workflow { return Workflow{AssetBaseURL: assetBaseURL} }

// Install returns the install-time workflow manifest for Daggerheart.
func Install(assetBaseURL string) campaignworkflow.Installation {
	return campaignworkflow.Installation{
		ID:                strings.ToLower(SystemLabel),
		Aliases:           []string{SystemLabel},
		CharacterCreation: New(assetBaseURL),
	}
}
