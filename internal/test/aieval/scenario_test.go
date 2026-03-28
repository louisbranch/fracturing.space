package aieval

import "testing"

func TestScenarioByID(t *testing.T) {
	scenario, ok := ScenarioByID("ai_gm_campaign_context_hope_experience")
	if !ok {
		t.Fatal("expected hope experience scenario to be registered")
	}
	if scenario.LiveTestName != "TestAIGMCampaignContextLiveCaptureHopeExperience" {
		t.Fatalf("live test name = %q", scenario.LiveTestName)
	}
}

func TestPilotScenariosReturnsCopy(t *testing.T) {
	first := PilotScenarios()
	second := PilotScenarios()
	first[0].ID = "mutated"
	if second[0].ID == "mutated" {
		t.Fatal("expected PilotScenarios to return a defensive copy")
	}
}
