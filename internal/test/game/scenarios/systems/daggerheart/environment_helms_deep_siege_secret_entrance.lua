local scn = Scenario.new("environment_helms_deep_siege_secret_entrance")
local dh = scn:system("DAGGERHEART")

-- Capture the secret entrance discovery roll.
scn:campaign{
  name = "Environment Helms Deep Siege Secret Entrance",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A PC searches for a hidden passage.
scn:start_session("Secret Entrance")

-- Missing DSL: reveal a secret route with Instinct/Knowledge success.
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 17, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
