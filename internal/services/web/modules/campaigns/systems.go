package campaigns

import (
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
)

// CampaignCreateSystemOption keeps campaign-create system choices transport-owned.
type CampaignCreateSystemOption struct {
	Value    string
	LabelKey string
}

// campaignSystemInstall captures one transport-owned game-system manifest entry
// before it is normalized into create-form aliases and workflow registrations.
type campaignSystemInstall struct {
	ID                campaignapp.GameSystem
	Aliases           []string
	CreateLabelKey    string
	DefaultCreate     bool
	CharacterCreation campaignworkflow.CharacterCreation
}

// campaignSystemRegistry keeps campaign-create system aliases, defaults, and
// installed workflows transport-owned in one place.
type campaignSystemRegistry struct {
	aliases       map[string]campaignapp.GameSystem
	createOptions []CampaignCreateSystemOption
	defaultCreate campaignapp.GameSystem
	workflows     campaignworkflow.Registry
}

// newCampaignSystemRegistry normalizes install-time system manifests into the
// alias and workflow registry used by create-form transport.
func newCampaignSystemRegistry(installs ...campaignSystemInstall) campaignSystemRegistry {
	registry := campaignSystemRegistry{
		aliases:       map[string]campaignapp.GameSystem{},
		createOptions: make([]CampaignCreateSystemOption, 0, len(installs)),
	}
	workflowInstalls := make([]campaignworkflow.Installation, 0, len(installs))
	for _, install := range installs {
		canonical := strings.TrimSpace(string(install.ID))
		if canonical == "" {
			continue
		}
		registry.createOptions = append(registry.createOptions, CampaignCreateSystemOption{
			Value:    canonical,
			LabelKey: strings.TrimSpace(install.CreateLabelKey),
		})
		if registry.defaultCreate == "" || install.DefaultCreate {
			registry.defaultCreate = install.ID
		}
		registry.aliases[normalizeCampaignSystemLabel(canonical)] = install.ID
		for _, alias := range install.Aliases {
			if normalized := normalizeCampaignSystemLabel(alias); normalized != "" {
				registry.aliases[normalized] = install.ID
			}
		}
		if install.CharacterCreation != nil {
			workflowInstalls = append(workflowInstalls, campaignworkflow.Installation{
				ID:                canonical,
				Aliases:           install.Aliases,
				CharacterCreation: install.CharacterCreation,
			})
		}
	}
	registry.workflows = campaignworkflow.Install(workflowInstalls...)
	return registry
}

// newCampaignSystemsFromWorkflows preserves concise package-test wiring while
// still routing production ownership through install-time manifests.
func newCampaignSystemsFromWorkflows(workflows ...campaignworkflow.Registry) campaignSystemRegistry {
	var creation campaignworkflow.CharacterCreation
	if len(workflows) > 0 && workflows[0] != nil {
		creation = workflows[0].Resolve(string(campaignapp.GameSystemDaggerheart))
	}
	return newCampaignSystemRegistry(campaignSystemInstall{
		ID:                campaignapp.GameSystemDaggerheart,
		Aliases:           []string{"Daggerheart", "game_system_daggerheart"},
		CreateLabelKey:    "game.create.field_system_value_daggerheart",
		DefaultCreate:     true,
		CharacterCreation: creation,
	})
}

// defaultCreateSystem returns the canonical create-form default without
// exposing app-layer system enums to transport callers.
func (r campaignSystemRegistry) defaultCreateSystem() string {
	return strings.TrimSpace(string(r.defaultCreate))
}

// parseCreateSystem resolves a submitted create-form system value through the
// transport-owned alias map.
func (r campaignSystemRegistry) parseCreateSystem(value string) (campaignapp.GameSystem, bool) {
	normalized := normalizeCampaignSystemLabel(value)
	if normalized == "" {
		normalized = normalizeCampaignSystemLabel(r.defaultCreateSystem())
	}
	system, ok := r.aliases[normalized]
	return system, ok
}

// workflowRegistry exposes the installed workflow registry to transport wiring
// without leaking alias bookkeeping details.
func (r campaignSystemRegistry) workflowRegistry() campaignworkflow.Registry {
	return r.workflows
}

// normalizeCampaignSystemLabel canonicalizes transport-facing system labels so
// aliases and defaults compare consistently.
func normalizeCampaignSystemLabel(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
