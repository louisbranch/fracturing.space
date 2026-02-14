local scene = Scenario.new("basic_flow")

-- Open a barebones campaign to show a quiet session.
scene:campaign{
  name = "Basic Flow Campaign",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "basics"
}

-- A simple onboarding beat: one PC joins the campaign.
scene:pc("Frodo")

-- The session opens and closes without any action.
scene:start_session("First Session")
scene:end_session()

return scene
