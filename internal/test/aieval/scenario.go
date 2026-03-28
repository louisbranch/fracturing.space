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
		ID:           "ai_gm_campaign_context_bootstrap",
		Label:        "Bootstrap",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureBootstrap",
	},
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
	{
		ID:           "ai_gm_campaign_context_ooc_replace",
		Label:        "OOCReplace",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureOOCReplace",
	},
	{
		ID:           "ai_gm_campaign_context_scene_switch",
		Label:        "SceneSwitch",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureSceneSwitch",
	},
	{
		ID:           "ai_gm_campaign_context_group_action_review_live",
		Label:        "GroupActionReview",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureGroupActionReview",
	},
	{
		ID:           "ai_gm_campaign_context_tag_team_review_live",
		Label:        "TagTeamReview",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureTagTeamReview",
	},
	{
		ID:           "ai_gm_campaign_context_capability_lookup_live",
		Label:        "CapabilityLookup",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureCapabilityLookup",
	},
	{
		ID:           "ai_gm_intent_hope_spend",
		Label:        "IntentHopeSpend",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureIntentHopeSpend",
	},
	{
		ID:           "ai_gm_intent_equipment_action",
		Label:        "IntentEquipmentAction",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureIntentEquipmentAction",
	},
	{
		ID:           "ai_gm_intent_impossible_action",
		Label:        "IntentImpossibleAction",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureIntentImpossibleAction",
	},
	{
		ID:           "ai_gm_intent_ambiguous_action",
		Label:        "IntentAmbiguousAction",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureIntentAmbiguousAction",
	},
	{
		ID:           "ai_gm_intent_domain_card",
		Label:        "IntentDomainCard",
		LiveTestName: "TestAIGMCampaignContextLiveCaptureIntentDomainCard",
	},
	{ID: "ai_gm_redteam_prompt_injection", Label: "RedTeamPromptInjection", LiveTestName: "TestAIGMCampaignContextLiveCaptureRedTeamPromptInjection"},
	{ID: "ai_gm_redteam_jailbreak", Label: "RedTeamJailbreak", LiveTestName: "TestAIGMCampaignContextLiveCaptureRedTeamJailbreak"},
	{ID: "ai_gm_redteam_hallucination", Label: "RedTeamHallucination", LiveTestName: "TestAIGMCampaignContextLiveCaptureRedTeamHallucination"},
	{ID: "ai_gm_redteam_hijacking", Label: "RedTeamHijacking", LiveTestName: "TestAIGMCampaignContextLiveCaptureRedTeamHijacking"},
	{ID: "ai_gm_redteam_overreliance", Label: "RedTeamOverreliance", LiveTestName: "TestAIGMCampaignContextLiveCaptureRedTeamOverreliance"},
	{ID: "ai_gm_redteam_excessive_agency", Label: "RedTeamExcessiveAgency", LiveTestName: "TestAIGMCampaignContextLiveCaptureRedTeamExcessiveAgency"},
	{ID: "ai_gm_multiturn_narrative_continuity", Label: "MultiTurnNarrativeContinuity", LiveTestName: "TestAIGMCampaignContextLiveCaptureMultiTurnNarrativeContinuity"},
	{ID: "ai_gm_multiturn_memory_recall", Label: "MultiTurnMemoryRecall", LiveTestName: "TestAIGMCampaignContextLiveCaptureMultiTurnMemoryRecall"},
	{ID: "ai_gm_multiturn_session_pacing", Label: "MultiTurnSessionPacing", LiveTestName: "TestAIGMCampaignContextLiveCaptureMultiTurnSessionPacing"},
	{ID: "ai_gm_starter_act_progression", Label: "StarterActProgression", LiveTestName: "TestAIGMCampaignContextLiveCaptureStarterActProgression"},
	{ID: "ai_gm_starter_conclusion", Label: "StarterConclusion", LiveTestName: "TestAIGMCampaignContextLiveCaptureStarterConclusion"},
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
