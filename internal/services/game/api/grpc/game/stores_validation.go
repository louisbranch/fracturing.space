package game

import (
	"fmt"
	"strings"
)

// Validate checks that every store field is non-nil and eagerly builds the
// adapter registry. Call this at service construction time so that handlers
// do not need per-method nil guards and adapter registration errors surface
// at startup instead of at runtime.
func (s *Stores) Validate() error {
	var missing []string
	missing = appendMissingRequirements(missing, s.projectionRequirements()...)
	missing = appendMissingRequirements(missing, s.infrastructureRequirements()...)
	missing = appendMissingRequirements(missing, s.contentRequirements()...)
	missing = appendMissingRequirements(missing, s.runtimeRequirements()...)
	if len(missing) > 0 {
		return fmt.Errorf("stores not configured: %s", strings.Join(missing, ", "))
	}

	adapters, err := TryAdapterRegistryForSystemStores(s.SystemStores)
	if err != nil {
		return fmt.Errorf("build adapter registry: %w", err)
	}
	s.adapters = adapters

	applier, err := s.TryApplier()
	if err != nil {
		return fmt.Errorf("build projection applier: %w", err)
	}
	if err := applier.ValidateStorePreconditions(); err != nil {
		return err
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

func (s Stores) projectionRequirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "Campaign", configured: s.Campaign != nil},
		{name: "Participant", configured: s.Participant != nil},
		{name: "ClaimIndex", configured: s.ClaimIndex != nil},
		{name: "Invite", configured: s.Invite != nil},
		{name: "Character", configured: s.Character != nil},
		{name: "SystemStores.Daggerheart", configured: s.SystemStores.Daggerheart != nil},
		{name: "Session", configured: s.Session != nil},
		{name: "SessionGate", configured: s.SessionGate != nil},
		{name: "SessionSpotlight", configured: s.SessionSpotlight != nil},
		{name: "SessionInteraction", configured: s.SessionInteraction != nil},
		{name: "Scene", configured: s.Scene != nil},
		{name: "SceneCharacter", configured: s.SceneCharacter != nil},
		{name: "SceneGate", configured: s.SceneGate != nil},
		{name: "SceneSpotlight", configured: s.SceneSpotlight != nil},
		{name: "SceneInteraction", configured: s.SceneInteraction != nil},
		{name: "CampaignFork", configured: s.CampaignFork != nil},
	}
}

func (s Stores) infrastructureRequirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "Event", configured: s.Event != nil},
		{name: "Audit", configured: s.Audit != nil},
		{name: "Statistics", configured: s.Statistics != nil},
		{name: "Snapshot", configured: s.Snapshot != nil},
	}
}

func (s Stores) contentRequirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "DaggerheartContent", configured: s.DaggerheartContent != nil},
	}
}

func (s Stores) runtimeRequirements() []dependencyRequirement {
	return []dependencyRequirement{
		{name: "Write.Executor", configured: s.Write.Executor != nil},
		{name: "Write.Runtime", configured: s.Write.Runtime != nil},
		{name: "Events", configured: s.Events != nil},
	}
}
