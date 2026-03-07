local scn = Scenario.new("basic_flow")
local dh = scn:system("DAGGERHEART")

-- Open a barebones campaign to show a quiet session.
scn:campaign{
  name = "Basic Flow Campaign",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "basics"
}

-- A simple onboarding beat: one PC joins the campaign.
scn:pc("Frodo")

-- The session opens and closes without any action.
scn:start_session("First Session")
scn:end_session()

return scn
