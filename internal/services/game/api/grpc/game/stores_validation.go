package game

import (
	"fmt"
	"strings"
)

// ValidateRootStoreGroups checks that every root store concern is configured
// before service construction so handlers do not need per-method nil guards.
func ValidateRootStoreGroups(
	projection ProjectionStores,
	systemStores SystemStores,
	infrastructure InfrastructureStores,
	content ContentStores,
	runtime RuntimeStores,
) error {
	var missing []string
	missing = appendMissingRequirements(missing, projection.requirements()...)
	missing = appendMissingRequirements(missing, dependencyRequirement{
		name:       "SystemStores.Daggerheart",
		configured: systemStores.Daggerheart != nil,
	})
	missing = appendMissingRequirements(missing, infrastructure.requirements()...)
	missing = appendMissingRequirements(missing, content.requirements()...)
	missing = appendMissingRequirements(missing, runtime.requirements()...)
	if len(missing) > 0 {
		return fmt.Errorf("stores not configured: %s", strings.Join(missing, ", "))
	}
	return nil
}

type dependencyRequirement struct {
	name       string
	configured bool
}

func appendMissingRequirements(missing []string, requirements ...dependencyRequirement) []string {
	for _, requirement := range requirements {
		if !requirement.configured {
			missing = append(missing, requirement.name)
		}
	}
	return missing
}

func (s ProjectionStores) requirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "Campaign", configured: s.Campaign != nil},
		{name: "Participant", configured: s.Participant != nil},
		{name: "ClaimIndex", configured: s.ClaimIndex != nil},
		{name: "Character", configured: s.Character != nil},
		{name: "Session", configured: s.Session != nil},
		{name: "SessionRecap", configured: s.SessionRecap != nil},
		{name: "SessionGate", configured: s.SessionGate != nil},
		{name: "SessionSpotlight", configured: s.SessionSpotlight != nil},
		{name: "SessionInteraction", configured: s.SessionInteraction != nil},
		{name: "Scene", configured: s.Scene != nil},
		{name: "SceneCharacter", configured: s.SceneCharacter != nil},
		{name: "SceneGate", configured: s.SceneGate != nil},
		{name: "SceneSpotlight", configured: s.SceneSpotlight != nil},
		{name: "SceneInteraction", configured: s.SceneInteraction != nil},
		{name: "SceneGMInteraction", configured: s.SceneGMInteraction != nil},
		{name: "CampaignFork", configured: s.CampaignFork != nil},
	}
}

func (s InfrastructureStores) requirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "Event", configured: s.Event != nil},
		{name: "Watermarks", configured: s.Watermarks != nil},
		{name: "Audit", configured: s.Audit != nil},
		{name: "Statistics", configured: s.Statistics != nil},
		{name: "Snapshot", configured: s.Snapshot != nil},
	}
}

func (s ContentStores) requirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "DaggerheartContent", configured: s.DaggerheartContent != nil},
	}
}

func (s RuntimeStores) requirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "Write.Executor", configured: s.Write.Executor != nil},
		{name: "Write.Runtime", configured: s.Write.Runtime != nil},
	}
}
