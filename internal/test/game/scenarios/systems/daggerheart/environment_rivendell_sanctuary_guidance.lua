local scn = Scenario.new("environment_rivendell_sanctuary_guidance")
local dh = scn:system("DAGGERHEART")

-- Capture divine guidance outcomes from prayer.
scn:campaign{
  name = "Environment Rivendell Sanctuary Guidance",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A prayer triggers a roll for guidance.
scn:start_session("Divine Guidance")

-- Missing DSL: apply outcome-based clarity and Hope gain.
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 13, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
