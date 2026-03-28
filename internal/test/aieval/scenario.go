package aieval

import "strings"

// PromptProfile identifies the instruction-profile variant used for one eval.
type PromptProfile string

const (
	// PromptProfileBaseline uses the default repo instruction set.
	PromptProfileBaseline PromptProfile = "baseline"
	// PromptProfileMechanicsHardened uses the eval-only mechanics-focused override.
	PromptProfileMechanicsHardened PromptProfile = "mechanics_hardened"
)

// Scenario identifies one live AI GM evaluation lane and the backing go test entrypoint.
type Scenario struct {
	ID           string
	Label        string
	LiveTestName string
}

var pilotScenarios = []Scenario{
	{
		ID:           "ai_gm_campaign_context_hope_experience",
		Label:        "HopeExperience",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureHopeExperience",
	},
	{
		ID:           "ai_gm_campaign_context_stance_capability",
		Label:        "StanceCapability",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureStanceCapability",
	},
	{
		ID:           "ai_gm_campaign_context_narrator_authority",
		Label:        "NarratorAuthority",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureNarratorAuthority",
	},
	{
		ID:           "ai_gm_campaign_context_subdue_intent_live",
		Label:        "SubdueIntent",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureSubdueIntent",
	},
	{
		ID:           "ai_gm_campaign_context_playbook_attack_review_live",
		Label:        "PlaybookAttackReview",
		LiveTestName: "TestAIGMCampaignContextLiveCapturePlaybookAttackReview",
	},
	{
		ID:           "ai_gm_campaign_context_spotlight_board_review_live",
		Label:        "SpotlightBoardReview",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureSpotlightBoardReview",
	},
}

// PilotScenarios returns the current Promptfoo pilot lane registry in a stable order.
func PilotScenarios() []Scenario {
	out := make([]Scenario, len(pilotScenarios))
	copy(out, pilotScenarios)
	return out
}

// ScenarioByID returns one registered pilot scenario by its stable scenario id.
func ScenarioByID(id string) (Scenario, bool) {
	needle := strings.TrimSpace(id)
	for _, scenario := range pilotScenarios {
		if scenario.ID == needle {
			return scenario, true
		}
	}
	return Scenario{}, false
}
