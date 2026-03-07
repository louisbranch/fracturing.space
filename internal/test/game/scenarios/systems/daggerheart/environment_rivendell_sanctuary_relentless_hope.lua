local scn = Scenario.new("environment_rivendell_sanctuary_relentless_hope")
local dh = scn:system("DAGGERHEART")

-- Model the once-per-scene stress spend to flip Fear to Hope.
scn:campaign{
  name = "Environment Rivendell Sanctuary Relentless Hope",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A PC marks Stress to turn Fear into Hope once per scene.
scn:start_session("Relentless Hope")

-- Missing DSL: convert a Fear outcome to Hope and mark Stress.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 13, outcome = "failure_fear" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
