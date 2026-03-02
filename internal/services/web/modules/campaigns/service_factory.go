package campaigns

import campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"

// newService builds package wiring for this web seam.
func newService(gateway CampaignGateway) campaignapp.Service {
	return newServiceWithWorkflows(gateway, nil)
}

// newServiceWithWorkflows builds package wiring for this web seam.
func newServiceWithWorkflows(gateway CampaignGateway, workflows map[string]CharacterCreationWorkflow) campaignapp.Service {
	return campaignapp.NewServiceWithWorkflows(gateway, mapWorkflowsToApp(workflows))
}

// mapWorkflowsToApp maps values across transport and domain boundaries.
func mapWorkflowsToApp(
	workflows map[string]CharacterCreationWorkflow,
) map[string]campaignapp.CharacterCreationWorkflow {
	if len(workflows) == 0 {
		return nil
	}
	mapped := make(map[string]campaignapp.CharacterCreationWorkflow, len(workflows))
	for system, workflow := range workflows {
		if workflow == nil {
			continue
		}
		mapped[system] = workflow
	}
	if len(mapped) == 0 {
		return nil
	}
	return mapped
}
